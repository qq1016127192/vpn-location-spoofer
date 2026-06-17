#!/bin/bash
# 服务器一键部署脚本 - Debian/Ubuntu (宝塔环境)
#
# 用法（服务器上执行）:
#   SERVER_IP=103.231.59.141 bash server-setup.sh
#
# 可选参数:
#   DB_PASS=自定义密码   数据库密码（默认 vpnpass2024）
#
# 前置条件：已将 vpn-package.tar.gz 解压到当前目录

set -e

# ── 参数 ──────────────────────────────────────────────────────────────────────
SERVER_IP="${SERVER_IP:-$(curl -s --max-time 5 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')}"
DB_PASS="${DB_PASS:-vpnpass2024}"
DEPLOY_DIR="/opt/vpn-location"
CERT_DIR="$DEPLOY_DIR/certs"

log()  { echo "[$(date '+%H:%M:%S')] $*"; }
die()  { echo "[ERROR] $*"; exit 1; }
ok()   { echo "[$(date '+%H:%M:%S')] ✓ $*"; }
warn() { echo "[$(date '+%H:%M:%S')] ⚠ $*"; }

# SERVER_IP 优先用环境变量，否则自动获取公网 IP
if [ -z "$SERVER_IP" ]; then
  SERVER_IP=$(curl -s --max-time 5 ifconfig.me 2>/dev/null \
    || curl -s --max-time 5 api.ipify.org 2>/dev/null \
    || hostname -I | awk '{print $1}')
fi
[ -z "$SERVER_IP" ] && die "无法获取服务器 IP，请手动指定: SERVER_IP=1.2.3.4 bash server-setup.sh"

# 找到脚本所在目录，兼容从任意目录执行
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
[ ! -f "$SCRIPT_DIR/backend/go.mod" ] && die "请在解压目录下执行此脚本（找不到 backend/go.mod）"
cd "$SCRIPT_DIR"

echo "========================================"
echo " VPN 虚拟定位系统 - 服务器部署"
echo " SERVER_IP : $SERVER_IP"
echo " DEPLOY_DIR: $DEPLOY_DIR"
echo "========================================"

# ── 1. 安装系统依赖 ────────────────────────────────────────────────────────────
log "[1/8] 安装系统依赖..."
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
  strongswan strongswan-pki \
  libcharon-extra-plugins libcharon-extauth-plugins \
  golang-go \
  nodejs npm \
  openssl curl iptables iptables-persistent

ok "系统依赖安装完成"
ipsec version | head -1
go version
node -v

# ── 2. 生成证书（服务器上直接生成，CN 绑定公网 IP）───────────────────────────
log "[2/8] 生成证书..."
mkdir -p $CERT_DIR
mkdir -p /etc/ipsec.d/{certs,private,cacerts}

# 生成 CA
openssl genrsa -out $CERT_DIR/caKey.pem 2048
openssl req -new -x509 -days 3650 \
  -key $CERT_DIR/caKey.pem \
  -out $CERT_DIR/caCert.pem \
  -subj "/C=CN/ST=Beijing/O=VPN Location Spoofer/OU=CA/CN=VPN Root CA"

# 生成服务器私钥和 CSR
openssl genrsa -out $CERT_DIR/serverKey.pem 2048
openssl req -new \
  -key $CERT_DIR/serverKey.pem \
  -out /tmp/serverReq.pem \
  -subj "/C=CN/ST=Beijing/O=VPN Location Spoofer/OU=Server/CN=$SERVER_IP"

# SAN 扩展（iOS 16+ 必须）
cat > /tmp/server_ext.cnf << EOF
subjectAltName = IP:$SERVER_IP
extendedKeyUsage = serverAuth
EOF

openssl x509 -req \
  -in /tmp/serverReq.pem \
  -CA $CERT_DIR/caCert.pem \
  -CAkey $CERT_DIR/caKey.pem \
  -CAcreateserial \
  -out $CERT_DIR/serverCert.pem \
  -days 3650 -extfile /tmp/server_ext.cnf

chmod 600 $CERT_DIR/caKey.pem $CERT_DIR/serverKey.pem
chmod 644 $CERT_DIR/caCert.pem $CERT_DIR/serverCert.pem

# 部署到 strongSwan（caKey 不放进去，避免被误读）
cp $CERT_DIR/serverCert.pem /etc/ipsec.d/certs/serverCert.pem
cp $CERT_DIR/serverKey.pem  /etc/ipsec.d/private/serverKey.pem
cp $CERT_DIR/caCert.pem     /etc/ipsec.d/cacerts/caCert.pem
chmod 600 /etc/ipsec.d/private/serverKey.pem
chmod 644 /etc/ipsec.d/certs/serverCert.pem /etc/ipsec.d/cacerts/caCert.pem

ok "证书生成完成"
openssl x509 -in $CERT_DIR/caCert.pem -noout -subject -dates | head -3

# ── 3. 构建前端 ────────────────────────────────────────────────────────────────
log "[3/8] 构建前端..."
cd frontend
npm install --silent
npm run build
cd ..
ok "前端构建完成"

# ── 4. 编译后端 ────────────────────────────────────────────────────────────────
log "[4/8] 编译后端..."
cd backend
go build -ldflags="-s -w" -o /tmp/vpn-server .
cd ..
ok "后端编译完成"

# ── 5. 部署应用文件 ────────────────────────────────────────────────────────────
log "[5/8] 部署应用文件..."
mkdir -p $DEPLOY_DIR/frontend

cp /tmp/vpn-server         $DEPLOY_DIR/vpn-server
cp -r frontend/dist        $DEPLOY_DIR/frontend/dist
chmod +x $DEPLOY_DIR/vpn-server

# 生成服务器专用 config.yaml
cat > $DEPLOY_DIR/config.yaml << EOF
server:
  port: 8080
  mode: release

database:
  host: 127.0.0.1
  port: 3306
  username: vpnlocation
  password: $DB_PASS
  database: vpnlocation
  charset: utf8mb4

vpn:
  server_address: $SERVER_IP
  ipsec_bin: /usr/sbin/ipsec
  ipsec_secrets: /etc/ipsec.secrets

cert:
  root_ca_path: $CERT_DIR/caCert.pem
  root_key_path: $CERT_DIR/caKey.pem

auth:
  jwt_secret: $(openssl rand -hex 32)
  expire_hours: 24
EOF

ok "应用文件部署完成"

# ── 6. 配置 StrongSwan ─────────────────────────────────────────────────────────
log "[6/8] 配置 StrongSwan..."

# ipsec.conf（NAT 环境：left 用内网 IP，leftid 用公网 IP）
INTERNAL_IP=$(ip route get 1.1.1.1 2>/dev/null | awk '/src/{print $7}' | head -1)
[ -z "$INTERNAL_IP" ] && INTERNAL_IP=$(hostname -I | awk '{print $1}')

cat > /etc/ipsec.conf << EOF
config setup
    charondebug="ike 2, knl 2, cfg 2"
    uniqueids=no

conn %default
    keyexchange=ikev2
    ike=aes256-sha256-modp2048!
    esp=aes256-sha256!
    dpdaction=clear
    dpddelay=300s
    rekey=no
    mobike=yes
    forceencaps=yes
    fragmentation=yes

conn ikev2-eap
    left=$INTERNAL_IP
    leftid=$SERVER_IP
    leftcert=serverCert.pem
    leftsendcert=always
    leftsubnet=0.0.0.0/0
    right=%any
    rightauth=eap-mschapv2
    rightsourceip=10.10.0.0/24
    rightdns=8.8.8.8,8.8.4.4
    eap_identity=%any
    auto=add
EOF

cp vpn-config/ipsec.secrets /etc/ipsec.secrets
chmod 600 /etc/ipsec.secrets

# charon.conf：解除半开连接数限制
CHARON_CONF=$(find /etc/strongswan.d -name "charon.conf" 2>/dev/null | head -1)
if [ -n "$CHARON_CONF" ]; then
  cat > "$CHARON_CONF" << 'CHARONEOF'
charon {
    block_threshold = 0
    half_open_timeout = 30
    install_routes = yes
    crypto_test {}
    host_resolver {}
    processor { priority_threads {} }
    start-scripts {}
    stop-scripts {}
    tls {}
    x509 {}
}
CHARONEOF
  ok "charon.conf 已配置"
fi

# 启动 StrongSwan（Debian 11 服务名是 strongswan-starter）
SWAN_SVC="strongswan-starter"
systemctl list-units --full --all 2>/dev/null | grep -q "strongswan-starter" || SWAN_SVC="strongswan"
systemctl enable "$SWAN_SVC"
systemctl restart "$SWAN_SVC"
sleep 3

if ipsec statusall 2>/dev/null | grep -q "ikev2-eap"; then
  ok "StrongSwan 已加载 ikev2-eap 配置"
else
  warn "StrongSwan 状态异常，请检查:"
  ipsec statusall | head -15
fi

# ── 7. IP 转发 + NAT + 防火墙 ─────────────────────────────────────────────────
log "[7/8] 配置网络..."

# IP 转发持久化
grep -q "^net.ipv4.ip_forward" /etc/sysctl.conf \
  && sed -i 's|^net.ipv4.ip_forward.*|net.ipv4.ip_forward = 1|' /etc/sysctl.conf \
  || echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf
sysctl -w net.ipv4.ip_forward=1

# 找出口网卡
OUTIF=$(ip route show default | awk '{print $5}' | head -1)
log "出口网卡: $OUTIF"

# iptables 规则（幂等）
iptables -t nat -C POSTROUTING -s 10.10.0.0/24 -o "$OUTIF" -j MASQUERADE 2>/dev/null \
  || iptables -t nat -A POSTROUTING -s 10.10.0.0/24 -o "$OUTIF" -j MASQUERADE
iptables -C FORWARD -s 10.10.0.0/24 -j ACCEPT 2>/dev/null \
  || iptables -A FORWARD -s 10.10.0.0/24 -j ACCEPT
iptables -C FORWARD -d 10.10.0.0/24 -j ACCEPT 2>/dev/null \
  || iptables -A FORWARD -d 10.10.0.0/24 -j ACCEPT
iptables -C INPUT -p udp --dport 500  -j ACCEPT 2>/dev/null \
  || iptables -A INPUT -p udp --dport 500  -j ACCEPT
iptables -C INPUT -p udp --dport 4500 -j ACCEPT 2>/dev/null \
  || iptables -A INPUT -p udp --dport 4500 -j ACCEPT
iptables -C INPUT -p tcp --dport 8080 -j ACCEPT 2>/dev/null \
  || iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# 持久化
mkdir -p /etc/iptables
iptables-save > /etc/iptables/rules.v4
netfilter-persistent save 2>/dev/null || true

ok "网络配置完成"

# ── 8. 数据库 + 后端服务 ───────────────────────────────────────────────────────
log "[8/8] 配置数据库和后端服务..."

# MySQL（宝塔环境）
MYSQL_CMD=""
for p in /www/server/mysql/bin/mysql /usr/bin/mysql /usr/local/mysql/bin/mysql; do
  [ -x "$p" ] && MYSQL_CMD="$p" && break
done

if [ -n "$MYSQL_CMD" ]; then
  $MYSQL_CMD -uroot -e "CREATE DATABASE IF NOT EXISTS vpnlocation CHARACTER SET utf8mb4;" 2>/dev/null || true
  $MYSQL_CMD -uroot -e "CREATE USER IF NOT EXISTS 'vpnlocation'@'localhost' IDENTIFIED BY '$DB_PASS';" 2>/dev/null || true
  $MYSQL_CMD -uroot -e "GRANT ALL ON vpnlocation.* TO 'vpnlocation'@'localhost';" 2>/dev/null || true
  $MYSQL_CMD -uroot -e "FLUSH PRIVILEGES;" 2>/dev/null || true
  ok "数据库初始化完成"
else
  warn "未找到 MySQL，请在宝塔面板创建数据库后重启服务"
  warn "  数据库名: vpnlocation  用户: vpnlocation  密码: $DB_PASS"
fi

# systemd 服务
cat > /etc/systemd/system/vpn-location.service << EOF
[Unit]
Description=VPN Location Server
After=network.target

[Service]
User=root
WorkingDirectory=$DEPLOY_DIR
ExecStart=$DEPLOY_DIR/vpn-server
Restart=always
RestartSec=5
Environment=FRONTEND_DIST=$DEPLOY_DIR/frontend/dist

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable vpn-location
systemctl restart vpn-location
sleep 3

if systemctl is-active --quiet vpn-location; then
  ok "后端服务运行中"
else
  warn "后端启动失败，日志:"
  journalctl -u vpn-location -n 30 --no-pager
fi

# ── 完成 ──────────────────────────────────────────────────────────────────────
echo ""
echo "========================================"
echo " 部署完成！"
echo ""
echo " Web 面板 : http://$SERVER_IP:8080"
echo " VPN 地址 : $SERVER_IP (UDP 500/4500)"
echo " VPN 账号 : vpnuser / vpnpass2024"
echo ""
echo " iOS 安装步骤:"
echo "  1. Safari 打开 http://$SERVER_IP:8080"
echo "  2. 点击「下载 VPN 配置」"
echo "  3. 设置 → 通用 → VPN与设备管理 → 安装描述文件"
echo "  4. 设置 → 通用 → 关于本机 → 证书信任 → 信任新 CA"
echo "  5. 连接 VPN"
echo ""
echo " 验证命令:"
echo "  ipsec statusall"
echo "  systemctl status vpn-location"
echo "========================================"
