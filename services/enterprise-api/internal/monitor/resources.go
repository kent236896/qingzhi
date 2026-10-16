package monitor

import (
	"context"
	"os"
	"path/filepath"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// ReadResources samples CPU, memory, and the disk that holds the working directory.
// The standard library does not expose portable CPU and disk usage.
func ReadResources(ctx context.Context) (Resources, error) {
	usage, err := disk.UsageWithContext(ctx, diskRoot())
	if err != nil {
		return Resources{}, err
	}
	memory, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Resources{}, err
	}
	percents, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return Resources{}, err
	}
	cpuPercent := 0.0
	if len(percents) > 0 {
		cpuPercent = round1(percents[0])
	}
	mount := usage.Path
	if mount == "" {
		mount = diskRoot()
	}
	return Resources{
		Available:     true,
		CPUPercent:    cpuPercent,
		MemoryUsed:    memory.Used,
		MemoryTotal:   memory.Total,
		MemoryPercent: round1(memory.UsedPercent),
		DiskUsed:      usage.Used,
		DiskTotal:     usage.Total,
		DiskPercent:   round1(usage.UsedPercent),
		DiskMount:     mount,
	}, nil
}

func diskRoot() string {
	wd, err := os.Getwd()
	if err != nil || wd == "" {
		return "/"
	}
	volume := filepath.VolumeName(wd)
	if volume == "" {
		return "/"
	}
	return volume + string(filepath.Separator)
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
