package manager

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"bandwidth-hub/internal/config"
)

type ProxyManager struct {
	mu sync.RWMutex
}

func NewProxyManager() *ProxyManager {
	return &ProxyManager{}
}

// AddProxy thêm một proxy mới vào Kho Proxy Chung
func (pm *ProxyManager) AddProxy(proxyURL string, country string) (*config.ProxyItem, error) {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("định dạng URL proxy không hợp lệ: %w", err)
	}

	scheme := parsed.Scheme
	if scheme != "http" && scheme != "https" && scheme != "socks5" {
		return nil, fmt.Errorf("giao thức không được hỗ trợ (chỉ chấp nhận http, https, socks5)")
	}

	host, portStr, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		host = parsed.Host
		if scheme == "socks5" {
			portStr = "1080"
		} else {
			portStr = "8080"
		}
	}
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	if country == "" {
		country = "GLOBAL"
	}

	id := fmt.Sprintf("prx_%d", time.Now().UnixNano()%100000)
	item := &config.ProxyItem{
		ID:          id,
		URL:         proxyURL,
		Protocol:    scheme,
		Host:        host,
		Port:        port,
		Country:     country,
		Status:      "untested",
		LatencyMs:   0,
		AssignedTo:  []string{},
		LastChecked: time.Now().Format(time.RFC3339),
	}

	config.GlobalConfig.Proxies = append(config.GlobalConfig.Proxies, item)
	_ = config.SaveConfig()

	// Test ngay proxy ở background
	go pm.TestProxy(item.ID)

	return item, nil
}

// DeleteProxy xoá proxy khỏi kho
func (pm *ProxyManager) DeleteProxy(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	idx := -1
	for i, p := range config.GlobalConfig.Proxies {
		if p.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("không tìm thấy proxy id: %s", id)
	}

	// Bỏ gán khỏi các services
	for _, svc := range config.GlobalConfig.Services {
		var filtered []string
		for _, assignedID := range svc.AssignedProxies {
			if assignedID != id {
				filtered = append(filtered, assignedID)
			}
		}
		svc.AssignedProxies = filtered
	}

	config.GlobalConfig.Proxies = append(config.GlobalConfig.Proxies[:idx], config.GlobalConfig.Proxies[idx+1:]...)
	return config.SaveConfig()
}

// TestProxy kiểm tra độ sống và ping của proxy
func (pm *ProxyManager) TestProxy(id string) (*config.ProxyItem, error) {
	pm.mu.Lock()
	var target *config.ProxyItem
	for _, p := range config.GlobalConfig.Proxies {
		if p.ID == id {
			target = p
			break
		}
	}
	pm.mu.Unlock()

	if target == nil {
		return nil, fmt.Errorf("không tìm thấy proxy: %s", id)
	}

	proxyParsed, err := url.Parse(target.URL)
	if err != nil {
		target.Status = "dead"
		target.LastChecked = time.Now().Format(time.RFC3339)
		_ = config.SaveConfig()
		return target, err
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyParsed),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/ip", nil)
	resp, err := client.Do(req)
	latency := time.Since(startTime).Milliseconds()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if err != nil || resp.StatusCode != http.StatusOK {
		// Thử fallback tới endpoint khác nhẹ hơn
		req2, _ := http.NewRequestWithContext(ctx, "GET", "http://api.ipify.org", nil)
		resp2, err2 := client.Do(req2)
		if err2 != nil || resp2.StatusCode != http.StatusOK {
			target.Status = "dead"
			target.LatencyMs = 9999
		} else {
			target.Status = "alive"
			target.LatencyMs = time.Since(startTime).Milliseconds()
			resp2.Body.Close()
		}
	} else {
		target.Status = "alive"
		target.LatencyMs = latency
		resp.Body.Close()
	}

	target.LastChecked = time.Now().Format(time.RFC3339)
	_ = config.SaveConfig()

	return target, nil
}

// TestAllProxies kiểm tra tất cả proxy đồng thời
func (pm *ProxyManager) TestAllProxies() {
	var wg sync.WaitGroup
	for _, p := range config.GlobalConfig.Proxies {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			pm.TestProxy(pid)
		}(p.ID)
	}
	wg.Wait()
}
