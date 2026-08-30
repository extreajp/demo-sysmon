package collector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/extreajp/demo-sysmon/internal/config"
	"github.com/extreajp/demo-sysmon/internal/procfs"
)

func TestCgroupPSIReresolvesAfterPathMove(t *testing.T) {
	sys := t.TempDir()
	oldDir := filepath.Join(sys, "fs/cgroup/sysmon-demo-web")
	writeCPUPressure(t, oldDir, "12.50")

	c := &CgroupPSI{
		FS:      procfs.New("", sys),
		Cgroups: []config.CgroupRef{{Name: "web", Container: "sysmon-demo-web"}},
	}
	if got := avg10(t, c); got != 12.5 {
		t.Fatalf("first collect avg10=%v", got)
	}

	if err := os.RemoveAll(oldDir); err != nil {
		t.Fatal(err)
	}
	newDir := filepath.Join(sys, "fs/cgroup/system.slice/sysmon-demo-web")
	writeCPUPressure(t, newDir, "87.00")

	if got := avg10(t, c); got != 87 {
		t.Fatalf("after move avg10=%v want 87", got)
	}
}

func writeCPUPressure(t *testing.T, dir, avg10 string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := "some avg10=" + avg10 + " avg60=4.00 avg300=1.20 total=12345\nfull avg10=2.00 avg60=0.80 avg300=0.30 total=100\n"
	if err := os.WriteFile(filepath.Join(dir, "cpu.pressure"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func avg10(t *testing.T, c *CgroupPSI) float64 {
	t.Helper()
	samples, err := c.Collect()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range samples {
		if s.Name == "cgroup.web.psi.cpu.some.avg10" {
			return s.Value
		}
	}
	t.Fatalf("missing cgroup.web.psi.cpu.some.avg10 in %#v", samples)
	return 0
}
