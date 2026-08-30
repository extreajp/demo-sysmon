package collector

import (
	"errors"
	"strings"

	"github.com/extreajp/demo-sysmon/internal/metric"
	"github.com/extreajp/demo-sysmon/internal/procfs"
)

var errInvalidLoadavg = errors.New("invalid loadavg")

type LoadAvg struct {
	FS procfs.FS
}

func (l *LoadAvg) ncpu() float64 {
	raw, err := l.FS.ProcFile("stat")
	if err != nil {
		return 1
	}
	n := 0
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "cpu") && len(line) > 3 && line[3] >= '0' && line[3] <= '9' {
			n++
		}
	}
	if n == 0 {
		return 1
	}
	return float64(n)
}

func (l *LoadAvg) Collect() ([]metric.Sample, error) {
	raw, err := l.FS.ProcFile("loadavg")
	if err != nil {
		return nil, err
	}
	f := strings.Fields(raw)
	if len(f) < 3 {
		return nil, errInvalidLoadavg
	}
	a1, _ := procfs.ParseFloat(f[0])
	a5, _ := procfs.ParseFloat(f[1])
	a15, _ := procfs.ParseFloat(f[2])
	n := l.ncpu()
	return []metric.Sample{
		{Name: "loadavg.1", Value: a1},
		{Name: "loadavg.5", Value: a5},
		{Name: "loadavg.15", Value: a15},
		{Name: "loadavg.per_core.1", Value: a1 / n},
	}, nil
}
