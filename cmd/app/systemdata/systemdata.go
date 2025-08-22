package systemdata

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"log/slog"
	"runtime"
	"time"
)

type (
	SystemData struct {
		Alloc         uint64  `json:"alloc"`
		SystemAlloc   uint64  `json:"systemAlloc"`
		NumGoRoutines int     `json:"numGoRoutines"`
		NumCPUs       int     `json:"numCPUs"`
		CPUPercent    float64 `json:"CPUPercent"`
	}
)

var (
	systemData = SystemData{}
	stopChan   = make(chan struct{})
)

func GetSystemData() SystemData {
	return systemData
}

func StartSystemDataUpdates() {
	slog.Debug("Starting system data updates")
	go func() {
		updateSystemData()
		oneMinTicker := time.NewTicker(time.Minute)
		defer func() {
			oneMinTicker.Stop()
		}()
		for {
			select {
			case <-oneMinTicker.C:
				updateSystemData()
			case <-stopChan:
				slog.Info("Stopping system data updates")
				return
			}
		}
	}()
}

func StopSystemDataUpdates() {
	select {
	case stopChan <- struct{}{}:
	default:
	}
}

func updateSystemData() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	cpuPerc, err := cpu.Percent(time.Duration(0), false)
	if err != nil || len(cpuPerc) == 0 {
		slog.Warn("cpu percent unavailable", slog.String("error", fmt.Sprintf("%v", err)))
	}
	systemData.Alloc = m.Alloc
	systemData.SystemAlloc = m.Sys
	systemData.NumGoRoutines = runtime.NumGoroutine()
	systemData.NumCPUs = runtime.NumCPU()
	if len(cpuPerc) > 0 {
		systemData.CPUPercent = cpuPerc[0]
	}
	slog.Debug("system data updated",
		slog.Uint64("alloc", systemData.Alloc),
		slog.Uint64("sysAlloc", systemData.SystemAlloc),
		slog.Int("goroutines", systemData.NumGoRoutines),
		slog.Int("cpus", systemData.NumCPUs),
		slog.Float64("cpu%", systemData.CPUPercent),
	)
}
