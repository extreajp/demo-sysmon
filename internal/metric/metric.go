package metric

import "time"

type Sample struct {
	Name   string            `json:"name"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
}

type Snapshot struct {
	Timestamp time.Time `json:"timestamp"`
	Samples   []Sample  `json:"samples"`
}

func (s Snapshot) Find(name string, labels map[string]string) (Sample, bool) {
	for _, sm := range s.Samples {
		if sm.Name != name {
			continue
		}
		if labelsMatch(sm.Labels, labels) {
			return sm, true
		}
	}
	return Sample{}, false
}

func labelsMatch(have, want map[string]string) bool {
	if len(want) == 0 {
		return true
	}
	for k, v := range want {
		if have[k] != v {
			return false
		}
	}
	return true
}
