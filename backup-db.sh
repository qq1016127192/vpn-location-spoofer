#!/bin/bash

# VPN虚拟定位项目数据库备份脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
BACKUP_DIR="/opt/vpn-location-spoofer/backups"
RETENTION_DAYS=7
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 从配置文件读取数据库信息
CONFIG_FILE="/opt/vpn-location-spoofer/backend/config.yaml"

if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}配置文件不存在: $CONFIG_FILE${NC}"
    exit 1
fi

# 解析YAML配置文件获取数据库信息
DB_NAME=$(grep "^  database:" "$CONFIG_FILE" -A 5 | grep "database:" | awk '{print $2}')
DB_USER=$(grep "^  database:" "$CONFIG_FILE" -A 5 | grep "username:" | awk '{print $2}')
DB_PASS=$(grep "^  database:" "$CONFIG_FILE" -A 5 | grep "password:" | awk '{print $2}')

# 如果解析失败，使用默认值
DB_NAME=${DB_NAME:-vpn_location}
DB_USER=${DB_USER:-vpn_user}

echo "========================================="
echo "数据库备份脚本"
echo "========================================="
echo "数据库名: $DB_NAME"
echo "数据库用户: $DB_USER"
echo "备份时间: $TIMESTAMP"
echo ""

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 备份数据库
echo -e "${GREEN}正在备份数据库...${NC}"
BACKUP_FILE="$BACKUP_DIR/db_backup_${TIMESTAMP}.sql"

mysqldump -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" > "$BACKUP_FILE" 2>/dev/null

if [ $? -eq 0 ]; then
    # 压缩备份文件
    gzip "$BACKUP_FILE"
    BACKUP_FILE="${BACKUP_FILE}.gz"
    
    FILE_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo -e "${GREEN}备份成功: $BACKUP_FILE ($FILE_SIZE)${NC}"
else
    echo -e "${RED}备份失败${NC}"
    exit 1
fi

# 清理旧备份
echo -e "${GREEN}清理 $RETENTION_DAYS 天前的旧备份...${NC}"
find "$BACKUP_DIR" -name "db_backup_*.sql.gz" -type f -mtime +$RETENTION_DAYS -delete

# 显示备份列表
echo ""
echo "当前备份文件:"
ls -lh "$BACKUP_DIR"/db_backup_*.sql.gz 2>/dev/null || echo "暂无备份文件"

echo ""
echo -e "${GREEN}备份完成！${NC}"
echo "备份目录: $BACKUP_DIR"
echo "保留天数: $RETENTION_DAYS"
