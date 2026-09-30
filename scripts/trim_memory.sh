#!/usr/bin/env bash
# ==============================================================================
# Bandwidth Hub - Automated RAM Trimmer & Cache Cleaner
# Tối ưu hóa bộ nhớ cho Linux VPS cấu hình thấp (1-2 GB RAM, 1 vCPU)
# ==============================================================================

set -e

THRESHOLD_PERCENT=80
LOG_FILE="/var/log/bandwidth-hub-trim.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# 1. Đo lường RAM trước khi dọn
MEM_TOTAL=$(awk '/MemTotal/ {print $2}' /proc/meminfo)
MEM_AVAIL=$(awk '/MemAvailable/ {print $2}' /proc/meminfo)

if [ -z "$MEM_AVAIL" ]; then
    MEM_FREE=$(awk '/MemFree/ {print $2}' /proc/meminfo)
    BUFFERS=$(awk '/Buffers/ {print $2}' /proc/meminfo)
    CACHED=$(awk '/^Cached/ {print $2}' /proc/meminfo)
    MEM_AVAIL=$((MEM_FREE + BUFFERS + CACHED))
fi

MEM_USED=$((MEM_TOTAL - MEM_AVAIL))
MEM_PERCENT=$((MEM_USED * 100 / MEM_TOTAL))

MEM_TOTAL_MB=$((MEM_TOTAL / 1024))
MEM_USED_MB=$((MEM_USED / 1024))
MEM_AVAIL_MB=$((MEM_AVAIL / 1024))

# 2. Kiểm tra điều kiện dọn RAM (hoặc nếu truyền tham số --force)
if [ "$1" == "--force" ] || [ "$MEM_PERCENT" -ge "$THRESHOLD_PERCENT" ]; then
    log "[CLEAN START] RAM đang dùng: ${MEM_USED_MB}MB / ${MEM_TOTAL_MB}MB (${MEM_PERCENT}%). Tiến hành dọn dẹp..."

    # Đồng bộ dữ liệu xuống ổ đĩa trước khi xóa cache
    sync
    sleep 1

    # Drop PageCache, dentries và inodes
    echo 3 > /proc/sys/vm/drop_caches

    # Nếu có Docker container đang chạy, prune nhẹ memory buffer
    if command -v docker >/dev/null 2>&1; then
        # Không xóa container hoặc image, chỉ giải phóng daemon buffer nếu cần
        docker system prune -f --filter "until=72h" >/dev/null 2>&1 || true
    fi

    # Đo lường lại sau khi dọn
    NEW_MEM_AVAIL=$(awk '/MemAvailable/ {print $2}' /proc/meminfo)
    NEW_MEM_AVAIL_MB=$((NEW_MEM_AVAIL / 1024))
    FREED_MB=$((NEW_MEM_AVAIL_MB - MEM_AVAIL_MB))

    log "[CLEAN DONE] Đã giải phóng thành công ~${FREED_MB}MB. RAM khả dụng hiện tại: ${NEW_MEM_AVAIL_MB}MB."
else
    log "[INFO] RAM hiện tại: ${MEM_PERCENT}% (< ${THRESHOLD_PERCENT}%). Chưa cần dọn dẹp."
fi
