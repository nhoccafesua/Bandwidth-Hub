#!/usr/bin/env bash
# ==============================================================================
# Bandwidth Hub - One-Click Deployment Script cho AlmaLinux / RockyLinux / RHEL 9/10
# Tối ưu hóa toàn diện cho VPS 1-2 GB RAM, 1 vCPU
# ==============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${CYAN}================================================================${NC}"
echo -e "${CYAN}        BANDWIDTH HUB - ALMALINUX/RHEL DEPLOYMENT INSTALLER    ${NC}"
echo -e "${CYAN}================================================================${NC}"

# 1. Kiểm tra quyền root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}[ERROR] Vui lòng chạy script với quyền root: sudo bash install-alma.sh${NC}"
    exit 1
fi

INSTALL_DIR="/opt/bandwidth-hub"
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo -e "\n${YELLOW}[1/7] Cập nhật hệ thống & cài đặt các gói cần thiết (DNF)...${NC}"
dnf update -y
dnf install -y curl wget python3 cronie iptables procps-ng jq firewalld

# Bật và khởi động cronie (thay thế cho cron trên Ubuntu)
systemctl enable crond
systemctl start crond

# 2. Tự động kiểm tra và tạo 2GB Swapfile
echo -e "\n${YELLOW}[2/7] Kiểm tra bộ nhớ Swap bảo vệ máy chủ...${NC}"
SWAP_EXISTS=$(free -m | awk '/Swap/ {print $2}')
if [ "$SWAP_EXISTS" -lt 1024 ]; then
    echo -e "${CYAN}[+] Phát hiện Swap thấp (${SWAP_EXISTS}MB). Đang tự động tạo 2GB Swapfile...${NC}"
    swapoff -a 2>/dev/null || true
    if [ ! -f /swapfile ]; then
        fallocate -l 2G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=2048
        chmod 600 /swapfile
        mkswap /swapfile
    fi
    swapon /swapfile
    if ! grep -q '/swapfile' /etc/fstab; then
        echo '/swapfile none swap sw 0 0' >> /etc/fstab
    fi
    echo -e "${GREEN}[✓] Đã kích hoạt 2GB Swapfile thành công!${NC}"
else
    echo -e "${GREEN}[✓] Swapfile đã có sẵn (${SWAP_EXISTS}MB). Bỏ qua bước tạo swap.${NC}"
fi

# 3. Tinh chỉnh Linux Kernel Sysctl
echo -e "\n${YELLOW}[3/7] Tinh chỉnh Linux Kernel (sysctl) cho Low-RAM VPS...${NC}"
SYSCTL_CONF="/etc/sysctl.d/99-bandwidth-hub.conf"
cat << 'EOF' > "$SYSCTL_CONF"
vm.swappiness = 10
vm.vfs_cache_pressure = 50
vm.overcommit_memory = 1
net.core.somaxconn = 2048
net.ipv4.tcp_max_syn_backlog = 2048
net.ipv4.ip_local_port_range = 10240 65535
EOF
sysctl -p "$SYSCTL_CONF" >/dev/null 2>&1 || sysctl --system >/dev/null 2>&1
echo -e "${GREEN}[✓] Tinh chỉnh Kernel hoàn tất.${NC}"

# 4. Sao chép mã nguồn vào /opt/bandwidth-hub
echo -e "\n${YELLOW}[4/7] Cài đặt mã nguồn vào ${INSTALL_DIR}...${NC}"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/web"
mkdir -p "$INSTALL_DIR/scripts"

cp -r "$CURRENT_DIR/server.py" "$INSTALL_DIR/" 2>/dev/null || true
cp -r "$CURRENT_DIR/web/index.html" "$INSTALL_DIR/web/" 2>/dev/null || true
cp -r "$CURRENT_DIR/scripts/trim_memory.sh" "$INSTALL_DIR/scripts/" 2>/dev/null || true
cp -r "$CURRENT_DIR/scripts/bandwidth-hub.service" "$INSTALL_DIR/scripts/" 2>/dev/null || true

if [ ! -f "$INSTALL_DIR/config.json" ]; then
    if [ -f "$CURRENT_DIR/config.json" ]; then
        cp "$CURRENT_DIR/config.json" "$INSTALL_DIR/"
    fi
fi

chmod +x "$INSTALL_DIR/server.py"
chmod +x "$INSTALL_DIR/scripts/trim_memory.sh"

# 5. Mở port Firewall (firewalld)
echo -e "\n${YELLOW}[5/7] Cấu hình Firewall mở cổng 8888...${NC}"
systemctl enable firewalld --now >/dev/null 2>&1 || true
if command -v firewall-cmd >/dev/null 2>&1; then
    firewall-cmd --zone=public --add-port=8888/tcp --permanent >/dev/null 2>&1 || true
    firewall-cmd --reload >/dev/null 2>&1 || true
    echo -e "${GREEN}[✓] Đã mở cổng 8888 qua firewalld.${NC}"
else
    echo -e "${CYAN}[!] firewalld không khả dụng, bỏ qua cấu hình tường lửa.${NC}"
fi

# 6. Cài đặt Cron Job tự động dọn RAM (Mỗi 4 tiếng)
echo -e "\n${YELLOW}[6/7] Thiết lập Cron Job tự động dọn RAM...${NC}"
CRON_JOB="0 */4 * * * /opt/bandwidth-hub/scripts/trim_memory.sh >/dev/null 2>&1"
(crontab -l 2>/dev/null | grep -Fv "/opt/bandwidth-hub/scripts/trim_memory.sh" ; echo "$CRON_JOB") | crontab -
echo -e "${GREEN}[✓] Cron dọn dẹp RAM đã được kích hoạt.${NC}"

# 7. Cài đặt và kích hoạt Systemd Service
echo -e "\n${YELLOW}[7/7] Cài đặt Systemd Service cho Bandwidth Hub...${NC}"
cp "$INSTALL_DIR/scripts/bandwidth-hub.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable bandwidth-hub
systemctl restart bandwidth-hub

# Lấy địa chỉ IP Public
SERVER_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || echo "IP-CỦA-VPS")

echo -e "\n${GREEN}================================================================${NC}"
echo -e "${GREEN}      CÀI ĐẶT BANDWIDTH HUB (ALMALINUX) HOÀN TẤT!               ${NC}"
echo -e "${GREEN}================================================================${NC}"
echo -e "Dashboard Web UI: ${CYAN}http://${SERVER_IP}:8888${NC}"
echo -e "Thư mục cài đặt : ${YELLOW}${INSTALL_DIR}${NC}"
echo -e "Xem trạng thái  : ${CYAN}systemctl status bandwidth-hub${NC}"
echo -e "================================================================\n"
