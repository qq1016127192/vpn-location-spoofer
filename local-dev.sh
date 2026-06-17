#!/bin/bash
# macOS 本地开发启动脚本
# 用法: SERVER_IP=192.168.10.7 bash local-dev.sh

set -e

SERVER_IP="${SERVER_IP:-192.168.10.7}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "========================================"
echo " macOS 本地开发环境启动"
echo " SERVER_IP: $SERVER_IP"
echo "========================================"

# ── 1. 生成证书并部署到本地 strongSwan ────────────────────────────────────────
echo ""
echo "[1/3] 生成证书并配置 strongSwan..."
SERVER_IP=$SERVER_IP bash "$SCRIPT_DIR/vpn-config/generate-certs.sh"

# ── 2. 启动后端 ────────────────────────────────────────────────────────────────
echo ""
echo "[2/3] 启动后端服务 (http://$SERVER_IP:8080)..."
cd "$SCRIPT_DIR/backend"

# 更新 config.yaml 为本地配置
cat > config.yaml << EOF
server:
  port: 8080
  mode: debug

vpn:
  server_address: $SERVER_IP
  ipsec_bin: /opt/homebrew/bin/ipsec
  ipsec_secrets: /opt/homebrew/etc/ipsec.secrets

cert:
  root_ca_path: $SCRIPT_DIR/vpn-config/caCert.pem
  root_key_path: $SCRIPT_DIR/vpn-config/caKey.pem
EOF

echo "后端配置已更新，运行命令:"
echo "  cd backend && go run ."
echo ""
echo "[3/3] 启动前端开发服务器 (http://localhost:3000)..."
echo "运行命令:"
echo "  cd frontend && npm run dev"
echo ""
echo "========================================"
echo " 启动顺序："
echo "  1. 终端1: cd backend && go run ."
echo "  2. 终端2: cd frontend && npm run dev"
echo "  3. iOS Safari: http://$SERVER_IP:8080"
echo "========================================"
