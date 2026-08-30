package collector

import (
	"testing"

	"github.com/extreajp/demo-sysmon/internal/procfs"
)

func TestCPUUsage(t *testing.T) {
	c := &CPU{FS: procfs.New(testdataProc(t), "")}
	first, err := c.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if first[0].Name != "cpu.usage_percent" {
		t.Fatalf("%+v", first)
	}
	// second collect with same fixture → 0% (no delta)
	second, err := c.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if second[0].Value != 0 {
		t.Fatalf("expected 0, got %v", second[0].Value)
	}
}

func TestParseCPUStat(t *testing.T) {
	s, err := parseCPUStat("cpu  10 0 10 80 0 0 0 0\n")
	if err != nil {
		t.Fatal(err)
	}
	if s.user != 10 || s.idle != 80 {
		t.Fatalf("%+v", s)
	}
}
