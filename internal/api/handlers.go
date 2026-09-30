package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bandwidth-hub/internal/config"
	"bandwidth-hub/internal/manager"
	"bandwidth-hub/internal/metrics"
)

type APIHandler struct {
	procMgr  *manager.ProcessManager
	proxyMgr *manager.ProxyManager
}

func NewAPIHandler(pm *manager.ProcessManager, pxm *manager.ProxyManager) *APIHandler {
	return &APIHandler{
		procMgr:  pm,
		proxyMgr: pxm,
	}
}

// JSONResponse hàm tiện ích trả về JSON
func JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// RegisterRoutes đăng ký toàn bộ endpoints
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/status", h.handleGetStatus)
	mux.HandleFunc("/api/system/clean-ram", h.handleCleanRAM)
	mux.HandleFunc("/api/services/run-all", h.handleRunAll)
	mux.HandleFunc("/api/services/stop-all", h.handleStopAll)
	mux.HandleFunc("/api/services/", h.handleServiceActions)
	mux.HandleFunc("/api/proxies", h.handleProxies)
	mux.HandleFunc("/api/proxies/", h.handleProxyActions)
	mux.HandleFunc("/api/config/backup", h.handleBackupConfig)
	mux.HandleFunc("/api/config/restore", h.handleRestoreConfig)
}

// handleGetStatus trả về toàn bộ dashboard metrics
func (h *APIHandler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var totalWorkerRAM float64
	var totalBandwidthGB float64
	var totalEarnings float64
	var totalActiveNodes int
	var totalTargetNodes int

	for _, svc := range config.GlobalConfig.Services {
		totalWorkerRAM += svc.RAMUsageMB
		totalBandwidthGB += svc.BandwidthGB
		totalEarnings += svc.Earnings
		totalActiveNodes += svc.ActiveNodes
		totalTargetNodes += svc.NodeTarget
	}

	sysMetrics := metrics.GetSystemMetrics(totalWorkerRAM, totalBandwidthGB)

	response := map[string]interface{}{
		"system":               sysMetrics,
		"total_earnings_usd":   fmt.Sprintf("%.2f", totalEarnings),
		"total_active_nodes":   totalActiveNodes,
		"total_target_nodes":   totalTargetNodes,
		"total_bandwidth_gb":   fmt.Sprintf("%.1f", totalBandwidthGB),
		"total_worker_ram_mb":  fmt.Sprintf("%.1f", totalWorkerRAM),
		"services":             config.GlobalConfig.Services,
		"proxies":              config.GlobalConfig.Proxies,
	}

	JSONResponse(w, http.StatusOK, response)
}

// handleCleanRAM thực thi lệnh giải phóng bộ nhớ
func (h *APIHandler) handleCleanRAM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	msg, err := metrics.CleanRAM()
	if err != nil {
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": msg,
	})
}

// handleRunAll khởi động toàn bộ services
func (h *APIHandler) handleRunAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	results := h.procMgr.RunAll()
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"results": results,
	})
}

// handleStopAll dừng toàn bộ services
func (h *APIHandler) handleStopAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	h.procMgr.StopAll()
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Đã dừng toàn bộ dịch vụ",
	})
}

// handleServiceActions điều phối start, stop, restart, config, logs cho service
func (h *APIHandler) handleServiceActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/services/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Service ID missing", http.StatusBadRequest)
		return
	}

	serviceID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "start":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := h.procMgr.StartService(serviceID); err != nil {
			JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		JSONResponse(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Đã khởi động dịch vụ"})

	case "stop":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := h.procMgr.StopService(serviceID); err != nil {
			JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		JSONResponse(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Đã dừng dịch vụ"})

	case "restart":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := h.procMgr.RestartService(serviceID); err != nil {
			JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		JSONResponse(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Đã khởi động lại dịch vụ"})

	case "config":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleUpdateServiceConfig(w, r, serviceID)

	case "logs":
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		lines := h.procMgr.GetLogs(serviceID)
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"service_id": serviceID,
			"logs":       lines,
		})

	default:
		http.Error(w, "Action không hợp lệ", http.StatusNotFound)
	}
}

// handleUpdateServiceConfig lưu cấu hình tài khoản & thông số của service
func (h *APIHandler) handleUpdateServiceConfig(w http.ResponseWriter, r *http.Request, serviceID string) {
	svc, exists := config.GlobalConfig.Services[serviceID]
	if !exists {
		JSONResponse(w, http.StatusNotFound, map[string]interface{}{"success": false, "message": "Dịch vụ không tồn tại"})
		return
	}

	var payload struct {
		Enabled         *bool    `json:"enabled"`
		Email           *string  `json:"email"`
		Password        *string  `json:"password"`
		Token           *string  `json:"token"`
		DeviceName      *string  `json:"device_name"`
		NodeTarget      *int     `json:"node_target"`
		MemoryLimitMB   *int     `json:"memory_limit_mb"`
		AssignedProxies []string `json:"assigned_proxies"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Dữ liệu JSON không hợp lệ"})
		return
	}

	if payload.Enabled != nil {
		svc.Enabled = *payload.Enabled
	}
	if payload.Email != nil {
		svc.Email = strings.TrimSpace(*payload.Email)
	}
	if payload.Password != nil && *payload.Password != "" {
		svc.Password = strings.TrimSpace(*payload.Password)
	}
	if payload.Token != nil {
		svc.Token = strings.TrimSpace(*payload.Token)
	}
	if payload.DeviceName != nil {
		svc.DeviceName = strings.TrimSpace(*payload.DeviceName)
	}
	if payload.NodeTarget != nil && *payload.NodeTarget > 0 {
		svc.NodeTarget = *payload.NodeTarget
	}
	if payload.MemoryLimitMB != nil && *payload.MemoryLimitMB > 20 {
		svc.MemoryLimitMB = *payload.MemoryLimitMB
	}
	if payload.AssignedProxies != nil {
		svc.AssignedProxies = payload.AssignedProxies
	}

	if err := config.SaveConfig(); err != nil {
		JSONResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Cấu hình %s đã được lưu thành công", svc.Name),
		"service": svc,
	})
}

// handleProxies xử lý xem danh sách hoặc thêm proxy mới
func (h *APIHandler) handleProxies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		JSONResponse(w, http.StatusOK, config.GlobalConfig.Proxies)
	case http.MethodPost:
		var req struct {
			URL     string `json:"url"`
			Country string `json:"country"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
			JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "URL proxy không được để trống"})
			return
		}
		item, err := h.proxyMgr.AddProxy(req.URL, req.Country)
		if err != nil {
			JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Đã thêm proxy vào kho chung",
			"proxy":   item,
		})
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleProxyActions xử lý test hoặc xóa proxy
func (h *APIHandler) handleProxyActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/proxies/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Proxy ID missing", http.StatusBadRequest)
		return
	}

	if parts[0] == "test-all" && r.Method == http.MethodPost {
		go h.proxyMgr.TestAllProxies()
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Đang tiến hành kiểm tra toàn bộ proxy...",
		})
		return
	}

	proxyID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if r.Method == http.MethodDelete || action == "delete" {
		if err := h.proxyMgr.DeleteProxy(proxyID); err != nil {
			JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		JSONResponse(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Đã xoá proxy khỏi kho"})
		return
	}

	if action == "test" && r.Method == http.MethodPost {
		item, err := h.proxyMgr.TestProxy(proxyID)
		if err != nil {
			JSONResponse(w, http.StatusOK, map[string]interface{}{"success": false, "message": err.Error(), "proxy": item})
			return
		}
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Proxy phản hồi tốt! Ping: %dms", item.LatencyMs),
			"proxy":   item,
		})
		return
	}

	http.Error(w, "Endpoint không hợp lệ", http.StatusNotFound)
}

// handleBackupConfig xuất file backup json
func (h *APIHandler) handleBackupConfig(w http.ResponseWriter, r *http.Request) {
	filename, data, err := config.BackupConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleRestoreConfig nạp lại cấu hình từ file json tải lên
func (h *APIHandler) handleRestoreConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "Không thể đọc dữ liệu tải lên"})
		return
	}

	var newCfg config.AppConfig
	if err := json.Unmarshal(data, &newCfg); err != nil {
		JSONResponse(w, http.StatusBadRequest, map[string]interface{}{"success": false, "message": "File JSON sao lưu không hợp lệ: " + err.Error()})
		return
	}

	config.GlobalConfig = &newCfg
	if err := config.SaveConfig(); err != nil {
		JSONResponse(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "message": err.Error()})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Khôi phục cấu hình thành công!",
	})
}
