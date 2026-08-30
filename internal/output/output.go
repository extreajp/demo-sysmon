package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/extreajp/demo-sysmon/internal/metric"
)

func JSON(w io.Writer, snap metric.Snapshot) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(snap)
}

func JSONL(w io.Writer, snap metric.Snapshot) error {
	enc := json.NewEncoder(w)
	return enc.Encode(snap)
}

func Table(w io.Writer, snap metric.Snapshot) error {
	fmt.Fprintf(w, "timestamp  %s\n", snap.Timestamp.Format("15:04:05"))
	samples := append([]metric.Sample(nil), snap.Samples...)
	sort.Slice(samples, func(i, j int) bool { return samples[i].Name < samples[j].Name })
	for _, s := range samples {
		lbl := ""
		if len(s.Labels) > 0 {
			var parts []string
			for k, v := range s.Labels {
				parts = append(parts, k+"="+v)
			}
			sort.Strings(parts)
			lbl = " {" + strings.Join(parts, ",") + "}"
		}
		fmt.Fprintf(w, "%-40s %10.2f%s\n", s.Name, s.Value, lbl)
	}
	return nil
}

func Prometheus(w io.Writer, snap metric.Snapshot) error {
	fmt.Fprintf(w, "# sysmon snapshot %s\n", snap.Timestamp.UTC().Format("2006-01-02T15:04:05Z"))
	for _, s := range snap.Samples {
		name := promName(s.Name)
		if len(s.Labels) == 0 {
			fmt.Fprintf(w, "%s %g\n", name, s.Value)
			continue
		}
		var parts []string
		for k, v := range s.Labels {
			parts = append(parts, fmt.Sprintf(`%s="%s"`, promName(k), v))
		}
		sort.Strings(parts)
		fmt.Fprintf(w, "%s{%s} %g\n", name, strings.Join(parts, ","), s.Value)
	}
	return nil
}

func promName(s string) string {
	b := strings.Builder{}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
