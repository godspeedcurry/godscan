package utils

import (
	"sync"
)

type HostErrorStat struct {
	Count  int
	Sample string
}

var (
	hostErrorMu   sync.Mutex
	hostErrors    = map[string]HostErrorStat{}
	hostErrorOnce sync.Map
)

func recordHostError(host, sample string) {
	hostErrorMu.Lock()
	defer hostErrorMu.Unlock()
	stat := hostErrors[host]
	stat.Count++
	if stat.Sample == "" {
		stat.Sample = sample
	}
	hostErrors[host] = stat
}

func CollectHostErrorStats(reset bool) map[string]HostErrorStat {
	hostErrorMu.Lock()
	defer hostErrorMu.Unlock()
	out := make(map[string]HostErrorStat, len(hostErrors))
	for k, v := range hostErrors {
		out[k] = v
	}
	if reset {
		hostErrors = map[string]HostErrorStat{}
		hostErrorOnce = sync.Map{}
	}
	return out
}
