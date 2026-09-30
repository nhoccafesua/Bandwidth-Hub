package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// ServiceConfig đại diện cho cấu hình và trạng thái của một nền tảng chia sẻ băng thông
type ServiceConfig struct {
	ID              string   `json:"id"`               // pawns, honeygain, earnapp, traffmonetizer, wipter, repocket
	Name            string   `json:"name"`             // Tên hiển thị (vd: Pawns.app)
	Category        string   `json:"category"`         // Loại nền tảng
	Enabled         bool     `json:"enabled"`          // Kích hoạt tự chạy
	Status          string   `json:"status"`           // ready, running, stopped, error
	StatusMessage   string   `json:"status_message"`   // Chi tiết lỗi hoặc trạng thái
	Email           string   `json:"email,omitempty"`
	Password        string   `json:"password,omitempty"`
	Token           string   `json:"token,omitempty"`  // App Token / Device UUID / API Key
	DeviceName      string   `json:"device_name"`      // Tên thiết bị hiển thị trên dashboard app
	NodeTarget      int      `json:"node_target"`      // Số lượng node mong muốn chạy (default: 1)
	ActiveNodes     int      `json:"active_nodes"`     // Số lượng node đang chạy thực tế
	MemoryLimitMB   int      `json:"memory_limit_mb"`  // Giới hạn RAM tối đa cho service (vd: 120MB)
	AssignedProxies []string `json:"assigned_proxies"` // Danh sách proxy ID hoặc URL được gán
	Earnings        float64  `json:"earnings"`         // Thu nhập tích lũy ($ USD)
	BandwidthGB     float64  `json:"bandwidth_gb"`     // Tổng dung lượng chia sẻ (GB)
	RAMUsageMB      float64  `json:"ram_usage_mb"`     // Lượng RAM thực tế đang dùng (MB)
	RevenueSharePct float64  `json:"revenue_share_pct"`// Tỉ lệ % đóng góp doanh thu
	ContainerImage  string   `json:"container_image"`  // Docker Image chính thức nếu chạy container
	BinaryPath      string   `json:"binary_path"`      // Đường dẫn file thực thi native nếu chạy CLI
	LastUpdated     string   `json:"last_updated"`
}

// ProxyItem đại diện cho một proxy trong Kho Proxy Chung
type ProxyItem struct {
	ID          string   `json:"id"`           // UUID hoặc ID dạng prx_1
	URL         string   `json:"url"`          // socks5://user:pass@host:port hoặc http://host:port
	Protocol    string   `json:"protocol"`     // http, socks5
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Country     string   `json:"country"`      // Mã quốc gia (VN, US, SG, ...)
	Status      string   `json:"status"`       // alive, dead, untested
	LatencyMs   int64    `json:"latency_ms"`   // Độ trễ ping (ms)
	AssignedTo  []string `json:"assigned_to"`  // Service IDs đang dùng proxy này
	LastChecked string   `json:"last_checked"`
}

// AppConfig cấu hình chung của toàn bộ Bandwidth Hub
type AppConfig struct {
	ServerPort           int                       `json:"server_port"`
	AuthUsername         string                    `json:"auth_username"`
	AuthPassword         string                    `json:"auth_password"`
	AutoCleanRAM         bool                      `json:"auto_clean_ram"`
	AutoCleanIntervalMin int                       `json:"auto_clean_interval_min"`
	MemoryAlertThreshold int                       `json:"memory_alert_threshold_pct"` // Ngưỡng cảnh báo RAM (%)
	Services             map[string]*ServiceConfig `json:"services"`
	Proxies              []*ProxyItem              `json:"proxies"`
}

var (
	GlobalConfig *AppConfig
	cfgMutex     sync.RWMutex
	ConfigPath   = "config.json"
)

// DefaultConfig trả về cấu hình mặc định sẵn sàng cho 6 nền tảng lớn
func DefaultConfig() *AppConfig {
	return &AppConfig{
		ServerPort:           8888,
		AuthUsername:         "admin",
		AuthPassword:         "admin123",
		AutoCleanRAM:         true,
		AutoCleanIntervalMin: 60,
		MemoryAlertThreshold: 85,
		Services: map[string]*ServiceConfig{
			"pawns": {
				ID:              "pawns",
				Name:            "Pawns.app",
				Category:        "Bandwidth & Surveys",
				Enabled:         true,
				Status:          "ready",
				StatusMessage:   "Ready to start",
				Email:           "",
				Password:        "",
				DeviceName:      "vps-node-01",
				NodeTarget:      1,
				ActiveNodes:     0,
				MemoryLimitMB:   100,
				AssignedProxies: []string{},
				Earnings:        3.45,
				BandwidthGB:     28.6,
				RAMUsageMB:      0.0,
				RevenueSharePct: 28.5,
				ContainerImage:  "iproyal/pawns-cli:latest",
				BinaryPath:      "/usr/local/bin/pawns-cli",
				LastUpdated:     time.Now().Format(time.RFC3339),
			},
			"honeygain": {
				ID:              "honeygain",
				Name:            "Honeygain",
				Category:        "Residential Bandwidth",
				Enabled:         true,
				Status:          "ready",
				StatusMessage:   "Ready to start",
				Email:           "",
				Password:        "",
				DeviceName:      "hg-vps-01",
				NodeTarget:      1,
				ActiveNodes:     0,
				MemoryLimitMB:   120,
				AssignedProxies: []string{},
				Earnings:        4.80,
				BandwidthGB:     45.2,
				RAMUsageMB:      0.0,
				RevenueSharePct: 39.7,
				ContainerImage:  "honeygain/honeygain:latest",
				BinaryPath:      "/usr/local/bin/honeygain",
				LastUpdated:     time.Now().Format(time.RFC3339),
			},
			"earnapp": {
				ID:              "earnapp",
				Name:            "EarnApp",
				Category:        "Bright Data Network",
				Enabled:         true,
				Status:          "ready",
				StatusMessage:   "Ready to start",
				Token:           "",
				DeviceName:      "earnapp-node-01",
				NodeTarget:      1,
				ActiveNodes:     0,
				MemoryLimitMB:   80,
				AssignedProxies: []string{},
				Earnings:        1.90,
				BandwidthGB:     19.4,
				RAMUsageMB:      0.0,
				RevenueSharePct: 15.7,
				ContainerImage:  "fazalfarhan01/earnapp:lite",
				BinaryPath:      "/usr/bin/earnapp",
				LastUpdated:     time.Now().Format(time.RFC3339),
			},
			"traffmonetizer": {
				ID:              "traffmonetizer",
				Name:            "Traffmonetizer",
				Category:        "Distributed Bandwidth",
				Enabled:         true,
				Status:          "ready",
				StatusMessage:   "Ready to start",
				Token:           "",
				DeviceName:      "tm-vps-node",
				NodeTarget:      1,
				ActiveNodes:     0,
				MemoryLimitMB:   90,
				AssignedProxies: []string{},
				Earnings:        1.25,
				BandwidthGB:     14.8,
				RAMUsageMB:      0.0,
				RevenueSharePct: 10.3,
				ContainerImage:  "traffmonetizer/cli_v2:latest",
				BinaryPath:      "/usr/local/bin/traffmonetizer",
				LastUpdated:     time.Now().Format(time.RFC3339),
			},
			"wipter": {
				ID:              "wipter",
				Name:            "Wipter",
				Category:        "P2P Bandwidth Node",
				Enabled:         false,
				Status:          "stopped",
				StatusMessage:   "Stopped by user",
				Token:           "",
				DeviceName:      "wipter-vps",
				NodeTarget:      1,
				ActiveNodes:     0,
				MemoryLimitMB:   80,
				AssignedProxies: []string{},
				Earnings:        0.42,
				BandwidthGB:     5.1,
				RAMUsageMB:      0.0,
				RevenueSharePct: 3.5,
				ContainerImage:  "wipter/client:latest",
				BinaryPath:      "/usr/local/bin/wipter",
				LastUpdated:     time.Now().Format(time.RFC3339),
			},
			"repocket": {
				ID:              "repocket",
				Name:            "Repocket",
				Category:        "Passive Proxy Share",
				Enabled:         false,
				Status:          "stopped",
				StatusMessage:   "Stopped by user",
				Email:           "",
				Token:           "",
				DeviceName:      "repocket-vps",
				NodeTarget:      1,
				ActiveNodes:     0,
				MemoryLimitMB:   90,
				AssignedProxies: []string{},
				Earnings:        0.28,
				BandwidthGB:     3.9,
				RAMUsageMB:      0.0,
				RevenueSharePct: 2.3,
				ContainerImage:  "repocket/repocket:latest",
				BinaryPath:      "/usr/local/bin/repocket",
				LastUpdated:     time.Now().Format(time.RFC3339),
			},
		},
		Proxies: []*ProxyItem{
			{
				ID:          "prx_1",
				URL:         "socks5://192.168.1.100:1080",
				Protocol:    "socks5",
				Host:        "192.168.1.100",
				Port:        1080,
				Country:     "VN",
				Status:      "alive",
				LatencyMs:   45,
				AssignedTo:  []string{"pawns"},
				LastChecked: time.Now().Format(time.RFC3339),
			},
			{
				ID:          "prx_2",
				URL:         "http://usr_proxy:pass123@103.145.2.4:8080",
				Protocol:    "http",
				Host:        "103.145.2.4",
				Port:        8080,
				Country:     "SG",
				Status:      "alive",
				LatencyMs:   88,
				AssignedTo:  []string{"honeygain"},
				LastChecked: time.Now().Format(time.RFC3339),
			},
		},
	}
}

// LoadConfig tải cấu hình từ file json, tạo mới nếu chưa có
func LoadConfig(path string) (*AppConfig, error) {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	ConfigPath = path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		GlobalConfig = DefaultConfig()
		if err := saveConfigUnlocked(path, GlobalConfig); err != nil {
			return nil, fmt.Errorf("không thể tạo file cấu hình mẫu: %w", err)
		}
		return GlobalConfig, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc file cấu hình: %w", err)
	}

	cfg := &AppConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("lỗi cú pháp JSON file config: %w", err)
	}

	// Đảm bảo các field mặc định tồn tại nếu file cũ thiếu
	defaultCfg := DefaultConfig()
	if cfg.Services == nil {
		cfg.Services = defaultCfg.Services
	} else {
		for k, v := range defaultCfg.Services {
			if _, exists := cfg.Services[k]; !exists {
				cfg.Services[k] = v
			}
		}
	}
	if cfg.Proxies == nil {
		cfg.Proxies = []*ProxyItem{}
	}

	GlobalConfig = cfg
	return GlobalConfig, nil
}

// SaveConfig lưu cấu hình hiện tại xuống đĩa an toàn
func SaveConfig() error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	return saveConfigUnlocked(ConfigPath, GlobalConfig)
}

func saveConfigUnlocked(path string, cfg *AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// BackupConfig xuất bản sao lưu cấu hình kèm timestamp
func BackupConfig() (string, []byte, error) {
	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	data, err := json.MarshalIndent(GlobalConfig, "", "  ")
	if err != nil {
		return "", nil, err
	}
	filename := fmt.Sprintf("bandwidth-hub-backup-%s.json", time.Now().Format("20060102-150405"))
	return filename, data, nil
}
