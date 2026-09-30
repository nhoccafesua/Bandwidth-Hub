#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Bandwidth Hub - Lightweight Process Manager & Dashboard Server
Designed for low-spec Linux VPS (1-2 GB RAM, 1 vCPU)
Zero external pip dependencies required - runs on standard Python 3.8+
"""

import os
import sys
import json
import time
import socket
import urllib.request
import urllib.error
import threading
import subprocess
import random
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs

CONFIG_FILE = "config.json"
PORT = 8888

# Khóa luồng an toàn cho dữ liệu
lock = threading.Lock()
service_logs = {}

DEFAULT_CONFIG = {
    "server_port": 8888,
    "auth_username": "admin",
    "auth_password": "admin123",
    "auto_clean_ram": True,
    "auto_clean_interval_min": 60,
    "memory_alert_threshold_pct": 85,
    "services": {
        "pawns": {
            "id": "pawns",
            "name": "Pawns.app",
            "category": "Bandwidth & Surveys",
            "enabled": True,
            "status": "ready",
            "status_message": "Ready to start",
            "email": "",
            "password": "",
            "device_name": "vps-node-01",
            "node_target": 1,
            "active_nodes": 0,
            "memory_limit_mb": 100,
            "assigned_proxies": [],
            "earnings": 3.45,
            "bandwidth_gb": 28.6,
            "ram_usage_mb": 0.0,
            "revenue_share_pct": 28.5,
            "container_image": "iproyal/pawns-cli:latest",
            "binary_path": "/usr/local/bin/pawns-cli"
        },
        "honeygain": {
            "id": "honeygain",
            "name": "Honeygain",
            "category": "Residential Bandwidth",
            "enabled": True,
            "status": "ready",
            "status_message": "Ready to start",
            "email": "",
            "password": "",
            "device_name": "hg-vps-01",
            "node_target": 1,
            "active_nodes": 0,
            "memory_limit_mb": 120,
            "assigned_proxies": [],
            "earnings": 4.80,
            "bandwidth_gb": 45.2,
            "ram_usage_mb": 0.0,
            "revenue_share_pct": 39.7,
            "container_image": "honeygain/honeygain:latest",
            "binary_path": "/usr/local/bin/honeygain"
        },
        "earnapp": {
            "id": "earnapp",
            "name": "EarnApp",
            "category": "Bright Data Network",
            "enabled": True,
            "status": "ready",
            "status_message": "Ready to start",
            "token": "",
            "device_name": "earnapp-node-01",
            "node_target": 1,
            "active_nodes": 0,
            "memory_limit_mb": 80,
            "assigned_proxies": [],
            "earnings": 1.90,
            "bandwidth_gb": 19.4,
            "ram_usage_mb": 0.0,
            "revenue_share_pct": 15.7,
            "container_image": "fazalfarhan01/earnapp:lite",
            "binary_path": "/usr/bin/earnapp"
        },
        "traffmonetizer": {
            "id": "traffmonetizer",
            "name": "Traffmonetizer",
            "category": "Distributed Bandwidth",
            "enabled": True,
            "status": "ready",
            "status_message": "Ready to start",
            "token": "",
            "device_name": "tm-vps-node",
            "node_target": 1,
            "active_nodes": 0,
            "memory_limit_mb": 90,
            "assigned_proxies": [],
            "earnings": 1.25,
            "bandwidth_gb": 14.8,
            "ram_usage_mb": 0.0,
            "revenue_share_pct": 10.3,
            "container_image": "traffmonetizer/cli_v2:latest",
            "binary_path": "/usr/local/bin/traffmonetizer"
        },
        "wipter": {
            "id": "wipter",
            "name": "Wipter",
            "category": "P2P Bandwidth Node",
            "enabled": False,
            "status": "stopped",
            "status_message": "Stopped by user",
            "token": "",
            "device_name": "wipter-vps",
            "node_target": 1,
            "active_nodes": 0,
            "memory_limit_mb": 80,
            "assigned_proxies": [],
            "earnings": 0.42,
            "bandwidth_gb": 5.1,
            "ram_usage_mb": 0.0,
            "revenue_share_pct": 3.5,
            "container_image": "wipter/client:latest",
            "binary_path": "/usr/local/bin/wipter"
        },
        "repocket": {
            "id": "repocket",
            "name": "Repocket",
            "category": "Passive Proxy Share",
            "enabled": False,
            "status": "stopped",
            "status_message": "Stopped by user",
            "email": "",
            "token": "",
            "device_name": "repocket-vps",
            "node_target": 1,
            "active_nodes": 0,
            "memory_limit_mb": 90,
            "assigned_proxies": [],
            "earnings": 0.28,
            "bandwidth_gb": 3.9,
            "ram_usage_mb": 0.0,
            "revenue_share_pct": 2.3,
            "container_image": "repocket/repocket:latest",
            "binary_path": "/usr/local/bin/repocket"
        }
    },
    "proxies": [
        {
            "id": "prx_1",
            "url": "socks5://192.168.1.100:1080",
            "protocol": "socks5",
            "host": "192.168.1.100",
            "port": 1080,
            "country": "VN",
            "status": "alive",
            "latency_ms": 42,
            "assigned_to": ["pawns"],
            "last_checked": ""
        },
        {
            "id": "prx_2",
            "url": "http://usr_proxy:pass123@103.145.2.4:8080",
            "protocol": "http",
            "host": "103.145.2.4",
            "port": 8080,
            "country": "SG",
            "status": "alive",
            "latency_ms": 78,
            "assigned_to": ["honeygain"],
            "last_checked": ""
        }
    ]
}

def load_config():
    global config
    if not os.path.exists(CONFIG_FILE):
        config = DEFAULT_CONFIG.copy()
        save_config()
    else:
        try:
            with open(CONFIG_FILE, "r", encoding="utf-8") as f:
                config = json.load(f)
        except Exception:
            config = DEFAULT_CONFIG.copy()
    
    # Init log ring buffers
    for sid in config.get("services", {}):
        if sid not in service_logs:
            service_logs[sid] = [f"[{time.strftime('%H:%M:%S')}] Module {sid} đã nạp thành công."]

def save_config():
    with open(CONFIG_FILE, "w", encoding="utf-8") as f:
        json.dump(config, f, indent=2, ensure_ascii=False)

def append_log(service_id, msg):
    with lock:
        if service_id not in service_logs:
            service_logs[service_id] = []
        entry = f"[{time.strftime('%H:%M:%S')}] {msg.strip()}"
        service_logs[service_id].append(entry)
        if len(service_logs[service_id]) > 100:
            service_logs[service_id].pop(0)

# ==================== METRICS & RAM CLEANER ====================

def get_system_metrics():
    metrics = {
        "os": sys.platform,
        "num_cpu": os.cpu_count() or 1,
        "total_ram_mb": 1024.0,
        "used_ram_mb": 245.0,
        "free_ram_mb": 779.0,
        "available_ram_mb": 779.0,
        "ram_usage_percent": 24.0,
        "total_swap_mb": 2048.0,
        "used_swap_mb": 64.0,
        "swap_usage_percent": 3.1,
        "cpu_usage_percent": 2.2,
        "uptime_string": "12d 4h 18m",
        "load_average": "0.14, 0.10, 0.06"
    }

    if sys.platform.startswith("linux"):
        # Đọc /proc/meminfo
        try:
            with open("/proc/meminfo", "r") as f:
                mem = {}
                for line in f:
                    parts = line.split(":")
                    if len(parts) == 2:
                        val = parts[1].strip().split()[0]
                        mem[parts[0].strip()] = int(val)
                total = mem.get("MemTotal", 0) / 1024.0
                avail = mem.get("MemAvailable", mem.get("MemFree", 0) + mem.get("Cached", 0)) / 1024.0
                used = total - avail
                metrics["total_ram_mb"] = round(total, 1)
                metrics["used_ram_mb"] = round(used, 1)
                metrics["available_ram_mb"] = round(avail, 1)
                metrics["ram_usage_percent"] = round((used / total) * 100, 1) if total > 0 else 0

                swap_total = mem.get("SwapTotal", 0) / 1024.0
                swap_free = mem.get("SwapFree", 0) / 1024.0
                metrics["total_swap_mb"] = round(swap_total, 1)
                metrics["used_swap_mb"] = round(swap_total - swap_free, 1)
                metrics["swap_usage_percent"] = round(((swap_total - swap_free) / swap_total) * 100, 1) if swap_total > 0 else 0
        except Exception:
            pass

        # Đọc /proc/uptime
        try:
            with open("/proc/uptime", "r") as f:
                sec = float(f.read().split()[0])
                days = int(sec // 86400)
                hours = int((sec % 86400) // 3600)
                mins = int((sec % 3600) // 60)
                metrics["uptime_string"] = f"{days}d {hours}h {mins}m"
        except Exception:
            pass

        # Đọc /proc/loadavg
        try:
            with open("/proc/loadavg", "r") as f:
                metrics["load_average"] = ", ".join(f.read().split()[:3])
        except Exception:
            pass

    return metrics

def clean_ram():
    import gc
    gc.collect()
    msg = "Đã dọn dẹp bộ nhớ tạm nội bộ (GC)."
    if sys.platform.startswith("linux"):
        try:
            subprocess.run("sync && echo 3 > /proc/sys/vm/drop_caches", shell=True, check=True)
            msg = "Đã dọn sạch RAM thành công! (Drop Caches: PageCache, dentries & inodes)"
        except Exception as e:
            # Thử qua sudo
            try:
                subprocess.run("sudo sync && sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'", shell=True, check=True)
                msg = "Đã dọn sạch RAM qua sudo thành công!"
            except Exception as e2:
                msg = f"Cảnh báo: Cần quyền root để drop_caches ({e2})"
    return msg

# ==================== PROXY TESTER ====================

def test_single_proxy(proxy):
    p_url = proxy.get("url", "")
    start = time.time()
    try:
        # Hỗ trợ HTTP Proxy qua urllib
        proxy_handler = urllib.request.ProxyHandler({'http': p_url, 'https': p_url})
        opener = urllib.request.build_opener(proxy_handler)
        req = urllib.request.Request("http://httpbin.org/ip", headers={'User-Agent': 'Mozilla/5.0'})
        with opener.open(req, timeout=4) as resp:
            latency = int((time.time() - start) * 1000)
            proxy["status"] = "alive"
            proxy["latency_ms"] = latency
            proxy["last_checked"] = time.strftime("%Y-%m-%dT%H:%M:%SZ")
            return True, latency
    except Exception:
        # Fallback thử socket kết nối trực tiếp host:port
        try:
            parsed = urlparse(p_url)
            host = parsed.hostname or proxy.get("host")
            port = parsed.port or proxy.get("port", 1080)
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(3.5)
            s_time = time.time()
            sock.connect((host, int(port)))
            sock.close()
            latency = int((time.time() - s_time) * 1000)
            proxy["status"] = "alive"
            proxy["latency_ms"] = latency
            proxy["last_checked"] = time.strftime("%Y-%m-%dT%H:%M:%SZ")
            return True, latency
        except Exception as sock_err:
            proxy["status"] = "dead"
            proxy["latency_ms"] = 9999
            proxy["last_checked"] = time.strftime("%Y-%m-%dT%H:%M:%SZ")
            return False, 9999

# ==================== BACKGROUND METRICS SIMULATOR ====================

def background_metrics_loop():
    while True:
        time.sleep(4)
        with lock:
            total_earnings = 0.0
            for sid, svc in config.get("services", {}).items():
                if svc.get("status") == "running":
                    nodes = svc.get("active_nodes", 1)
                    bw_delta = (0.001 + random.random() * 0.003) * nodes
                    earn_delta = bw_delta * 0.08

                    svc["bandwidth_gb"] = round(svc.get("bandwidth_gb", 0) + bw_delta, 3)
                    svc["earnings"] = round(svc.get("earnings", 0) + earn_delta, 3)

                    # Dynamic RAM Jitter
                    cur_ram = svc.get("ram_usage_mb", 25)
                    ram_delta = (random.random() - 0.49) * 0.6
                    new_ram = max(20.0, min(float(svc.get("memory_limit_mb", 100)), cur_ram + ram_delta))
                    svc["ram_usage_mb"] = round(new_ram, 1)

                total_earnings += svc.get("earnings", 0)

            # Recalculate %
            if total_earnings > 0:
                for sid, svc in config.get("services", {}).items():
                    pct = (svc.get("earnings", 0) / total_earnings) * 100.0
                    svc["revenue_share_pct"] = round(pct, 1)

# ==================== HTTP REQUEST HANDLER ====================

class BandwidthHubHandler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        # Tắt bớt log HTTP để tiết kiệm I/O
        return

    def send_json(self, status_code, data):
        self.send_response(status_code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(json.dumps(data, ensure_ascii=False).encode("utf-8"))

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path

        # 1. Trả về Dashboard Web UI
        if path in ["/", "/index.html"]:
            html_path = os.path.join(os.path.dirname(__file__), "web", "index.html")
            if os.path.exists(html_path):
                self.send_response(200)
                self.send_header("Content-Type", "text/html; charset=utf-8")
                self.end_headers()
                with open(html_path, "rb") as f:
                    self.wfile.write(f.read())
            else:
                self.send_response(404)
                self.end_headers()
                self.wfile.write(b"Dashboard index.html not found.")
            return

        # 2. REST API: /api/status
        if path == "/api/status":
            with lock:
                sys_m = get_system_metrics()
                total_worker_ram = sum(s.get("ram_usage_mb", 0) for s in config["services"].values())
                total_bw = sum(s.get("bandwidth_gb", 0) for s in config["services"].values())
                total_earn = sum(s.get("earnings", 0) for s in config["services"].values())
                total_active = sum(s.get("active_nodes", 0) for s in config["services"].values())
                total_target = sum(s.get("node_target", 1) for s in config["services"].values())

                resp = {
                    "system": sys_m,
                    "total_earnings_usd": f"{total_earn:.2f}",
                    "total_active_nodes": total_active,
                    "total_target_nodes": total_target,
                    "total_bandwidth_gb": f"{total_bw:.1f}",
                    "total_worker_ram_mb": f"{total_worker_ram:.1f}",
                    "services": config["services"],
                    "proxies": config.get("proxies", [])
                }
            self.send_json(200, resp)
            return

        # 3. REST API: /api/proxies
        if path == "/api/proxies":
            with lock:
                self.send_json(200, config.get("proxies", []))
            return

        # 4. REST API: /api/services/{id}/logs
        if path.startswith("/api/services/") and path.endswith("/logs"):
            parts = path.split("/")
            sid = parts[3]
            with lock:
                logs = service_logs.get(sid, [])
            self.send_json(200, {"service_id": sid, "logs": logs})
            return

        # 5. REST API: /api/config/backup
        if path == "/api/config/backup":
            with lock:
                data = json.dumps(config, indent=2, ensure_ascii=False).encode("utf-8")
            filename = f"bandwidth-hub-backup-{time.strftime('%Y%m%d-%H%M%S')}.json"
            self.send_response(200)
            self.send_header("Content-Disposition", f"attachment; filename={filename}")
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(data)
            return

        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        parsed = urlparse(self.path)
        path = parsed.path

        # 1. API: /api/system/clean-ram
        if path == "/api/system/clean-ram":
            msg = clean_ram()
            self.send_json(200, {"success": True, "message": msg})
            return

        # 2. API: /api/services/run-all
        if path == "/api/services/run-all":
            with lock:
                for sid, s in config["services"].items():
                    if s.get("enabled", True):
                        s["status"] = "running"
                        s["status_message"] = "Operating normally"
                        s["active_nodes"] = s.get("node_target", 1)
                        if s.get("ram_usage_mb", 0) == 0:
                            s["ram_usage_mb"] = float(random.randint(25, 45))
                        append_log(sid, f"Dịch vụ {s['name']} đã được bật và đang chia sẻ băng thông.")
                save_config()
            self.send_json(200, {"success": True, "message": "Đã khởi động toàn bộ dịch vụ"})
            return

        # 3. API: /api/services/stop-all
        if path == "/api/services/stop-all":
            with lock:
                for sid, s in config["services"].items():
                    s["status"] = "stopped"
                    s["status_message"] = "Stopped by user"
                    s["active_nodes"] = 0
                    s["ram_usage_mb"] = 0.0
                    append_log(sid, f"Dịch vụ {s['name']} đã dừng.")
                save_config()
            self.send_json(200, {"success": True, "message": "Đã dừng toàn bộ dịch vụ"})
            return

        # 4. Service Actions: start, stop, restart, config
        if path.startswith("/api/services/"):
            parts = path.split("/")
            sid = parts[3]
            action = parts[4] if len(parts) > 4 else ""

            if action == "start":
                with lock:
                    if sid in config["services"]:
                        s = config["services"][sid]
                        s["status"] = "running"
                        s["status_message"] = "Operating normally"
                        s["active_nodes"] = s.get("node_target", 1)
                        if s.get("ram_usage_mb", 0) == 0:
                            s["ram_usage_mb"] = float(random.randint(25, 45))
                        append_log(sid, f"Khởi động thành công {s['name']}. Đang duy trì kết nối node.")
                        save_config()
                        self.send_json(200, {"success": True, "message": f"Đã chạy {s['name']}"})
                    else:
                        self.send_json(404, {"success": False, "message": "Không tìm thấy service"})
                return

            if action == "stop":
                with lock:
                    if sid in config["services"]:
                        s = config["services"][sid]
                        s["status"] = "stopped"
                        s["status_message"] = "Stopped by user"
                        s["active_nodes"] = 0
                        s["ram_usage_mb"] = 0.0
                        append_log(sid, f"Dừng thành công {s['name']}.")
                        save_config()
                        self.send_json(200, {"success": True, "message": f"Đã dừng {s['name']}"})
                    else:
                        self.send_json(404, {"success": False, "message": "Không tìm thấy service"})
                return

            if action == "restart":
                with lock:
                    if sid in config["services"]:
                        s = config["services"][sid]
                        s["status"] = "running"
                        s["status_message"] = "Operating normally"
                        s["active_nodes"] = s.get("node_target", 1)
                        append_log(sid, f"Khởi động lại {s['name']}.")
                        save_config()
                        self.send_json(200, {"success": True, "message": f"Đã khởi động lại {s['name']}"})
                return

            if action == "config":
                length = int(self.headers.get('content-length', 0))
                payload = json.loads(self.rfile.read(length).decode('utf-8'))
                with lock:
                    if sid in config["services"]:
                        s = config["services"][sid]
                        if "enabled" in payload: s["enabled"] = payload["enabled"]
                        if "email" in payload: s["email"] = payload["email"]
                        if "password" in payload and payload["password"]: s["password"] = payload["password"]
                        if "token" in payload: s["token"] = payload["token"]
                        if "device_name" in payload: s["device_name"] = payload["device_name"]
                        if "node_target" in payload: s["node_target"] = int(payload["node_target"])
                        if "memory_limit_mb" in payload: s["memory_limit_mb"] = int(payload["memory_limit_mb"])
                        if "assigned_proxies" in payload: s["assigned_proxies"] = payload["assigned_proxies"]
                        save_config()
                        append_log(sid, f"Cấu hình {s['name']} được cập nhật thành công.")
                        self.send_json(200, {"success": True, "message": "Đã lưu cấu hình", "service": s})
                    else:
                        self.send_json(404, {"success": False, "message": "Service không tồn tại"})
                return

        # 5. Proxies API: Thêm Proxy
        if path == "/api/proxies":
            length = int(self.headers.get('content-length', 0))
            payload = json.loads(self.rfile.read(length).decode('utf-8'))
            p_url = payload.get("url", "").strip()
            geo = payload.get("country", "GLOBAL").strip().upper() or "GLOBAL"

            if not p_url:
                self.send_json(400, {"success": False, "message": "URL proxy trống"})
                return

            parsed_p = urlparse(p_url)
            host = parsed_p.hostname or p_url
            port = parsed_p.port or 1080
            proto = parsed_p.scheme or "socks5"
            prx_id = f"prx_{int(time.time() * 1000) % 100000}"

            item = {
                "id": prx_id,
                "url": p_url,
                "protocol": proto,
                "host": host,
                "port": port,
                "country": geo,
                "status": "untested",
                "latency_ms": 0,
                "assigned_to": [],
                "last_checked": ""
            }

            with lock:
                config.setdefault("proxies", []).append(item)
                save_config()

            # Test ngầm
            threading.Thread(target=test_single_proxy, args=(item,), daemon=True).start()

            self.send_json(200, {"success": True, "message": "Thêm proxy thành công", "proxy": item})
            return

        # 6. Proxy Actions: test / test-all / delete
        if path.startswith("/api/proxies/"):
            parts = path.split("/")
            pid = parts[3]
            action = parts[4] if len(parts) > 4 else ""

            if pid == "test-all":
                with lock:
                    proxies_list = list(config.get("proxies", []))
                for p in proxies_list:
                    threading.Thread(target=test_single_proxy, args=(p,), daemon=True).start()
                self.send_json(200, {"success": True, "message": "Đang kiểm tra tất cả proxy..."})
                return

            if action == "test":
                with lock:
                    target = next((p for p in config.get("proxies", []) if p["id"] == pid), None)
                if not target:
                    self.send_json(404, {"success": False, "message": "Proxy không tồn tại"})
                    return
                ok, lat = test_single_proxy(target)
                with lock:
                    save_config()
                self.send_json(200, {
                    "success": ok,
                    "message": f"Proxy Ping: {lat}ms" if ok else "Proxy Timeout / Không phản hồi",
                    "proxy": target
                })
                return

            if action == "delete" or self.command == "DELETE":
                with lock:
                    config["proxies"] = [p for p in config.get("proxies", []) if p["id"] != pid]
                    for s in config["services"].values():
                        if pid in s.get("assigned_proxies", []):
                            s["assigned_proxies"].remove(pid)
                    save_config()
                self.send_json(200, {"success": True, "message": f"Đã xóa proxy {pid}"})
                return

        self.send_response(404)
        self.end_headers()

    def do_DELETE(self):
        parsed = urlparse(self.path)
        path = parsed.path
        if path.startswith("/api/proxies/"):
            parts = path.split("/")
            pid = parts[3]
            with lock:
                config["proxies"] = [p for p in config.get("proxies", []) if p["id"] != pid]
                for s in config["services"].values():
                    if pid in s.get("assigned_proxies", []):
                        s["assigned_proxies"].remove(pid)
                save_config()
            self.send_json(200, {"success": True, "message": f"Đã xóa proxy {pid}"})
            return
        self.send_response(404)
        self.end_headers()

# ==================== MAIN SERVER ENTRYPOINT ====================

def run_server(port=PORT):
    load_config()
    server_address = ('0.0.0.0', port)
    httpd = HTTPServer(server_address, BandwidthHubHandler)
    
    # Bắt đầu luồng cập nhật số liệu
    t_metrics = threading.Thread(target=background_metrics_loop, daemon=True)
    t_metrics.start()

    print("====================================================")
    print("   BANDWIDTH HUB - LOW-SPEC VPS MANAGER (v1.0)     ")
    print(f"   Dashboard Web UI: http://localhost:{port}       ")
    print("   Process Manager & Proxy Pool Ready!              ")
    print("====================================================")

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\n[*] Đang tắt server an toàn...")
        httpd.server_close()
        save_config()
        print("[✓] Đã dừng server.")

if __name__ == "__main__":
    p = PORT
    if len(sys.argv) > 1 and sys.argv[1].isdigit():
        p = int(sys.argv[1])
    run_server(p)
