package collector

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/extreajp/demo-sysmon/internal/config"
	"github.com/extreajp/demo-sysmon/internal/metric"
	"github.com/extreajp/demo-sysmon/internal/procfs"
)

type CgroupPSI struct {
	FS      procfs.FS
	Cgroups []config.CgroupRef
	resolved map[string]string
}

func (c *CgroupPSI) Collect() ([]metric.Sample, error) {
	if c.resolved == nil {
		c.resolved = map[string]string{}
	}
	root := filepath.Join(c.FS.Sys, "fs/cgroup")
	var out []metric.Sample
	for _, ref := range c.Cgroups {
		dir := c.resolved[ref.Name]
		if dir == "" {
			dir = resolveCgroup(root, ref.Container)
			if dir != "" {
				c.resolved[ref.Name] = dir
			}
		}
		if dir == "" {
			continue
		}
		for _, res := range []string{"cpu", "memory", "io"} {
			raw, err := os.ReadFile(filepath.Join(dir, res+".pressure"))
			if err != nil {
				continue
			}
			lines, err := ParsePSI(string(raw))
			if err != nil {
				continue
			}
			out = append(out, psiSamples("cgroup."+ref.Name+".psi", res, lines)...)
		}
	}
	return out, nil
}

func resolveCgroup(root, container string) string {
	if container == "" {
		return ""
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if !strings.Contains(d.Name(), container) {
			return nil
		}
		if _, err := os.Stat(filepath.Join(path, "cpu.pressure")); err == nil {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found != "" {
		return found
	}
	return resolveViaDocker(container)
}

func resolveViaDocker(container string) string {
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", container).Output()
	if err != nil {
		return ""
	}
	pid := strings.TrimSpace(string(out))
	if pid == "" || pid == "0" {
		return ""
	}
	b, err := os.ReadFile("/proc/" + pid + "/cgroup")
	if err != nil {
		return ""
	}
	// cgroup v2: "0::/system.slice/docker-....scope"
	for _, line := range strings.Split(string(b), "\n") {
		_, path, ok := strings.Cut(line, "::")
		if !ok {
			continue
		}
		dir := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(path, "/"))
		if _, err := os.Stat(filepath.Join(dir, "cpu.pressure")); err == nil {
			return dir
		}
	}
	return ""
}
