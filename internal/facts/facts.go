// Package facts reads what a server is and how hard it is working.
package facts

import (
	"context"
	"strconv"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

type Facts struct {
	Hostname         string  `json:"hostname"`
	IPAddress        string  `json:"ipAddress"`
	OperatingSystem  string  `json:"operatingSystem"`
	Kernel           string  `json:"kernel"`
	CPUCount         int     `json:"cpuCount"`
	CPUUsage         float64 `json:"cpuUsage"`
	MemoryTotalBytes int64   `json:"memoryTotalBytes"`
	MemoryUsedBytes  int64   `json:"memoryUsedBytes"`
	DiskTotalBytes   int64   `json:"diskTotalBytes"`
	DiskUsedBytes    int64   `json:"diskUsedBytes"`
	UptimeMs         int64   `json:"uptimeMs"`
}

const (
	cmdOS       = `. /etc/os-release 2>/dev/null; printf '%s' "$PRETTY_NAME"`
	cmdKernel   = `uname -r`
	cmdCPUs     = `nproc`
	cmdMemory   = `free -b | awk '/^Mem:/{print $2" "$3}'`
	cmdDisk     = `df -B1 --output=size,used / | tail -1`
	cmdUptime   = `cut -d' ' -f1 /proc/uptime`
	cmdLoad     = `cut -d' ' -f1 /proc/loadavg`
	cmdIP       = `hostname -I 2>/dev/null | awk '{print $1}'`
	cmdHostname = `hostname -f 2>/dev/null || hostname`
)

// Gather probes the machine. Individual probes are allowed to fail — a missing
// value is better than refusing to show a server at all — so only a transport
// failure is returned as an error.
func Gather(ctx context.Context, s *steps.Session) (Facts, error) {
	var f Facts

	read := func(cmd string) string {
		out, err := s.Capture(ctx, cmd)
		if err != nil {
			return ""
		}
		return out
	}

	f.Hostname = read(cmdHostname)
	f.IPAddress = read(cmdIP)
	f.OperatingSystem = read(cmdOS)
	f.Kernel = read(cmdKernel)
	f.CPUCount = atoi(read(cmdCPUs))

	if total, used, ok := twoInts(read(cmdMemory)); ok {
		f.MemoryTotalBytes, f.MemoryUsedBytes = total, used
	}
	if total, used, ok := twoInts(read(cmdDisk)); ok {
		f.DiskTotalBytes, f.DiskUsedBytes = total, used
	}

	if seconds, err := strconv.ParseFloat(read(cmdUptime), 64); err == nil {
		f.UptimeMs = int64(seconds * 1000)
	}

	// Load average over core count approximates "how busy is this box" well
	// enough for a dashboard, and costs one cheap read.
	if load, err := strconv.ParseFloat(read(cmdLoad), 64); err == nil && f.CPUCount > 0 {
		f.CPUUsage = clamp(load/float64(f.CPUCount), 0, 1)
	}

	return f, nil
}

func twoInts(s string) (int64, int64, bool) {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return 0, 0, false
	}
	first, err1 := strconv.ParseInt(fields[0], 10, 64)
	second, err2 := strconv.ParseInt(fields[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return first, second, true
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
