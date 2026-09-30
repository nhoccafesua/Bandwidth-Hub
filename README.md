# 🚀 Bandwidth Hub - Lightweight Bandwidth Sharing Manager & Dashboard

> **Công cụ Quản trị & Dashboard Tập trung Tối ưu cho Linux VPS Cấu hình Thấp (1-2 GB RAM, 1 vCPU)**  
> Tích hợp 6 nền tảng kiếm tiền băng thông thụ động lớn nhất thế giới, kèm hệ thống **Kho Proxy Chung (Multi-Proxy Pool)** và cơ chế **Auto Memory-Trimming** giữ mức tiêu thụ toàn hệ thống dưới 500 MB.

---

## 📑 Mục lục
1. [Tổng quan & Đặc điểm nổi bật](#-tổng-quan--đặc-điểm-nổi-bật)
2. [Kiến trúc hệ thống](#-kiến-trúc-hệ-thống)
3. [6 Nền tảng được tích hợp](#-6-nền-tảng-được-tích-hợp)
4. [Chiến lược tối ưu RAM cho VPS 1GB](#-chiến-lược-tối-ưu-ram-cho-vps-1gb)
5. [Cài đặt nhanh 1-Click (Ubuntu / Debian)](#-cài-đặt-nhanh-1-click-ubuntu--debian)
6. [Tùy chọn Khởi chạy: Go Binary vs Python](#-tùy-chọn-khởi-chạy-go-binary-vs-python)
7. [Hướng dẫn sử dụng Giao diện Dashboard](#-hướng-dẫn-sử-dụng-giao-diện-dashboard)
8. [Quản trị Kho Proxy Chung (Multi-Proxy Pool)](#-quản-trị-kho-proxy-chung-multi-proxy-pool)
9. [Lệnh Quản trị Systemd & Dọn dẹp RAM](#-lệnh-quản-trị-systemd--dọn-dẹp-ram)

---

## 🌟 Tổng quan & Đặc điểm nổi bật

Các nền tảng chia sẻ băng thông (Bandwidth Sharing) thường gặp vấn đề lớn khi chạy đồng thời nhiều node trên VPS giá rẻ: rò rỉ bộ nhớ (memory leaks), cache Linux tích tụ lâu ngày làm cạn RAM và bị **Linux OOM Killer** đột ngột hạ gục tiến trình.

**Bandwidth Hub** được thiết kế từ gốc để giải quyết triệt để vấn đề này:
- ⚡ **Siêu nhẹ**: Bộ điều khiển core chỉ tốn **12 - 25 MB RAM**, khởi động trong **0.05 giây**.
- 📊 **Dashboard Dark Neon**: Giao diện Datacenter hiện đại, hiển thị thời gian thực (Real-time Web UI, không dùng React/Next cồng kềnh, chỉ thuần HTML5 + Tailwind CSS + Vanilla JS).
- 🧹 **Tự động dọn RAM (Drop Caches)**: Tích hợp trigger giải phóng PageCache, Dentries, Inodes tự động qua cron hoặc chỉ với 1 click.
- 🛡️ **Kho Proxy Chung (Shared Proxy Pool)**: Cho phép nạp danh sách HTTP/SOCKS5 proxy, kiểm tra độ trễ (ping ms) và phân bổ linh hoạt vào từng node riêng biệt.
- 🔄 **Quản lý vòng đời tiến trình**: Hỗ trợ khởi động đồng loạt (`Run All`), dừng tất cả (`Stop All`), kiểm tra trạng thái từng service và xem log stdout/stderr trực tiếp trên Web.

---

## 📐 Kiến trúc hệ thống

```mermaid
flowchart TD
    subgraph WEB_DASHBOARD["🖥️ Web Dashboard (Dark Neon UI)"]
        UI_HEADER["Header Actions: Run All / Stop All / Clean RAM / Backup"]
        UI_KPIS["4 Global Cards: Earnings ($) / Active Nodes / Traffic (GB) / Worker RAM"]
        UI_REV["Revenue Contribution Progress Bar (%)"]
        UI_CARDS["6 Modular Cards: Pawns, Honeygain, EarnApp, Traffmonetizer, Wipter, Repocket"]
        UI_PROXY["Kho Proxy Chung (Table & Latency Tester)"]
    end

    subgraph CORE_ENGINE["⚙️ Core Manager & Backend (Go / Python)"]
        PROC_MGR["Process Lifecycle Manager (Native CLI / Docker Headless)"]
        PROXY_MGR["Proxy Pool & Ping Tester (SOCKS5 / HTTP)"]
        METRICS_COLLECTOR["Direct /proc Metrics Engine (Meminfo, Stat, Loadavg)"]
        REST_API["RESTful API Endpoints (/api/status, /api/services, ...)"]
        CACHE_TRIMMER["Memory Trimmer: sync & drop_caches"]
    end

    subgraph LINUX_HOST["🐧 Low-Spec Linux VPS (1-2 GB RAM, 1 vCPU)"]
        SWAP["2GB Swapfile (ZRAM/Fallocate Protection)"]
        SYSCTL["Kernel Tunings: swappiness=10, somaxconn=2048"]
        WORKER_NODES["Background Workers (cgroups RAM limited: 80-120MB each)"]
    end

    WEB_DASHBOARD <-->|HTTP REST / SSE / JSON| REST_API
    REST_API --> PROC_MGR
    REST_API --> PROXY_MGR
    METRICS_COLLECTOR -->|Đọc trực tiếp| LINUX_HOST
    CACHE_TRIMMER -->|echo 3 > /proc/sys/vm/drop_caches| LINUX_HOST
    PROC_MGR -->|Khởi chạy & Giới hạn RAM| WORKER_NODES
```

---

## 📦 6 Nền tảng được tích hợp

| # | Nền tảng | Danh mục | Phương thức xác thực | Giới hạn RAM khuyến nghị |
|---|---|---|---|---|
| 1 | **Pawns.app** (IPRoyal) | Bandwidth & Surveys | Email + Mật khẩu | 100 MB |
| 2 | **Honeygain** | Residential Bandwidth | Email + Mật khẩu | 120 MB |
| 3 | **EarnApp** | Bright Data Network | UUID Token / OAuth | 80 MB |
| 4 | **Traffmonetizer** | Distributed Bandwidth | Token API Key | 90 MB |
| 5 | **Wipter** | P2P Bandwidth Node | API Token | 80 MB |
| 6 | **Repocket** | Passive Proxy Share | Email + API Key | 90 MB |

---

## 🧠 Chiến lược tối ưu RAM cho VPS 1GB

Để chạy mượt mà 6 modules cùng lúc trên VPS chỉ có **1 GB RAM**:
1. **Thiết lập 2GB Swapfile với Swappiness = 10**:
   - Chỉ swap khi RAM thực tế chạm ngưỡng > 90%.
   - Tránh hiện tượng giật lag I/O ổ cứng.
2. **Cgroups / Memory Limit cho từng worker**:
   - Mỗi tiến trình worker bị khóa cứng trần bộ nhớ (MemoryMax: 80MB - 120MB). Không một module nào có thể phình to làm tràn RAM của VPS.
3. **Automated Cache Dropping (`trim_memory.sh`)**:
   - Khi lưu lượng băng thông lớn chạy qua VPS, Linux kernel liên tục lưu đệm gói tin vào `PageCache` và `Inodes`.
   - Hệ thống tự động kích hoạt `echo 3 > /proc/sys/vm/drop_caches` mỗi 4 tiếng hoặc khi RAM > 80%, thu hồi ngay 150 - 350 MB RAM nhàn rỗi.

---

## ⚡ Cài đặt nhanh 1-Click

Chỉ cần đăng nhập SSH vào VPS của bạn và chạy lệnh duy nhất tùy theo hệ điều hành:

### Dành cho Ubuntu / Debian
```bash
# Tải và chạy script cài đặt tự động toàn diện
curl -sSL https://raw.githubusercontent.com/your-repo/bandwidth-hub/main/scripts/install.sh -o install.sh && sudo bash install.sh
```

### Dành cho AlmaLinux 10 / RockyLinux / RHEL 9 (Minimal)
Phiên bản AlmaLinux Minimal thường sử dụng `dnf` và `firewalld`. Script chuyên dụng sẽ tự động mở port và cài các gói cần thiết:
```bash
# Tải và chạy script cài đặt tự động cho họ RHEL
curl -sSL https://raw.githubusercontent.com/your-repo/bandwidth-hub/main/scripts/install-alma.sh -o install-alma.sh && sudo bash install-alma.sh
```

Hoặc nếu bạn clone toàn bộ mã nguồn về máy:

```bash
git clone https://github.com/your-repo/bandwidth-hub.git /opt/bandwidth-hub
cd /opt/bandwidth-hub
# Chọn script tương ứng với HĐH của bạn:
sudo bash scripts/install.sh       # Cho Ubuntu/Debian
# hoặc
sudo bash scripts/install-alma.sh  # Cho AlmaLinux/Rocky/RHEL
```

### Script cài đặt tự động làm những gì?
- [x] Tạo tự động **2GB Swapfile** (nếu VPS chưa có).
- [x] Tinh chỉnh tham số nhân **Linux Kernel Sysctl** (`vm.swappiness=10`, `somaxconn=2048`).
- [x] Thiết lập crontab tự động dọn RAM mỗi 4 tiếng.
- [x] Khởi tạo **Systemd Service** `bandwidth-hub.service` tự chạy cùng hệ điều hành.
- [x] Xuất địa chỉ IP truy cập Web Dashboard: `http://<IP_VPS>:8888`.

---

## 🛠️ Tùy chọn Khởi chạy: Go Binary vs Python

Bandwidth Hub cung cấp **2 lựa chọn Backend tương thích 100%**:

### Cách 1: Sử dụng Python 3 (Khuyên dùng - Có sẵn trên 100% Linux VPS)
Không cần cài đặt thêm bất kỳ thư viện nào (`Zero pip dependencies`):
```bash
python3 server.py 8888
```

### Cách 2: Sử dụng Go Binary (Siêu tốc - RAM chỉ 10MB)
Nếu bạn có môi trường Golang:
```bash
# Build binary đơn
go build -ldflags="-s -w" -o bandwidth-hub ./cmd/server

# Chạy server
./bandwidth-hub -port 8888 -config config.json
```

---

## 💻 Hướng dẫn sử dụng Giao diện Dashboard

Sau khi mở trình duyệt tại `http://<IP_VPS>:8888`:

1. **Thanh điều khiển trên cùng (Header Controls)**:
   - `[▶ Run All]`: Khởi động toàn bộ các dịch vụ đã được kích hoạt.
   - `[⏹ Stop All]`: Dừng khẩn cấp toàn bộ các worker đang chia sẻ băng thông.
   - `[⚡ Clean RAM]`: Giải phóng PageCache và kích hoạt Garbage Collection tức thì.
   - `[💾 Backup]`: Tải về bản sao lưu cấu hình `config.json` về máy tính.
   - `AUTO: ON/OFF`: Bật/tắt chế độ tự động làm mới trạng thái mỗi 4 giây.

2. **Cấu hình tài khoản cho từng nền tảng**:
   - Trên mỗi thẻ dịch vụ (ví dụ: *Pawns.app* hoặc *Honeygain*), bấm nút **Cài đặt (⚙)**.
   - Nhập Email, Mật khẩu hoặc API Token của tài khoản của bạn.
   - Chọn Proxy muốn gán từ danh sách Kho Proxy.
   - Đặt giới hạn RAM tối đa (Memory Limit MB).
   - Nhấn **Lưu Cấu Hình**.

3. **Xem Nhật ký thời gian thực (Terminal Log)**:
   - Bấm biểu tượng **Terminal (>_)** trên từng thẻ để mở bảng log trực tiếp, theo dõi tình trạng kết nối node, ping và throughput.

---

## 🌐 Quản trị Kho Proxy Chung (Multi-Proxy Pool)

Nếu bạn muốn chạy nhiều node trên cùng 1 VPS mà không bị các nhà mạng phát hiện trùng IP:
1. Bấm nút **[+ Add Proxy]** trên Dashboard.
2. Nhập URL Proxy:
   - Định dạng SOCKS5: `socks5://user:pass@103.145.2.4:1080`
   - Định dạng HTTP: `http://user:pass@142.93.18.2:8080`
3. Nhập mã quốc gia (Geo: VN, US, SG...).
4. Hệ thống sẽ tự động test ping. Bạn có thể nhấn biểu tượng sét (⚡) để kiểm tra lại bất cứ lúc nào.
5. Vào từng service để gán Proxy tương ứng!

---

## 🔧 Lệnh Quản trị Systemd & Dọn dẹp RAM

```bash
# Xem trạng thái dịch vụ Bandwidth Hub
sudo systemctl status bandwidth-hub

# Xem log hoạt động theo thời gian thực
sudo journalctl -u bandwidth-hub -f

# Khởi động lại dịch vụ
sudo systemctl restart bandwidth-hub

# Dừng dịch vụ
sudo systemctl stop bandwidth-hub

# Chạy lệnh ép dọn RAM thủ công
sudo /opt/bandwidth-hub/scripts/trim_memory.sh --force
```

---

## 📂 Cấu trúc thư mục dự án

```text
bandwidth-hub/
├── cmd/
│   └── server/
│       └── main.go              # Entrypoint Golang (Embeds Web UI)
├── internal/
│   ├── config/config.go         # Xử lý cấu hình JSON, backup/restore
│   ├── manager/
│   │   ├── process_manager.go   # Điều khiển native process & Docker container
│   │   └── proxy_manager.go     # Quản lý kho proxy & test ping latency
│   ├── metrics/metrics.go       # Đọc số liệu /proc/meminfo, /proc/stat, clean RAM
│   └── api/handlers.go          # Bộ REST API endpoints
├── server.py                    # Pure Python 3 Backend (Zero dependencies)
├── web/
│   └── index.html               # Single-file Modern Dark Neon Dashboard UI
├── scripts/
│   ├── install.sh               # Script cài đặt tự động 1-click cho VPS
│   ├── trim_memory.sh           # Auto RAM cleaner & cache dropper
│   └── bandwidth-hub.service    # File Systemd unit
├── config.example.json          # File cấu hình mẫu đầy đủ 6 services & proxy
├── go.mod                       # Go module definition
├── Makefile                     # Build & deployment targets
└── README.md                    # Hướng dẫn chi tiết tiếng Việt
```

---
*Phát triển bởi Senior Systems & Full-Stack Developer - Sẵn sàng cho Production 24/7 trên mọi máy chủ Linux VPS.*
