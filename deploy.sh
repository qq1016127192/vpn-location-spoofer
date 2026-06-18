#!/bin/bash
# VPN 虚拟定位系统 - 一键部署脚本（本地 Mac → Linux VPS）
#
# 用法:
#   SERVER_IP=1.2.3.4 bash deploy.sh
#
# 可选参数:
#   SERVER_DOMAIN=xxx.com     域名（可选，用于 server_address、leftid、证书 SAN）
#   SERVER_USER=root          SSH 用户（默认 root）
#   SSH_PORT=22               SSH 端口（默认 22）
#   DB_PASS=自定义密码        数据库密码（默认 vpnpass2024）
#   DEPLOY_DIR=/opt/vpn       部署目录（默认 /opt/vpn-location）
#   SSH_KEY=~/.ssh/id_rsa     SSH 私钥路径（可选）
#
# 增量部署控制（设为 1 跳过对应步骤）:
#   SKIP_CERTS=1              跳过证书生成（证书已有且未过期时使用）
#   SKIP_FRONTEND=1           跳过前端构建（只改了后端代码时使用）
#   SKIP_BACKEND=1            跳过后端编译（只改了前端代码时使用）
#   SKIP_STRONGSWAN=1         跳过 StrongSwan 配置和重启
#   SKIP_DB=1                 跳过数据库初始化（已初始化过时使用）
#   SKIP_FW=1                 跳过防火墙配置
#
# 数据保护（设为 1 启用保护）:
#   PRESERVE_CONFIG=1         不覆盖服务器 config.yaml（保护 DB 密码等）
#   PRESERVE_SECRETS=1        不覆盖 ipsec.secrets（保护已注册的 EAP 用户）
#   PRESERVE_USERS=1          不覆盖 data/users.json（保护已注册的 Web 用户）
#
# 示例:
#   # 全量部署（首次）
#   SERVER_IP=116.62.147.43 SERVER_DOMAIN=map.iosr.cn bash deploy.sh
#
#   # 只更新后端二进制，不碰证书/前端/用户数据
#   SERVER_IP=116.62.147.43 SERVER_DOMAIN=map.iosr.cn \
#     SKIP_CERTS=1 SKIP_FRONTEND=1 PRESERVE_CONFIG=1 PRESERVE_SECRETS=1 PRESERVE_USERS=1 \
#     bash deploy.sh
#
#   # 只更新前端静态文件
#   SERVER_IP=116.62.147.43 SERVER_DOMAIN=map.iosr.cn \
#     SKIP_CERTS=1 SKIP_BACKEND=1 PRESERVE_CONFIG=1 PRESERVE_SECRETS=1 PRESERVE_USERS=1 \
#     bash deploy.sh
#
#   # 只更新 StrongSwan 配置（不重编译）
#   SERVER_IP=116.62.147.43 SERVER_DOMAIN=map.iosr.cn \
#     SKIP_CERTS=1 SKIP_FRONTEND=1 SKIP_BACKEND=1 PRESERVE_CONFIG=1 PRESERVE_USERS=1 \
#     bash deploy.sh

set -e

# ── 参数 ─────────────────────────────────────────────────────────────────────
SERVER_IP="${SERVER_IP:-}"
SERVER_DOMAIN="${SERVER_DOMAIN:-}"
SERVER_USER="${SERVER_USER:-root}"
SSH_PORT="${SSH_PORT:-22}"
DB_PASS="${DB_PASS:-vpnpass2024}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/vpn-location}"

# 跳过控制（默认 0 = 不跳过）
SKIP_CERTS="${SKIP_CERTS:-0}"
SKIP_FRONTEND="${SKIP_FRONTEND:-0}"
SKIP_BACKEND="${SKIP_BACKEND:-0}"
SKIP_STRONGSWAN="${SKIP_STRONGSWAN:-0}"
SKIP_DB="${SKIP_DB:-0}"
SKIP_FW="${SKIP_FW:-0}"

# 数据保护（默认 0 = 不保护，全量覆盖）
PRESERVE_CONFIG="${PRESERVE_CONFIG:-0}"
PRESERVE_SECRETS="${PRESERVE_SECRETS:-0}"
PRESERVE_USERS="${PRESERVE_USERS:-0}"

if [ -z "$SERVER_IP" ]; then
  echo "用法: SERVER_IP=1.2.3.4 bash deploy.sh"
  echo ""
  echo "示例:"
  echo "  # 全量部署"
  echo "  SERVER_IP=116.62.147.43 SERVER_DOMAIN=map.iosr.cn bash deploy.sh"
  echo ""
  echo "  # 只更新后端二进制"
  echo "  SERVER_IP=116.62.147.43 SERVER_DOMAIN=map.iosr.cn \\"
  echo "    SKIP_CERTS=1 SKIP_FRONTEND=1 PRESERVE_CONFIG=1 PRESERVE_SECRETS=1 PRESERVE_USERS=1 \\"
  echo "    bash deploy.sh"
  exit 1
fi

# server_address 优先用域名
VPN_SERVER_ADDR="${SERVER_DOMAIN:-$SERVER_IP}"

SSH_OPTS="-p $SSH_PORT -o StrictHostKeyChecking=no -o ConnectTimeout=15"
[ -n "$SSH_KEY" ] && SSH_OPTS="$SSH_OPTS -i $SSH_KEY"
SSH="${SERVER_USER}@${SERVER_IP}"

# 部署摘要
echo "========================================"
echo " VPN 虚拟定位系统 - 部署"
echo "========================================"
echo " 目标服务器:   $SERVER_IP"
echo " 服务器域名:   ${SERVER_DOMAIN:-（无，使用 IP）}"
echo " VPN 配置地址: $VPN_SERVER_ADDR"
echo " 部署目录:     $DEPLOY_DIR"
echo ""
echo " 部署模式:"
[ "$SKIP_CERTS" = "1" ]      && echo "   证书:       跳过（使用现有证书）" || echo "   证书:       重新生成"
[ "$SKIP_FRONTEND" = "1" ]   && echo "   前端:       跳过（使用现有前端）" || echo "   前端:       重新构建"
[ "$SKIP_BACKEND" = "1" ]    && echo "   后端:       跳过（使用现有二进制）" || echo "   后端:       重新编译"
[ "$SKIP_STRONGSWAN" = "1" ] && echo "   StrongSwan: 跳过" || echo "   StrongSwan: 重新配置"
[ "$SKIP_DB" = "1" ]         && echo "   数据库:     跳过" || echo "   数据库:     初始化"
echo ""
echo " 数据保护:"
[ "$PRESERVE_CONFIG" = "1" ]  && echo "   config.yaml:    保护（不覆盖服务器配置）" || echo "   config.yaml:    覆盖"
[ "$PRESERVE_SECRETS" = "1" ] && echo "   ipsec.secrets:  保护（不覆盖 EAP 用户）" || echo "   ipsec.secrets:  覆盖"
[ "$PRESERVE_USERS" = "1" ]   && echo "   users.json:     保护（不覆盖 Web 用户）" || echo "   users.json:     覆盖"
echo "========================================"
echo ""

# ── [1/6] 生成证书 ────────────────────────────────────────────────────────────
if [ "$SKIP_CERTS" = "1" ]; then
  echo "=== [1/6] 证书：跳过 ==="
  # 检查本地证书是否存在
  if [ ! -f vpn-config/caCert.pem ] || [ ! -f vpn-config/serverCert.pem ]; then
    echo "错误: 本地证书文件不存在，无法跳过证书生成"
    echo "请先运行一次完整部署生成证书，或去掉 SKIP_CERTS=1"
    exit 1
  fi
else
  echo "=== [1/6] 生成证书（SERVER_IP=$SERVER_IP, SERVER_DOMAIN=${SERVER_DOMAIN:-无}）==="
  SERVER_IP=$SERVER_IP SERVER_DOMAIN=$SERVER_DOMAIN bash vpn-config/generate-certs.sh
  echo "证书生成完成"
fi

# ── [2/6] 构建前端 ────────────────────────────────────────────────────────────
if [ "$SKIP_FRONTEND" = "1" ]; then
  echo "=== [2/6] 前端构建：跳过 ==="
  if [ ! -d frontend/dist ] || [ ! -f frontend/dist/index.html ]; then
    echo "错误: frontend/dist 不存在，无法跳过前端构建"
    echo "请先运行一次完整部署构建前端，或去掉 SKIP_FRONTEND=1"
    exit 1
  fi
else
  echo "=== [2/6] 构建前端 ==="
  cd frontend
  npm install --silent
  npm run build
  cd ..
  echo "前端构建完成 → frontend/dist"
fi

# ── [3/6] 交叉编译后端 ────────────────────────────────────────────────────────
if [ "$SKIP_BACKEND" = "1" ]; then
  echo "=== [3/6] 后端编译：跳过 ==="
  if [ ! -f backend/vpn-server-linux ]; then
    echo "错误: backend/vpn-server-linux 不存在，无法跳过后端编译"
    echo "请先运行一次完整部署编译后端，或去掉 SKIP_BACKEND=1"
    exit 1
  fi
else
  echo "=== [3/6] 交叉编译后端（Linux amd64 静态链接）==="
  cd backend
  cp config.yaml config.yaml.bak

  # 替换 Linux 服务器上的配置路径和参数
  sed -i.tmp \
    -e "s|server_address:.*|server_address: $VPN_SERVER_ADDR|" \
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
fi

# ── [4/6] 打包上传 ────────────────────────────────────────────────────────────
echo ""
echo "=== [4/6] 打包上传 ==="

# 生成 ipsec.conf：left= 用 IP，leftid= 用域名（如果有）
sed -e "s/left=192\.168\.10\.7/left=$SERVER_IP/g" \
  vpn-config/ipsec.conf > /tmp/ipsec-deploy.conf

# leftid=：优先域名，无域名则用 IP
if [ -n "$SERVER_DOMAIN" ]; then
  sed -i "s/leftid=map\.iosr\.cn/leftid=$SERVER_DOMAIN/g" /tmp/ipsec-deploy.conf
else
  sed -i "s/leftid=map\.iosr\.cn/leftid=$SERVER_IP/g" /tmp/ipsec-deploy.conf
fi

echo "ipsec.conf leftid 预览:"
grep "leftid" /tmp/ipsec-deploy.conf

# 构建 tar 包（只包含需要的文件）
TAR_FILES=""
[ "$SKIP_BACKEND" != "1" ]      && TAR_FILES="$TAR_FILES backend/vpn-server-linux backend/config.yaml"
[ "$SKIP_FRONTEND" != "1" ]     && TAR_FILES="$TAR_FILES frontend/dist"
[ "$SKIP_CERTS" != "1" ]        && TAR_FILES="$TAR_FILES vpn-config/caCert.pem vpn-config/caKey.pem vpn-config/serverCert.pem vpn-config/serverKey.pem"
[ "$PRESERVE_SECRETS" != "1" ]  && TAR_FILES="$TAR_FILES vpn-config/ipsec.secrets"

# 至少要有二进制或前端
if [ -z "$TAR_FILES" ]; then
  echo "所有构建步骤均已跳过，仅上传配置变更"
  # 最少上传二进制
  TAR_FILES="backend/vpn-server-linux"
  [ -f frontend/dist/index.html ] && TAR_FILES="$TAR_FILES frontend/dist"
fi

# 如果需要部署证书，也加入 tar 包
if [ "$SKIP_CERTS" != "1" ]; then
  TAR_FILES="$TAR_FILES vpn-config/caCert.pem vpn-config/caKey.pem vpn-config/serverCert.pem vpn-config/serverKey.pem"
fi

# 去重
TAR_FILES=$(echo $TAR_FILES | tr ' ' '\n' | sort -u | tr '\n' ' ')

echo "打包文件: $TAR_FILES"
tar -czf /tmp/vpn-deploy.tar.gz $TAR_FILES

# 上传
scp $SSH_OPTS /tmp/vpn-deploy.tar.gz $SSH:/tmp/vpn-deploy.tar.gz
scp $SSH_OPTS /tmp/ipsec-deploy.conf $SSH:/tmp/ipsec.conf
echo "上传完成"

# ── [5/6] 远程部署 ────────────────────────────────────────────────────────────
echo ""
echo "=== [5/6] 远程部署 ==="

ssh $SSH_OPTS $SSH bash << REMOTE
set -e
DEPLOY_DIR="$DEPLOY_DIR"
SERVER_IP="$SERVER_IP"
SERVER_DOMAIN="$SERVER_DOMAIN"
VPN_SERVER_ADDR="$VPN_SERVER_ADDR"
DB_PASS="$DB_PASS"
SKIP_CERTS="$SKIP_CERTS"
SKIP_FRONTEND="$SKIP_FRONTEND"
SKIP_BACKEND="$SKIP_BACKEND"
SKIP_STRONGSWAN="$SKIP_STRONGSWAN"
SKIP_DB="$SKIP_DB"
SKIP_FW="$SKIP_FW"
PRESERVE_CONFIG="$PRESERVE_CONFIG"
PRESERVE_SECRETS="$PRESERVE_SECRETS"
PRESERVE_USERS="$PRESERVE_USERS"

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
    apt-get install -y strongswan 2>/dev/null || yum install -y strongswan || true
    IPSEC_BIN=\$(which ipsec 2>/dev/null || echo /usr/sbin/ipsec)
    STRONGSWAN_SERVICE=strongswan
    ;;
esac
\$IPSEC_BIN version | head -1
log "StrongSwan 安装完成，服务名: \$STRONGSWAN_SERVICE"

# ── 解压部署包 ─────────────────────────────────────────────────────────────
log "解压部署包..."
mkdir -p \$DEPLOY_DIR/frontend \$DEPLOY_DIR/data
cd /tmp && tar -xzf vpn-deploy.tar.gz

# 保护 users.json：解压前备份，解压后恢复
if [ "\$PRESERVE_USERS" = "1" ]; then
  BACKUP_USERS=""
  if [ -f "\$DEPLOY_DIR/data/users.json" ]; then
    BACKUP_USERS=\$(cat "\$DEPLOY_DIR/data/users.json")
    log "✓ 已备份 users.json"
  fi
fi

# 保护 config.yaml：解压前备份
if [ "\$PRESERVE_CONFIG" = "1" ]; then
  BACKUP_CONFIG=""
  if [ -f "\$DEPLOY_DIR/config.yaml" ]; then
    BACKUP_CONFIG=\$(cat "\$DEPLOY_DIR/config.yaml")
    log "✓ 已备份 config.yaml（含数据库凭据）"
  fi
fi

# 保护 ipsec.secrets：解压前备份
if [ "\$PRESERVE_SECRETS" = "1" ]; then
  BACKUP_SECRETS=""
  if [ -f "/etc/ipsec.secrets" ]; then
    BACKUP_SECRETS=\$(cat "/etc/ipsec.secrets")
    log "✓ 已备份 ipsec.secrets（含 EAP 用户凭据）"
  fi
fi

# ── 部署二进制 ─────────────────────────────────────────────────────────────
if [ "\$SKIP_BACKEND" != "1" ] && [ -f "backend/vpn-server-linux" ]; then
  cp backend/vpn-server-linux \$DEPLOY_DIR/vpn-server
  chmod +x \$DEPLOY_DIR/vpn-server
  log "✓ 后端二进制已更新"
fi

# ── 部署前端 ───────────────────────────────────────────────────────────────
if [ "\$SKIP_FRONTEND" != "1" ] && [ -d "frontend/dist" ]; then
  rm -rf \$DEPLOY_DIR/frontend/dist
  cp -r frontend/dist \$DEPLOY_DIR/frontend/dist
  log "✓ 前端文件已更新"
fi

# ── 恢复 config.yaml ──────────────────────────────────────────────────────
if [ "\$PRESERVE_CONFIG" = "1" ] && [ -n "\$BACKUP_CONFIG" ]; then
  echo "\$BACKUP_CONFIG" > \$DEPLOY_DIR/config.yaml
  log "✓ 已恢复 config.yaml（数据库凭据等未被覆盖）"
elif [ -f "backend/config.yaml" ]; then
  cp backend/config.yaml \$DEPLOY_DIR/config.yaml
  log "✓ config.yaml 已更新（PRESERVE_CONFIG 未启用）"
fi

# ── 恢复 users.json ───────────────────────────────────────────────────────
if [ "\$PRESERVE_USERS" = "1" ] && [ -n "\$BACKUP_USERS" ]; then
  mkdir -p \$DEPLOY_DIR/data
  echo "\$BACKUP_USERS" > \$DEPLOY_DIR/data/users.json
  log "✓ 已恢复 users.json（用户账号未被覆盖）"
fi

# ── 部署证书 ───────────────────────────────────────────────────────────────
if [ "\$SKIP_CERTS" != "1" ]; then
  log "部署证书..."
  mkdir -p /etc/ipsec.d/{certs,private,cacerts}

  [ -f vpn-config/serverCert.pem ] && cp vpn-config/serverCert.pem /etc/ipsec.d/certs/serverCert.pem
  [ -f vpn-config/caCert.pem ]     && cp vpn-config/caCert.pem     /etc/ipsec.d/cacerts/caCert.pem
  [ -f vpn-config/serverKey.pem ]  && cp vpn-config/serverKey.pem  /etc/ipsec.d/private/serverKey.pem

  mkdir -p \$DEPLOY_DIR/certs
  [ -f vpn-config/caCert.pem ] && cp vpn-config/caCert.pem \$DEPLOY_DIR/certs/caCert.pem
  [ -f vpn-config/caKey.pem ]  && cp vpn-config/caKey.pem  \$DEPLOY_DIR/certs/caKey.pem
  chmod 600 /etc/ipsec.d/private/serverKey.pem \$DEPLOY_DIR/certs/caKey.pem 2>/dev/null || true
  chmod 644 /etc/ipsec.d/certs/serverCert.pem /etc/ipsec.d/cacerts/caCert.pem 2>/dev/null || true

  log "证书部署完成"
else
  log "证书部署：跳过"
fi

# ── 更新 config.yaml 证书路径（始终执行，保证路径正确）─────────────────────
sed -i \
  -e "s|root_ca_path:.*|root_ca_path: \$DEPLOY_DIR/certs/caCert.pem|" \
  -e "s|root_key_path:.*|root_key_path: \$DEPLOY_DIR/certs/caKey.pem|" \
  \$DEPLOY_DIR/config.yaml 2>/dev/null || true

# ── 恢复 ipsec.secrets ────────────────────────────────────────────────────
if [ "\$PRESERVE_SECRETS" = "1" ] && [ -n "\$BACKUP_SECRETS" ]; then
  echo "\$BACKUP_SECRETS" > /etc/ipsec.secrets
  chmod 600 /etc/ipsec.secrets
  log "✓ 已恢复 ipsec.secrets（EAP 用户凭据未被覆盖）"
elif [ -f "vpn-config/ipsec.secrets" ]; then
  cp vpn-config/ipsec.secrets /etc/ipsec.secrets
  chmod 600 /etc/ipsec.secrets
  log "✓ ipsec.secrets 已更新（PRESERVE_SECRETS 未启用）"
fi

# ── 部署 StrongSwan 配置 ───────────────────────────────────────────────────
if [ "\$SKIP_STRONGSWAN" != "1" ]; then
  log "部署 StrongSwan 配置..."
  cp /tmp/ipsec.conf /etc/ipsec.conf

  # charon.conf
  CHARON_CONF=\$(find /etc/strongswan.d /etc/strongswan -name "charon.conf" 2>/dev/null | head -1)
  if [ -n "\$CHARON_CONF" ]; then
    cat > "\$CHARON_CONF" << 'CHARONEOF'
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
    log "charon.conf 已配置"
  fi
else
  log "StrongSwan 配置：跳过"
fi

# ── IP 转发 + NAT（幂等，始终执行）─────────────────────────────────────────
log "配置 IP 转发和 NAT..."
grep -q "^net.ipv4.ip_forward" /etc/sysctl.conf \
  && sed -i 's|^net.ipv4.ip_forward.*|net.ipv4.ip_forward = 1|' /etc/sysctl.conf \
  || echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf
sysctl -w net.ipv4.ip_forward=1

OUTIF=\$(ip route show default 2>/dev/null | awk '/^default/{print \$5}' | head -1)
[ -z "\$OUTIF" ] && OUTIF=\$(ip route | awk '/default/{print \$5}' | head -1)
log "出口网卡: \$OUTIF"

iptables -t nat -C POSTROUTING -s 10.10.0.0/16 -o "\$OUTIF" -j MASQUERADE 2>/dev/null || \
  iptables -t nat -A POSTROUTING -s 10.10.0.0/16 -o "\$OUTIF" -j MASQUERADE
iptables -C FORWARD -s 10.10.0.0/16 -j ACCEPT 2>/dev/null || \
  iptables -A FORWARD -s 10.10.0.0/16 -j ACCEPT
iptables -C FORWARD -d 10.10.0.0/16 -j ACCEPT 2>/dev/null || \
  iptables -A FORWARD -d 10.10.0.0/16 -j ACCEPT

mkdir -p /etc/iptables
iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
command -v netfilter-persistent &>/dev/null && netfilter-persistent save 2>/dev/null || true
log "NAT 规则已配置"

# ── 防火墙 ─────────────────────────────────────────────────────────────────
if [ "\$SKIP_FW" != "1" ]; then
  log "配置防火墙..."
  if command -v ufw &>/dev/null && ufw status | grep -q "Status: active"; then
    ufw allow 500/udp  2>/dev/null || true
    ufw allow 4500/udp 2>/dev/null || true
    ufw allow 8080/tcp 2>/dev/null || true
    ufw reload 2>/dev/null || true
    log "ufw 规则已添加"
  elif command -v firewall-cmd &>/dev/null; then
    firewall-cmd --permanent --add-service=ipsec 2>/dev/null || true
    firewall-cmd --permanent --add-port=4500/udp  2>/dev/null || true
    firewall-cmd --permanent --add-port=8080/tcp  2>/dev/null || true
    firewall-cmd --reload 2>/dev/null || true
    log "firewalld 规则已添加"
  fi
else
  log "防火墙配置：跳过"
fi

# ── 重启服务 ───────────────────────────────────────────────────────────────
# StrongSwan 重启
if [ "\$SKIP_STRONGSWAN" != "1" ]; then
  log "重启 StrongSwan..."
  systemctl enable "\$STRONGSWAN_SERVICE" 2>/dev/null || true
  systemctl restart "\$STRONGSWAN_SERVICE" 2>/dev/null || \
    systemctl restart strongswan 2>/dev/null || \
    \$IPSEC_BIN restart
  sleep 3

  if \$IPSEC_BIN statusall 2>/dev/null | grep -q "ikev2-eap"; then
    log "✓ StrongSwan 已加载 ikev2-eap 连接配置"
  else
    log "⚠ StrongSwan 配置加载异常:"
    \$IPSEC_BIN statusall 2>/dev/null | head -10 || true
  fi
fi

# 后端服务重启（仅在更新了二进制时）
if [ "\$SKIP_BACKEND" != "1" ]; then
  log "重启后端服务..."
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
    log "✗ 后端启动失败，日志:"
    journalctl -u vpn-location -n 30 --no-pager
    exit 1
  fi
else
  log "后端服务：跳过（未更新二进制）"
  # 检查是否在运行
  if ! systemctl is-active --quiet vpn-location 2>/dev/null; then
    log "⚠ 后端服务未运行，尝试启动..."
    systemctl start vpn-location 2>/dev/null || true
  fi
fi

# ── 数据库初始化 ───────────────────────────────────────────────────────────
if [ "\$SKIP_DB" != "1" ]; then
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

    # 只在 PRESERVE_CONFIG 未启用时更新密码
    if [ "\$PRESERVE_CONFIG" != "1" ]; then
      sed -i \
        -e "s|username:.*|username: vpnlocation|" \
        -e "s|password:.*|password: \$DB_PASS|" \
        -e "s|database:.*|database: vpnlocation|" \
        \$DEPLOY_DIR/config.yaml
    fi
    log "✓ 数据库初始化完成"
  else
    log "未找到 MySQL，跳过"
  fi
else
  log "数据库初始化：跳过"
fi

# ── 最终验证 ───────────────────────────────────────────────────────────────
echo ""
echo "========================================"
echo " 部署完成！"
echo ""
if [ -n "\$SERVER_DOMAIN" ]; then
  echo " Web 面板: http://\$VPN_SERVER_ADDR:8080"
  echo " VPN 地址: \$VPN_SERVER_ADDR (UDP 500 / 4500)"
  echo " VPN IP:   \$SERVER_IP"
else
  echo " Web 面板: http://\$SERVER_IP:8080"
  echo " VPN 地址: \$SERVER_IP (UDP 500 / 4500)"
fi
echo ""
echo " iOS 安装描述文件:"
echo "   Safari 打开 http://\$VPN_SERVER_ADDR:8080"
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
# 恢复 config.yaml（如果被修改过）
[ -f backend/config.yaml.bak ] && mv backend/config.yaml.bak backend/config.yaml 2>/dev/null || true
echo "本地临时文件已清理，源文件已恢复"