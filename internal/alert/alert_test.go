package alert

import (
	"testing"
	"time"

	"github.com/extreajp/demo-sysmon/internal/config"
	"github.com/extreajp/demo-sysmon/internal/metric"
)

func TestEvaluateFor(t *testing.T) {
	e := New([]config.AlertRule{{
		Name: "hot", Metric: "cpu.usage_percent", Op: ">", Threshold: 50, For: "2s", Severity: "warning",
	}})
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	snap := func(ts time.Time, v float64) metric.Snapshot {
		return metric.Snapshot{Timestamp: ts, Samples: []metric.Sample{{Name: "cpu.usage_percent", Value: v}}}
	}
	s1 := e.Evaluate(snap(t0, 80))
	if s1[0].Firing {
		t.Fatal("should not fire immediately")
	}
	s2 := e.Evaluate(snap(t0.Add(2*time.Second), 80))
	if !s2[0].Firing {
		t.Fatal("should fire after for=")
	}
	s3 := e.Evaluate(snap(t0.Add(3*time.Second), 10))
	if s3[0].Firing {
		t.Fatal("should clear")
	}
}
