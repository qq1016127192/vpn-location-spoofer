#!/bin/bash
# VPN 虚拟定位系统 - 一键部署脚本（本地 Mac → Linux VPS）
#
# 用法:
#   SERVER_IP=1.2.3.4 bash deploy.sh
#
# 可选参数:
#   SERVER_USER=root          SSH 用户（默认 root）
#   SSH_PORT=22               SSH 端口（默认 22）
#   DB_PASS=自定义密码        数据库密码（默认 vpnpass2024）
#   DEPLOY_DIR=/opt/vpn       部署目录（默认 /opt/vpn-location）
#   SSH_KEY=~/.ssh/id_rsa     SSH 私钥路径（可选）

set -e

# ── 参数 ─────────────────────────────────────────────────────────────────────
SERVER_IP="${SERVER_IP:-}"
SERVER_USER="${SERVER_USER:-root}"
SSH_PORT="${SSH_PORT:-22}"
DB_PASS="${DB_PASS:-vpnpass2024}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/vpn-location}"

if [ -z "$SERVER_IP" ]; then
  echo "用法: SERVER_IP=1.2.3.4 bash deploy.sh"
  exit 1
fi

SSH_OPTS="-p $SSH_PORT -o StrictHostKeyChecking=no -o ConnectTimeout=15"
[ -n "$SSH_KEY" ] && SSH_OPTS="$SSH_OPTS -i $SSH_KEY"
SSH="${SERVER_USER}@${SERVER_IP}"

echo "========================================"
echo " 目标服务器: $SERVER_IP"
echo " 部署目录:   $DEPLOY_DIR"
echo "========================================"

# ── [1/5] 生成证书 ────────────────────────────────────────────────────────────
echo ""
echo "=== [1/5] 生成证书（SERVER_IP=$SERVER_IP）==="
SERVER_IP=$SERVER_IP bash vpn-config/generate-certs.sh
echo "证书生成完成"

# ── [2/5] 构建前端 ────────────────────────────────────────────────────────────
echo ""
echo "=== [2/5] 构建前端 ==="
cd frontend
npm install --silent
npm run build
cd ..
echo "前端构建完成 → frontend/dist"

# ── [3/5] 交叉编译后端（Linux amd64 静态链接）────────────────────────────────
echo ""
echo "=== [3/5] 交叉编译后端 ==="
cd backend
cp config.yaml config.yaml.bak

# 替换 Linux 服务器上的配置路径和参数
sed -i.tmp \
  -e "s|server_address:.*|server_address: $SERVER_IP|" \
  -e "s|root_ca_path:.*|root_ca_path: /etc/ipsec.d/cacerts/caCert.pem|" \
  -e "s|root_key_path:.*|root_key_path: /etc/ipsec.d/private/caKey.pem|" \
  -e "s|ipsec_bin:.*|ipsec_bin: /usr/sbin/ipsec|" \
  -e "s|ipsec_secrets:.*|ipsec_secrets: /etc/ipsec.secrets|" \
  -e "s|mode:.*debug.*|mode: release|" \
  config.yaml
rm -f config.yaml.tmp

echo "config.yaml 关键配置预览:"
grep -E "server_address|root_ca|root_key|ipsec_bin|mode" config.yaml

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-extldflags=-static -s -w" -o vpn-server-linux .
cd ..
echo "编译完成 → backend/vpn-server-linux"

# ── [4/5] 打包上传 ────────────────────────────────────────────────────────────
echo ""
echo "=== [4/5] 打包上传 ==="

# 生成 ipsec.conf（SERVER_IP 已替换）
sed \
  -e "s/left=192\.168\.10\.7/left=$SERVER_IP/g" \
  -e "s/leftid=192\.168\.10\.7/leftid=$SERVER_IP/g" \
  vpn-config/ipsec.conf > /tmp/ipsec-deploy.conf

tar -czf /tmp/vpn-deploy.tar.gz \
  backend/vpn-server-linux \
  backend/config.yaml \
  frontend/dist \
  vpn-config/caCert.pem \
  vpn-config/caKey.pem \
  vpn-config/serverCert.pem \
  vpn-config/serverKey.pem \
  vpn-config/ipsec.secrets

# 把替换好的 ipsec.conf 单独传（不污染源文件）
scp $SSH_OPTS /tmp/vpn-deploy.tar.gz $SSH:/tmp/vpn-deploy.tar.gz
scp $SSH_OPTS /tmp/ipsec-deploy.conf $SSH:/tmp/ipsec.conf
echo "上传完成"

# ── [5/5] 远程部署 ────────────────────────────────────────────────────────────
echo ""
echo "=== [5/5] 远程部署 ==="

ssh $SSH_OPTS $SSH bash << REMOTE
set -e
DEPLOY_DIR="$DEPLOY_DIR"
SERVER_IP="$SERVER_IP"
DB_PASS="$DB_PASS"

log() { echo "[\$(date '+%H:%M:%S')] \$*"; }

# ── 检测发行版 ─────────────────────────────────────────────────────────────
if [ -f /etc/os-release ]; then
  . /etc/os-release
  DISTRO=\$ID
else
  DISTRO=unknown
fi
log "发行版: \$DISTRO"

# ── 安装 StrongSwan ────────────────────────────────────────────────────────
log "安装 StrongSwan..."
case "\$DISTRO" in
  ubuntu|debian)
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
      strongswan strongswan-pki libcharon-extra-plugins \
      libcharon-extauth-plugins iptables-persistent 2>/dev/null || \
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
      strongswan strongswan-pki libcharon-extra-plugins iptables-persistent
    IPSEC_BIN=/usr/sbin/ipsec
    STRONGSWAN_SERVICE=strongswan-starter
    # Ubuntu 20.04+ 服务名不同
    systemctl list-units --full --all 2>/dev/null | grep -q strongswan-starter || \
      STRONGSWAN_SERVICE=strongswan
    ;;
  centos|rhel|rocky|almalinux)
    yum install -y epel-release 2>/dev/null || true
    yum install -y strongswan iptables-services
    IPSEC_BIN=/usr/sbin/ipsec
    STRONGSWAN_SERVICE=strongswan
    ;;
  *)
    # 尝试通用安装
    apt-get install -y strongswan 2>/dev/null || yum install -y strongswan || true
    IPSEC_BIN=\$(which ipsec 2>/dev/null || echo /usr/sbin/ipsec)
    STRONGSWAN_SERVICE=strongswan
    ;;
esac
\$IPSEC_BIN version | head -1
log "StrongSwan 安装完成，服务名: \$STRONGSWAN_SERVICE"

# ── 解压部署包 ─────────────────────────────────────────────────────────────
log "解压部署包..."
mkdir -p \$DEPLOY_DIR/frontend
cd /tmp && tar -xzf vpn-deploy.tar.gz

cp backend/vpn-server-linux \$DEPLOY_DIR/vpn-server
cp backend/config.yaml      \$DEPLOY_DIR/config.yaml
cp -r frontend/dist         \$DEPLOY_DIR/frontend/dist
chmod +x \$DEPLOY_DIR/vpn-server

# ── 部署证书（注意：caKey.pem 不放 ipsec.d/private，避免 strongSwan 误读）──
log "部署证书..."
mkdir -p /etc/ipsec.d/{certs,private,cacerts}

cp vpn-config/serverCert.pem /etc/ipsec.d/certs/serverCert.pem
cp vpn-config/caCert.pem     /etc/ipsec.d/cacerts/caCert.pem
cp vpn-config/serverKey.pem  /etc/ipsec.d/private/serverKey.pem

# caKey 只给后端服务用，单独存放，不混入 strongSwan 目录
mkdir -p \$DEPLOY_DIR/certs
cp vpn-config/caCert.pem  \$DEPLOY_DIR/certs/caCert.pem
cp vpn-config/caKey.pem   \$DEPLOY_DIR/certs/caKey.pem
chmod 600 /etc/ipsec.d/private/serverKey.pem \$DEPLOY_DIR/certs/caKey.pem
chmod 644 /etc/ipsec.d/certs/serverCert.pem /etc/ipsec.d/cacerts/caCert.pem

# 更新 config.yaml 证书路径指向实际存放位置
sed -i \
  -e "s|root_ca_path:.*|root_ca_path: \$DEPLOY_DIR/certs/caCert.pem|" \
  -e "s|root_key_path:.*|root_key_path: \$DEPLOY_DIR/certs/caKey.pem|" \
  \$DEPLOY_DIR/config.yaml

log "证书部署完成"

# ── 部署 StrongSwan 配置 ───────────────────────────────────────────────────
log "部署 StrongSwan 配置..."
cp /tmp/ipsec.conf            /etc/ipsec.conf
cp vpn-config/ipsec.secrets   /etc/ipsec.secrets
chmod 600 /etc/ipsec.secrets

# 找到并配置 charon.conf（路径因发行版而异）
CHARON_CONF=\$(find /etc/strongswan.d /etc/strongswan -name "charon.conf" 2>/dev/null | head -1)
if [ -n "\$CHARON_CONF" ]; then
  cat > "\$CHARON_CONF" << 'CHARONEOF'
charon {
    # 不限制半开连接数（默认值 10 会拒绝并发连接）
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
  log "charon.conf 已配置: \$CHARON_CONF"
fi

# ── IP 转发 + NAT ──────────────────────────────────────────────────────────
log "配置 IP 转发和 NAT..."
# 持久化 IP 转发
grep -q "^net.ipv4.ip_forward" /etc/sysctl.conf \
  && sed -i 's|^net.ipv4.ip_forward.*|net.ipv4.ip_forward = 1|' /etc/sysctl.conf \
  || echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf
sysctl -w net.ipv4.ip_forward=1

# 找出口网卡
OUTIF=\$(ip route show default 2>/dev/null | awk '/^default/{print \$5}' | head -1)
[ -z "\$OUTIF" ] && OUTIF=\$(ip route | awk '/default/{print \$5}' | head -1)
log "出口网卡: \$OUTIF"

# iptables MASQUERADE（幂等，先检查再添加）
iptables -t nat -C POSTROUTING -s 10.10.0.0/24 -o "\$OUTIF" -j MASQUERADE 2>/dev/null || \
  iptables -t nat -A POSTROUTING -s 10.10.0.0/24 -o "\$OUTIF" -j MASQUERADE
iptables -C FORWARD -s 10.10.0.0/24 -j ACCEPT 2>/dev/null || \
  iptables -A FORWARD -s 10.10.0.0/24 -j ACCEPT
iptables -C FORWARD -d 10.10.0.0/24 -j ACCEPT 2>/dev/null || \
  iptables -A FORWARD -d 10.10.0.0/24 -j ACCEPT

# 持久化 iptables（多发行版兼容）
mkdir -p /etc/iptables
iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
if command -v iptables-save &>/dev/null; then
  iptables-save > /etc/sysconfig/iptables 2>/dev/null || true
fi
# Ubuntu/Debian: netfilter-persistent
if command -v netfilter-persistent &>/dev/null; then
  netfilter-persistent save 2>/dev/null || true
fi
log "NAT 规则已配置并持久化"

# ── 开放防火墙端口 ─────────────────────────────────────────────────────────
log "配置防火墙..."
if command -v ufw &>/dev/null && ufw status | grep -q "Status: active"; then
  ufw allow 500/udp  comment 'IKEv2 VPN'  2>/dev/null || true
  ufw allow 4500/udp comment 'IKEv2 NAT-T' 2>/dev/null || true
  ufw allow 8080/tcp comment 'VPN Web'     2>/dev/null || true
  ufw reload 2>/dev/null || true
  log "ufw 规则已添加"
elif command -v firewall-cmd &>/dev/null; then
  firewall-cmd --permanent --add-service=ipsec 2>/dev/null || true
  firewall-cmd --permanent --add-port=4500/udp  2>/dev/null || true
  firewall-cmd --permanent --add-port=8080/tcp  2>/dev/null || true
  firewall-cmd --reload 2>/dev/null || true
  log "firewalld 规则已添加"
else
  log "未检测到 ufw/firewalld，跳过防火墙配置（请手动开放 500/4500 UDP 和 8080 TCP）"
fi

# ── 启动 StrongSwan ────────────────────────────────────────────────────────
log "启动 StrongSwan..."
systemctl enable "\$STRONGSWAN_SERVICE" 2>/dev/null || true
systemctl restart "\$STRONGSWAN_SERVICE" 2>/dev/null || \
  systemctl restart strongswan 2>/dev/null || \
  \$IPSEC_BIN restart
sleep 3

# 验证
if \$IPSEC_BIN statusall 2>/dev/null | grep -q "ikev2-eap"; then
  log "✓ StrongSwan 已加载 ikev2-eap 连接配置"
  \$IPSEC_BIN statusall 2>/dev/null | grep -E "uptime:|ikev2-eap:" | head -3
else
  log "✗ StrongSwan 配置加载异常，查看状态:"
  \$IPSEC_BIN statusall 2>/dev/null | head -20 || true
fi

# ── 配置数据库 ─────────────────────────────────────────────────────────────
log "初始化数据库..."
MYSQL_CMD=""
for p in /www/server/mysql/bin/mysql /usr/bin/mysql /usr/local/mysql/bin/mysql; do
  [ -x "\$p" ] && MYSQL_CMD="\$p" && break
done

if [ -n "\$MYSQL_CMD" ]; then
  \$MYSQL_CMD -uroot -e "CREATE DATABASE IF NOT EXISTS vpnlocation CHARACTER SET utf8mb4;" 2>/dev/null || true
  \$MYSQL_CMD -uroot -e "CREATE USER IF NOT EXISTS 'vpnlocation'@'localhost' IDENTIFIED BY '\$DB_PASS';" 2>/dev/null || true
  \$MYSQL_CMD -uroot -e "GRANT ALL ON vpnlocation.* TO 'vpnlocation'@'localhost';" 2>/dev/null || true
  \$MYSQL_CMD -uroot -e "FLUSH PRIVILEGES;" 2>/dev/null || true
  sed -i \
    -e "s|username:.*|username: vpnlocation|" \
    -e "s|password:.*|password: \$DB_PASS|" \
    -e "s|database:.*|database: vpnlocation|" \
    \$DEPLOY_DIR/config.yaml
  log "✓ 数据库初始化完成"
else
  log "未找到 MySQL，请手动创建数据库: vpnlocation / 用户: vpnlocation / 密码: \$DB_PASS"
fi

# ── 创建 systemd 服务 ──────────────────────────────────────────────────────
log "创建 systemd 服务..."
cat > /etc/systemd/system/vpn-location.service << SVCEOF
[Unit]
Description=VPN Location Server
After=network.target mysql.service strongswan.service

[Service]
User=root
WorkingDirectory=\$DEPLOY_DIR
ExecStart=\$DEPLOY_DIR/vpn-server
Restart=always
RestartSec=5
Environment=FRONTEND_DIST=\$DEPLOY_DIR/frontend/dist

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable vpn-location
systemctl restart vpn-location
sleep 3

if systemctl is-active --quiet vpn-location; then
  log "✓ 后端服务运行中"
else
  log "✗ 后端启动失败，日志如下:"
  journalctl -u vpn-location -n 30 --no-pager
  exit 1
fi

# ── 最终验证 ───────────────────────────────────────────────────────────────
echo ""
echo "========================================"
echo " 部署完成！"
echo ""
echo " Web 面板: http://\$SERVER_IP:8080"
echo " VPN 地址: \$SERVER_IP (UDP 500 / 4500)"
echo " VPN 账号: vpnuser / vpnpass2024"
echo ""
echo " iOS 安装描述文件:"
echo "   Safari 打开 http://\$SERVER_IP:8080"
echo "   点击「下载 VPN 配置」并安装"
echo ""
echo " 验证命令:"
echo "   ipsec statusall"
echo "   systemctl status vpn-location"
echo "========================================"
REMOTE

# ── 本地清理 ──────────────────────────────────────────────────────────────────
echo ""
echo "=== 本地清理 ==="
rm -f /tmp/vpn-deploy.tar.gz /tmp/ipsec-deploy.conf backend/vpn-server-linux
mv backend/config.yaml.bak backend/config.yaml 2>/dev/null || true
echo "本地临时文件已清理，源文件已恢复"
