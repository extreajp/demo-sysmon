package collector

import (
	"path/filepath"
	"testing"

	"github.com/extreajp/demo-sysmon/internal/procfs"
)

func TestParsePSI(t *testing.T) {
	raw := "some avg10=12.50 avg60=4.00 avg300=1.20 total=12345\nfull avg10=2.00 avg60=0.80 avg300=0.30 total=100\n"
	lines, err := ParsePSI(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
	if lines[0].Kind != "some" || lines[0].Avg10 != 12.5 {
		t.Fatalf("some: %+v", lines[0])
	}
	if lines[1].Kind != "full" || lines[1].Avg10 != 2 {
		t.Fatalf("full: %+v", lines[1])
	}
}

func TestHostPSICollect(t *testing.T) {
	root := testdataProc(t)
	h := &HostPSI{FS: procfs.New(root, "")}
	samples, err := h.Collect()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for _, s := range samples {
		got[s.Name] = s.Value
	}
	if got["host.psi.cpu.some.avg10"] != 12.5 {
		t.Fatalf("avg10=%v samples=%v", got["host.psi.cpu.some.avg10"], got)
	}
	if got["host.psi.cpu.some.stall_percent"] != 12.5 {
		t.Fatalf("stall=%v", got["host.psi.cpu.some.stall_percent"])
	}
}

func testdataProc(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../testdata/proc")
	if err != nil {
		t.Fatal(err)
	}
	return p
}
