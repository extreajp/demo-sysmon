package httpapi

import (
	"testing"
	"time"

	"github.com/extreajp/demo-sysmon/internal/alert"
	"github.com/extreajp/demo-sysmon/internal/metric"
)

type stubSnap struct {
	n    int
	snap metric.Snapshot
}

func (s *stubSnap) Snapshot() (metric.Snapshot, error) {
	s.n++
	return s.snap, nil
}

func TestCollectReusesWithinInterval(t *testing.T) {
	st := &stubSnap{snap: metric.Snapshot{
		Samples: []metric.Sample{{Name: "cpu.usage_percent", Value: 42}},
	}}
	s := &Server{Collect: st, Alerts: alert.New(nil), Interval: time.Second}
	if _, err := s.collect(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.collect(); err != nil {
		t.Fatal(err)
	}
	if st.n != 1 {
		t.Fatalf("Snapshot called %d, want 1", st.n)
	}
}

func TestCollectRefreshAfterInterval(t *testing.T) {
	st := &stubSnap{snap: metric.Snapshot{}}
	s := &Server{Collect: st, Alerts: alert.New(nil), Interval: time.Millisecond}
	if _, err := s.collect(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Millisecond)
	if _, err := s.collect(); err != nil {
		t.Fatal(err)
	}
	if st.n != 2 {
		t.Fatalf("Snapshot called %d, want 2", st.n)
	}
}
