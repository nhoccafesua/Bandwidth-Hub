package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bandwidth-hub/internal/api"
	"bandwidth-hub/internal/config"
	"bandwidth-hub/internal/manager"
	"bandwidth-hub/internal/metrics"
)

//go:embed web/*
var webFS embed.FS

func main() {
	portFlag := flag.Int("port", 8888, "Cổng web server (mặc định 8888)")
	configFlag := flag.String("config", "config.json", "Đường dẫn file cấu hình json")
	flag.Parse()

	log.Println("====================================================")
	log.Println("   BANDWIDTH HUB - LOW-SPEC VPS MANAGER (v1.0)     ")
	log.Println("   Hỗ trợ: Pawns, Honeygain, EarnApp, Traffmonetizer,")
	log.Println("           Wipter, Repocket & Multi-Proxy IP Pool   ")
	log.Println("====================================================")

	// 1. Tải hoặc khởi tạo cấu hình
	cfg, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Lỗi nạp cấu hình: %v", err)
	}
	if *portFlag != 8888 {
		cfg.ServerPort = *portFlag
	}

	log.Printf("[+] Khởi tạo cấu hình từ: %s", *configFlag)
	log.Printf("[+] Cổng lắng nghe: %d", cfg.ServerPort)

	// 2. Khởi tạo Process Manager và Proxy Manager
	procMgr := manager.NewProcessManager()
	proxyMgr := manager.NewProxyManager()

	// 3. Khởi động goroutine dọn RAM tự động định kỳ nếu bật
	if cfg.AutoCleanRAM {
		interval := time.Duration(cfg.AutoCleanIntervalMin) * time.Minute
		if interval < 5*time.Minute {
			interval = 30 * time.Minute
		}
		log.Printf("[+] Tự động dọn RAM/Drop Caches được bật (chu kỳ: %v)", interval)
		go startAutoRAMCleaner(interval)
	}

	// 4. Đăng ký REST API và Static Frontend
	mux := http.NewServeMux()
	apiHandler := api.NewAPIHandler(procMgr, proxyMgr)
	apiHandler.RegisterRoutes(mux)

	// Phục vụ frontend từ Go embed FS hoặc thư mục đĩa
	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Lỗi nạp thư mục web embed: %v", err)
	}
	fileServer := http.FileServer(http.FS(subFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Tránh ghi đè các API endpoints
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			data, err := subFS.Open("index.html")
			if err == nil {
				defer data.Close()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.ServeContent(w, r, "index.html", time.Now(), data.(fs.ReadSeeker))
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})

	serverAddr := fmt.Sprintf("0.0.0.0:%d", cfg.ServerPort)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Bắt tín hiệu dừng Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[*] Bandwidth Hub Dashboard đang chạy tại: http://%s", serverAddr)
		log.Println("[*] Nhấn Ctrl+C để dừng hệ thống.")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Lỗi lắng nghe HTTP server: %v", err)
		}
	}()

	<-stop
	log.Println("\n[*] Đang tắt Bandwidth Hub an toàn...")
	procMgr.StopAll()
	_ = config.SaveConfig()
	log.Println("[✓] Toàn bộ dịch vụ đã được dừng sạch sẽ. Tạm biệt!")
}

func startAutoRAMCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		msg, err := metrics.CleanRAM()
		if err != nil {
			log.Printf("[AutoRAM] Cảnh báo dọn RAM: %v", err)
		} else {
			log.Printf("[AutoRAM] %s", msg)
		}
	}
}
