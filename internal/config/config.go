package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Interval string       `json:"interval"`
	Cgroups  []CgroupRef  `json:"cgroups"`
	Serve    Serve        `json:"serve"`
	Alerts   []AlertRule  `json:"alerts"`
}

type CgroupRef struct {
	Name      string `json:"name"`
	Container string `json:"container"`
}

type Serve struct {
	Listen     string `json:"listen"`
	CORSOrigin string `json:"cors_origin"`
}

type AlertRule struct {
	Name      string            `json:"name"`
	Metric    string            `json:"metric"`
	Labels    map[string]string `json:"labels"`
	Op        string            `json:"op"`
	Threshold float64           `json:"threshold"`
	For       string            `json:"for"`
	Severity  string            `json:"severity"`
}

func Load(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.Interval == "" {
		c.Interval = "1s"
	}
	if c.Serve.Listen == "" {
		c.Serve.Listen = "127.0.0.1:9101"
	}
	if _, err := time.ParseDuration(c.Interval); err != nil {
		return c, fmt.Errorf("interval: %w", err)
	}
	return c, nil
}

func (c Config) IntervalDuration() time.Duration {
	d, err := time.ParseDuration(c.Interval)
	if err != nil {
		return time.Second
	}
	return d
}
