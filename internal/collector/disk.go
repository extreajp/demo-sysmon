package collector

import (
	"strings"

	"github.com/extreajp/demo-sysmon/internal/metric"
	"github.com/extreajp/demo-sysmon/internal/procfs"
	"golang.org/x/sys/unix"
)

type Disk struct {
	FS    procfs.FS
	Mount string
}

func (d *Disk) Collect() ([]metric.Sample, error) {
	mount := d.Mount
	if mount == "" {
		mount = "/"
	}
	var st unix.Statfs_t
	if err := unix.Statfs(mount, &st); err != nil {
		return nil, err
	}
	total := float64(st.Blocks) * float64(st.Bsize)
	free := float64(st.Bavail) * float64(st.Bsize)
	usedPct := 0.0
	if total > 0 {
		usedPct = (1 - free/total) * 100
	}
	out := []metric.Sample{
		{Name: "disk.usage_percent", Value: usedPct, Labels: map[string]string{"mount": mount}},
	}
	raw, err := d.FS.ProcFile("diskstats")
	if err != nil {
		return out, nil
	}
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Fields(line)
		if len(f) < 14 {
			continue
		}
		name := f[2]
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		rsect, _ := procfs.ParseFloat(f[5])
		wsect, _ := procfs.ParseFloat(f[9])
		lbl := map[string]string{"device": name}
		out = append(out,
			metric.Sample{Name: "disk.read_sectors", Value: rsect, Labels: lbl},
			metric.Sample{Name: "disk.write_sectors", Value: wsect, Labels: lbl},
		)
	}
	return out, nil
}
