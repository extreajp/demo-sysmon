package collector

import (
	"fmt"
	"strings"

	"github.com/extreajp/demo-sysmon/internal/metric"
	"github.com/extreajp/demo-sysmon/internal/procfs"
)

type PSILine struct {
	Kind  string
	Avg10 float64
	Avg60 float64
	Avg300 float64
	Total uint64
}

func ParsePSI(content string) ([]PSILine, error) {
	var out []PSILine
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return nil, fmt.Errorf("psi line: %q", line)
		}
		pl := PSILine{Kind: fields[0]}
		for _, f := range fields[1:] {
			k, v, ok := strings.Cut(f, "=")
			if !ok {
				continue
			}
			switch k {
			case "avg10":
				pl.Avg10, _ = procfs.ParseFloat(v)
			case "avg60":
				pl.Avg60, _ = procfs.ParseFloat(v)
			case "avg300":
				pl.Avg300, _ = procfs.ParseFloat(v)
			case "total":
				pl.Total, _ = procfs.ParseUint(v)
			}
		}
		out = append(out, pl)
	}
	return out, nil
}

func psiSamples(prefix, resource string, lines []PSILine) []metric.Sample {
	var out []metric.Sample
	for _, l := range lines {
		base := prefix + "." + resource + "." + l.Kind
		out = append(out,
			metric.Sample{Name: base + ".avg10", Value: l.Avg10},
			metric.Sample{Name: base + ".avg60", Value: l.Avg60},
			metric.Sample{Name: base + ".avg300", Value: l.Avg300},
		)
		if l.Kind == "some" {
			out = append(out, metric.Sample{Name: base + ".stall_percent", Value: l.Avg10})
		}
	}
	return out
}

type HostPSI struct {
	FS procfs.FS
}

func (h *HostPSI) Collect() ([]metric.Sample, error) {
	var out []metric.Sample
	for _, res := range []string{"cpu", "memory", "io"} {
		raw, err := h.FS.ProcFile("pressure/" + res)
		if err != nil {
			continue
		}
		lines, err := ParsePSI(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, psiSamples("host.psi", res, lines)...)
	}
	return out, nil
}
