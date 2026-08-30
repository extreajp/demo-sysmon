package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/extreajp/demo-sysmon/internal/alert"
	"github.com/extreajp/demo-sysmon/internal/metric"
	"github.com/extreajp/demo-sysmon/internal/output"
)

type snapshotter interface {
	Snapshot() (metric.Snapshot, error)
}

type Server struct {
	Collect  snapshotter
	Alerts   *alert.Engine
	CORS     string
	Interval time.Duration

	mu     sync.Mutex
	last   payload
	lastAt time.Time
}

type payload struct {
	Snapshot any           `json:"snapshot"`
	Alerts   []alert.State `json:"alerts"`
	Firing   int           `json:"firing"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/snapshot", s.snapshot)
	mux.HandleFunc("/api/stream", s.stream)
	mux.HandleFunc("/metrics", s.metrics)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.cors(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) cors(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	allow := s.allowed(origin)
	if allow != "" {
		w.Header().Set("Access-Control-Allow-Origin", allow)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
}

func (s *Server) allowed(origin string) string {
	if s.CORS == "" {
		return origin
	}
	for _, a := range strings.Split(s.CORS, ",") {
		a = strings.TrimSpace(a)
		if a == origin || a == "*" {
			return a
		}
	}
	// first configured origin as default for non-browser clients
	if origin == "" {
		return strings.TrimSpace(strings.Split(s.CORS, ",")[0])
	}
	return ""
}

func (s *Server) collect() (payload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	interval := s.Interval
	if interval <= 0 {
		interval = time.Second
	}
	if !s.lastAt.IsZero() && time.Since(s.lastAt) < interval {
		return s.last, nil
	}
	snap, err := s.Collect.Snapshot()
	if err != nil {
		return payload{}, err
	}
	states := s.Alerts.Evaluate(snap)
	p := payload{Snapshot: snap, Alerts: states, Firing: alert.FiringCount(states)}
	s.last = p
	s.lastAt = time.Now()
	return p, nil
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) snapshot(w http.ResponseWriter, _ *http.Request) {
	p, err := s.collect()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	interval := s.Interval
	if interval <= 0 {
		interval = time.Second
	}
	tick := time.NewTicker(interval)
	defer tick.Stop()
	send := func() bool {
		p, err := s.collect()
		if err != nil {
			return false
		}
		b, err := json.Marshal(p)
		if err != nil {
			return false
		}
		_, _ = fmtSSE(w, "metrics", b)
		fl.Flush()
		return true
	}
	if !send() {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			if !send() {
				return
			}
		}
	}
}

func fmtSSE(w http.ResponseWriter, event string, data []byte) (int, error) {
	return w.Write([]byte("event: " + event + "\ndata: " + string(data) + "\n\n"))
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	p, err := s.collect()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	snap, ok := p.Snapshot.(metric.Snapshot)
	if !ok {
		http.Error(w, "no snapshot", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_ = output.Prometheus(w, snap)
}
