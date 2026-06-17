#!/bin/bash
# 切换服务器地址（IP 或域名），一键完成证书重生成、配置更新、重启
# 用法: bash switch-server.sh 103.231.59.141
#   或: bash switch-server.sh vpn.wzgo.cc

set -e

SERVER="${1}"
APP_DIR="/www/wwwroot/vpn-app"

if [ -z "$SERVER" ]; then
  echo "用法: bash switch-server.sh <IP或域名>"
  exit 1
fi

echo "=== 切换服务器地址为: $SERVER ==="

echo ""
echo "--- [1/5] 生成证书 ---"
cd "$APP_DIR"
SERVER_IP="$SERVER" bash vpn-config/generate-certs.sh

echo ""
echo "--- [2/5] 部署证书到 StrongSwan ---"
cp vpn-config/serverCert.pem /etc/ipsec.d/certs/
cp vpn-config/caCert.pem     /etc/ipsec.d/cacerts/
cp vpn-config/serverKey.pem  /etc/ipsec.d/private/
cp vpn-config/caKey.pem      /etc/ipsec.d/private/
chmod 600 /etc/ipsec.d/private/*.pem
echo "证书部署完成"

echo ""
echo "--- [3/5] 更新 ipsec.conf 和 config.yaml ---"
sed -i "s/leftid=.*/leftid=$SERVER/" /etc/ipsec.conf
sed -i "s/^    left=.*/    left=$(ip route get 1 | awk '{print $7;exit}')/" /etc/ipsec.conf
sed -i "s/server_address:.*/server_address: $SERVER/" "$APP_DIR/backend/config.yaml"
echo "配置更新完成"
echo "ipsec.conf leftid: $(grep leftid /etc/ipsec.conf)"
echo "config.yaml server_address: $(grep server_address $APP_DIR/backend/config.yaml)"

echo ""
echo "--- [4/5] 重启 StrongSwan ---"
systemctl restart strongswan-starter
sleep 2
ipsec statusall | grep -E "uptime|ikev2-eap|local:"

echo ""
echo "--- [5/5] 重新编译后端并重启服务 ---"
cd "$APP_DIR/backend"
go build -o vpn-server .
# 尝试通过宝塔 Go 项目管理器重启，失败则用 systemd
if command -v bt &>/dev/null; then
  pkill -f vpn-server 2>/dev/null || true
  sleep 1
  FRONTEND_DIST="$APP_DIR/frontend/dist" nohup "$APP_DIR/backend/vpn-server" > /tmp/vpn-server.log 2>&1 &
  echo "后端已后台启动，日志: /tmp/vpn-server.log"
elif systemctl is-active --quiet vpn-location 2>/dev/null; then
  systemctl restart vpn-location
  echo "systemd 服务已重启"
else
  pkill -f vpn-server 2>/dev/null || true
  sleep 1
  FRONTEND_DIST="$APP_DIR/frontend/dist" nohup "$APP_DIR/backend/vpn-server" > /tmp/vpn-server.log 2>&1 &
  echo "后端已后台启动"
fi

sleep 2
curl -s http://127.0.0.1:8080/health && echo " ✓ 后端正常" || echo " ✗ 后端未响应"

echo ""
echo "============================================"
echo " 切换完成！服务器地址: $SERVER"
echo " iOS 需要重新下载安装 mobileconfig："
echo " http://$SERVER:8080/api/vpn/download"
echo "============================================"
