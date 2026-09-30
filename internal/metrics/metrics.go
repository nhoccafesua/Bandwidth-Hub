package metrics

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SystemMetrics chứa số liệu tài nguyên máy chủ
type SystemMetrics struct {
	TotalRAMMB       float64 `json:"total_ram_mb"`
	UsedRAMMB        float64 `json:"used_ram_mb"`
	FreeRAMMB        float64 `json:"free_ram_mb"`
	AvailableRAMMB   float64 `json:"available_ram_mb"`
	RAMUsagePercent  float64 `json:"ram_usage_percent"`
	TotalSwapMB      float64 `json:"total_swap_mb"`
	UsedSwapMB       float64 `json:"used_swap_mb"`
	SwapUsagePercent float64 `json:"swap_usage_percent"`
	CPUUsagePercent  float64 `json:"cpu_usage_percent"`
	WorkerRAMTotalMB float64 `json:"worker_ram_total_mb"`
	TotalBandwidthGB float64 `json:"total_bandwidth_gb"`
	UptimeString     string  `json:"uptime_string"`
	LoadAverage      string  `json:"load_average"`
	OS               string  `json:"os"`
	NumCPU           int     `json:"num_cpu"`
	LastCleaned      string  `json:"last_cleaned"`
}

var (
	prevIdleTime uint64
	prevTotalTime uint64
	cpuMutex      sync.Mutex
	lastCleanTime = "Never"
)

// GetSystemMetrics đọc số liệu hệ thống trực tiếp từ /proc trên Linux
func GetSystemMetrics(workerRAM float64, totalSharedGB float64) SystemMetrics {
	m := SystemMetrics{
		OS:               runtime.GOOS,
		NumCPU:           runtime.NumCPU(),
		WorkerRAMTotalMB: workerRAM,
		TotalBandwidthGB: totalSharedGB,
		LastCleaned:      lastCleanTime,
	}

	if runtime.GOOS == "linux" {
		readLinuxMemInfo(&m)
		readLinuxCPU(&m)
		readLinuxUptime(&m)
	} else {
		// Mock data chuẩn cho môi trường non-linux (phục vụ dev & preview)
		var rtm runtime.MemStats
		runtime.ReadMemStats(&rtm)
		m.TotalRAMMB = 1024.0
		m.UsedRAMMB = float64(rtm.Alloc)/1024/1024 + 185.0
		m.FreeRAMMB = m.TotalRAMMB - m.UsedRAMMB
		m.AvailableRAMMB = m.FreeRAMMB
		m.RAMUsagePercent = (m.UsedRAMMB / m.TotalRAMMB) * 100
		m.TotalSwapMB = 2048.0
		m.UsedSwapMB = 48.0
		m.SwapUsagePercent = (m.UsedSwapMB / m.TotalSwapMB) * 100
		m.CPUUsagePercent = 2.4
		m.UptimeString = "14d 6h 32m"
		m.LoadAverage = "0.12, 0.08, 0.05"
	}

	return m
}

// readLinuxMemInfo đọc chi tiết bộ nhớ từ /proc/meminfo
func readLinuxMemInfo(m *SystemMetrics) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer file.Close()

	var memTotal, memFree, memAvailable, buffers, cached uint64
	var swapTotal, swapFree uint64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val, _ := strconv.ParseUint(parts[1], 10, 64)

		switch key {
		case "MemTotal":
			memTotal = val
		case "MemFree":
			memFree = val
		case "MemAvailable":
			memAvailable = val
		case "Buffers":
			buffers = val
		case "Cached":
			cached = val
		case "SwapTotal":
			swapTotal = val
		case "SwapFree":
			swapFree = val
		}
	}

	if memTotal > 0 {
		m.TotalRAMMB = float64(memTotal) / 1024.0
		if memAvailable > 0 {
			m.AvailableRAMMB = float64(memAvailable) / 1024.0
			m.UsedRAMMB = m.TotalRAMMB - m.AvailableRAMMB
		} else {
			// Fallback cho Linux kernel cũ thiếu MemAvailable
			used := memTotal - (memFree + buffers + cached)
			m.UsedRAMMB = float64(used) / 1024.0
			m.AvailableRAMMB = float64(memFree+buffers+cached) / 1024.0
		}
		m.FreeRAMMB = float64(memFree) / 1024.0
		m.RAMUsagePercent = (m.UsedRAMMB / m.TotalRAMMB) * 100.0
	}

	if swapTotal > 0 {
		m.TotalSwapMB = float64(swapTotal) / 1024.0
		swapUsed := swapTotal - swapFree
		m.UsedSwapMB = float64(swapUsed) / 1024.0
		m.SwapUsagePercent = (m.UsedSwapMB / m.TotalSwapMB) * 100.0
	}
}

// readLinuxCPU tính % CPU bằng cách đọc delta từ /proc/stat
func readLinuxCPU(m *SystemMetrics) {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	file, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 4 && fields[0] == "cpu" {
			var total, idle uint64
			for i := 1; i < len(fields); i++ {
				val, _ := strconv.ParseUint(fields[i], 10, 64)
				total += val
				if i == 4 { // idle
					idle = val
				}
			}

			if prevTotalTime > 0 && total > prevTotalTime {
				totalDiff := float64(total - prevTotalTime)
				idleDiff := float64(idle - prevIdleTime)
				m.CPUUsagePercent = ((totalDiff - idleDiff) / totalDiff) * 100.0
			} else {
				m.CPUUsagePercent = 1.0
			}

			prevTotalTime = total
			prevIdleTime = idle
		}
	}
}

// readLinuxUptime đọc uptime và load average
func readLinuxUptime(m *SystemMetrics) {
	// Uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if secs, err := strconv.ParseFloat(fields[0], 64); err == nil {
				d := time.Duration(secs) * time.Second
				days := int(d.Hours()) / 24
				hours := int(d.Hours()) % 24
				mins := int(d.Minutes()) % 60
				m.UptimeString = fmt.Sprintf("%dd %dh %dm", days, hours, mins)
			}
		}
	}

	// Load Average
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			m.LoadAverage = fmt.Sprintf("%s, %s, %s", fields[0], fields[1], fields[2])
		}
	}
}

// CleanRAM giải phóng buffer/cache hệ thống và kích hoạt Go GC
func CleanRAM() (string, error) {
	// 1. Gọi Garbage Collection của Go runtime
	runtime.GC()

	msg := "Đã chạy Garbage Collector nội bộ."

	if runtime.GOOS == "linux" {
		// Gọi sync và drop_caches
		cmd := exec.Command("sh", "-c", "sync && echo 3 > /proc/sys/vm/drop_caches")
		if out, err := cmd.CombinedOutput(); err != nil {
			// Thử qua sudo nếu không có quyền root trực tiếp
			sudoCmd := exec.Command("sudo", "sh", "-c", "sync && echo 3 > /proc/sys/vm/drop_caches")
			if sudoOut, sudoErr := sudoCmd.CombinedOutput(); sudoErr != nil {
				return msg, fmt.Errorf("không thể dọn drop_caches (cần quyền root): %v - %s", sudoErr, string(sudoOut))
			}
		} else {
			_ = out
		}
		msg = "Đã dọn dẹp RAM thành công! (Drop Caches: PageCache, dentries & inodes + Go GC)"
	}

	lastCleanTime = time.Now().Format("15:04:05 (02/01)")
	return msg, nil
}
