package alert

import (
	"time"

	"github.com/extreajp/demo-sysmon/internal/config"
	"github.com/extreajp/demo-sysmon/internal/metric"
)

type State struct {
	Name      string  `json:"name"`
	Severity  string  `json:"severity"`
	Metric    string  `json:"metric"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Firing    bool    `json:"firing"`
	Since     string  `json:"since,omitempty"`
}

type Engine struct {
	rules []config.AlertRule
	since map[string]time.Time
}

func New(rules []config.AlertRule) *Engine {
	return &Engine{rules: rules, since: map[string]time.Time{}}
}

func (e *Engine) Evaluate(snap metric.Snapshot) []State {
	now := snap.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	out := make([]State, 0, len(e.rules))
	for _, r := range e.rules {
		st := State{Name: r.Name, Severity: r.Severity, Metric: r.Metric, Threshold: r.Threshold}
		sm, ok := snap.Find(r.Metric, r.Labels)
		if ok {
			st.Value = sm.Value
		}
		hold := parseFor(r.For)
		if ok && cmp(r.Op, sm.Value, r.Threshold) {
			if e.since[r.Name].IsZero() {
				e.since[r.Name] = now
			}
			st.Since = e.since[r.Name].Format(time.RFC3339)
			if now.Sub(e.since[r.Name]) >= hold {
				st.Firing = true
			}
		} else {
			delete(e.since, r.Name)
		}
		out = append(out, st)
	}
	return out
}

func FiringCount(states []State) int {
	n := 0
	for _, s := range states {
		if s.Firing {
			n++
		}
	}
	return n
}

func cmp(op string, v, th float64) bool {
	switch op {
	case ">":
		return v > th
	case ">=":
		return v >= th
	case "<":
		return v < th
	case "<=":
		return v <= th
	default:
		return v > th
	}
}

func parseFor(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}
