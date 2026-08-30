package collector

import (
	"fmt"
	"strings"

	"github.com/extreajp/demo-sysmon/internal/metric"
	"github.com/extreajp/demo-sysmon/internal/procfs"
)

type cpuStat struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func (s cpuStat) total() uint64 {
	return s.user + s.nice + s.system + s.idle + s.iowait + s.irq + s.softirq + s.steal
}

func (s cpuStat) idleAll() uint64 {
	return s.idle + s.iowait
}

type CPU struct {
	FS   procfs.FS
	prev *cpuStat
}

func parseCPUStat(content string) (cpuStat, error) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 8 {
			return cpuStat{}, fmt.Errorf("cpu line: %q", line)
		}
		var s cpuStat
		nums := []*uint64{&s.user, &s.nice, &s.system, &s.idle, &s.iowait, &s.irq, &s.softirq, &s.steal}
		for i, p := range nums {
			v, err := procfs.ParseUint(f[i+1])
			if err != nil {
				return cpuStat{}, err
			}
			*p = v
		}
		return s, nil
	}
	return cpuStat{}, fmt.Errorf("no cpu line")
}

func (c *CPU) Collect() ([]metric.Sample, error) {
	raw, err := c.FS.ProcFile("stat")
	if err != nil {
		return nil, err
	}
	cur, err := parseCPUStat(raw)
	if err != nil {
		return nil, err
	}
	if c.prev == nil {
		c.prev = &cur
		return []metric.Sample{{Name: "cpu.usage_percent", Value: 0}}, nil
	}
	dt := float64(cur.total() - c.prev.total())
	usage := 0.0
	if dt > 0 {
		didle := float64(cur.idleAll() - c.prev.idleAll())
		usage = (1 - didle/dt) * 100
	}
	c.prev = &cur
	return []metric.Sample{{Name: "cpu.usage_percent", Value: usage}}, nil
}
