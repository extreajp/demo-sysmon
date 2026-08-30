package collector

import (
	"strings"

	"github.com/extreajp/demo-sysmon/internal/metric"
	"github.com/extreajp/demo-sysmon/internal/procfs"
)

type Memory struct {
	FS procfs.FS
}

func (m *Memory) Collect() ([]metric.Sample, error) {
	raw, err := m.FS.ProcFile("meminfo")
	if err != nil {
		return nil, err
	}
	kv := map[string]uint64{}
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		key := strings.TrimSuffix(f[0], ":")
		v, err := procfs.ParseUint(f[1])
		if err != nil {
			continue
		}
		kv[key] = v
	}
	total := float64(kv["MemTotal"])
	avail := float64(kv["MemAvailable"])
	if avail == 0 {
		avail = float64(kv["MemFree"] + kv["Buffers"] + kv["Cached"])
	}
	usedPct := 0.0
	if total > 0 {
		usedPct = (1 - avail/total) * 100
	}
	return []metric.Sample{
		{Name: "memory.usage_percent", Value: usedPct},
		{Name: "memory.available_kb", Value: avail},
		{Name: "memory.swap_total_kb", Value: float64(kv["SwapTotal"])},
		{Name: "memory.swap_free_kb", Value: float64(kv["SwapFree"])},
		{Name: "memory.swap_used_kb", Value: float64(kv["SwapTotal"] - kv["SwapFree"])},
	}, nil
}
