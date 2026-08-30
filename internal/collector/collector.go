package collector

import (
	"time"

	"github.com/extreajp/demo-sysmon/internal/config"
	"github.com/extreajp/demo-sysmon/internal/metric"
	"github.com/extreajp/demo-sysmon/internal/procfs"
)

type Collector interface {
	Collect() ([]metric.Sample, error)
}

type Set struct {
	items []Collector
}

func New(fs procfs.FS, cfg config.Config) *Set {
	return &Set{items: []Collector{
		&CPU{FS: fs},
		&Memory{FS: fs},
		&Disk{FS: fs, Mount: "/"},
		&LoadAvg{FS: fs},
		&HostPSI{FS: fs},
		&CgroupPSI{FS: fs, Cgroups: cfg.Cgroups},
	}}
}

func (s *Set) Snapshot() (metric.Snapshot, error) {
	snap := metric.Snapshot{Timestamp: time.Now().UTC()}
	for _, c := range s.items {
		samples, err := c.Collect()
		if err != nil {
			continue
		}
		snap.Samples = append(snap.Samples, samples...)
	}
	return snap, nil
}
