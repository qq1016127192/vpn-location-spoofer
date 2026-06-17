#!/bin/bash
# VPN 部署全面诊断脚本
# 在服务器上执行: bash check-deploy.sh

DEPLOY_DIR="/opt/vpn-location"
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

ok()   { echo -e "${GREEN}[✓]${NC} $*"; }
fail() { echo -e "${RED}[✗]${NC} $*"; }
warn() { echo -e "${YELLOW}[!]${NC} $*"; }
info() { echo -e "${BLUE}[i]${NC} $*"; }
section() { echo ""; echo -e "${BLUE}━━━ $* ━━━${NC}"; }

echo "========================================"
echo " VPN 部署全面诊断"
echo " $(date)"
echo "========================================"

# ── 网络环境 ──────────────────────────────────────────────────────────────────
section "网络环境"
OUTIF=$(ip route show default | awk '{print $5}' | head -1)
INTERNAL_IP=$(ip addr show "$OUTIF" 2>/dev/null | grep 'inet ' | awk '{print $2}' | cut -d/ -f1)
PUBLIC_IP=$(curl -s --max-time 5 ifconfig.me 2>/dev/null || curl -s --max-time 5 api.ipify.org)
info "出口网卡: $OUTIF"
info "内网 IP:  $INTERNAL_IP"
info "公网 IP:  $PUBLIC_IP"
[ "$INTERNAL_IP" != "$PUBLIC_IP" ] && warn "NAT 环境（内网 IP ≠ 公网 IP）"

# ── ipsec.conf ────────────────────────────────────────────────────────────────
section "/etc/ipsec.conf"
if [ ! -f /etc/ipsec.conf ]; then
    fail "文件不存在"
else
    LEFT=$(grep -E "^\s*left=" /etc/ipsec.conf | head -1 | tr -d ' ' | cut -d= -f2)
    LEFTID=$(grep -E "^\s*leftid=" /etc/ipsec.conf | head -1 | tr -d ' ' | cut -d= -f2)
    FORCE=$(grep -E "^\s*forceencaps" /etc/ipsec.conf | head -1 | tr -d ' ')
    RIGHTAUTH=$(grep -E "^\s*rightauth" /etc/ipsec.conf | head -1 | tr -d ' ')
    RIGHTIP=$(grep -E "^\s*rightsourceip" /etc/ipsec.conf | head -1 | tr -d ' ')

    [ "$LEFT" = "$INTERNAL_IP" ] && ok "left=$LEFT" || fail "left=$LEFT (应为 $INTERNAL_IP)"
    [ "$LEFTID" = "$PUBLIC_IP" ] && ok "leftid=$LEFTID" || fail "leftid=$LEFTID (应为 $PUBLIC_IP)"
    echo "$FORCE" | grep -q "yes" && ok "forceencaps=yes" || fail "forceencaps 未设为 yes（NAT 环境必须）"
    ok "$RIGHTAUTH"
    ok "$RIGHTIP"
    info "完整内容:"
    cat /etc/ipsec.conf | grep -v "^#\|^$" | sed 's/^/  /'
fi

# ── ipsec.secrets ─────────────────────────────────────────────────────────────
section "/etc/ipsec.secrets"
if [ ! -f /etc/ipsec.secrets ]; then
    fail "文件不存在"
else
    grep -q "RSA serverKey.pem" /etc/ipsec.secrets && ok "RSA 私钥配置存在" || fail "缺少 RSA serverKey.pem"
    grep -q "EAP" /etc/ipsec.secrets && ok "EAP 用户凭据存在" || fail "缺少 EAP 凭据"
    info "内容:"
    cat /etc/ipsec.secrets | grep -v "^#\|^$" | sed 's/^/  /'
fi

# ── strongswan.conf ───────────────────────────────────────────────────────────
section "/etc/strongswan.conf"
if [ ! -f /etc/strongswan.conf ]; then
    fail "文件不存在"
else
    grep -q "cookie_threshold" /etc/strongswan.conf && ok "cookie_threshold 已配置" || warn "未配置 cookie_threshold"
    info "内容:"
    cat /etc/strongswan.conf | grep -v "^#\|^$" | sed 's/^/  /'
fi

# ── 证书文件 ──────────────────────────────────────────────────────────────────
section "证书文件"

check_cert() {
    local label=$1 path=$2
    if [ ! -f "$path" ]; then
        fail "$label: 文件不存在 ($path)"
        return
    fi
    local subject serial dates
    subject=$(openssl x509 -in "$path" -noout -subject 2>/dev/null | sed 's/subject=//')
    serial=$(openssl x509 -in "$path" -noout -serial 2>/dev/null | cut -d= -f2 | head -c 16)
    dates=$(openssl x509 -in "$path" -noout -dates 2>/dev/null | tr '\n' ' ')
    ok "$label: $subject"
    info "  serial: $serial... | $dates"
}

check_cert "StrongSwan CA    " /etc/ipsec.d/cacerts/caCert.pem
check_cert "StrongSwan Cert  " /etc/ipsec.d/certs/serverCert.pem
check_cert "后端 CA          " $DEPLOY_DIR/certs/caCert.pem

# CA 一致性
CA1=$(openssl x509 -in /etc/ipsec.d/cacerts/caCert.pem -noout -fingerprint -sha1 2>/dev/null | cut -d= -f2)
CA2=$(openssl x509 -in $DEPLOY_DIR/certs/caCert.pem -noout -fingerprint -sha1 2>/dev/null | cut -d= -f2)
[ "$CA1" = "$CA2" ] && ok "两套 CA 指纹一致: $CA1" || fail "CA 不一致! StrongSwan=$CA1 后端=$CA2"

# 服务器证书验证
if openssl verify -CAfile /etc/ipsec.d/cacerts/caCert.pem /etc/ipsec.d/certs/serverCert.pem 2>/dev/null | grep -q OK; then
    ok "服务器证书由当前 CA 签名 ✓"
else
    fail "服务器证书验证失败（证书和 CA 不匹配）"
fi

# SAN 检查
SAN=$(openssl x509 -in /etc/ipsec.d/certs/serverCert.pem -noout -text 2>/dev/null | grep "IP Address:")
[ -n "$SAN" ] && ok "服务器证书含 SAN: $SAN" || fail "服务器证书缺少 SAN（iOS 16+ 必须）"

# 私钥检查
[ -f /etc/ipsec.d/private/serverKey.pem ] && ok "服务器私钥存在" || fail "服务器私钥不存在"
[ -f /etc/ipsec.d/private/caKey.pem ] && warn "caKey.pem 在 ipsec.d/private 中（会被误读，建议删除）" || ok "ipsec.d/private 中无多余的 caKey"

# ── StrongSwan 运行状态 ───────────────────────────────────────────────────────
section "StrongSwan 运行状态"
if ! systemctl is-active --quiet strongswan-starter 2>/dev/null && ! systemctl is-active --quiet strongswan 2>/dev/null; then
    fail "StrongSwan 未运行"
else
    ok "StrongSwan 运行中"
    ipsec statusall 2>/dev/null | grep -E "uptime|Listening|ikev2-eap:|local:|remote:|child:" | sed 's/^/  /'
fi

# ── 防火墙 ────────────────────────────────────────────────────────────────────
section "防火墙规则"
iptables -L INPUT -n --line-numbers 2>/dev/null | grep -E "500|4500|8080|DROP|ACCEPT" | head -15 | sed 's/^/  /'
echo ""
iptables -t nat -L POSTROUTING -n 2>/dev/null | grep -v "^Chain\|^target" | sed 's/^/  NAT: /'

# ── 后端服务 ──────────────────────────────────────────────────────────────────
section "后端服务"
if systemctl is-active --quiet vpn-location; then
    ok "后端服务运行中"
    SVR=$(grep "server_address" $DEPLOY_DIR/config.yaml 2>/dev/null | awk '{print $2}')
    CA_PATH=$(grep "root_ca_path" $DEPLOY_DIR/config.yaml 2>/dev/null | awk '{print $2}')
    info "server_address: $SVR"
    info "root_ca_path:   $CA_PATH"
    [ -f "$CA_PATH" ] && ok "CA 文件存在" || fail "CA 文件不存在: $CA_PATH"
    # 测试 HTTP
    HTTP=$(curl -s --max-time 3 http://localhost:8080/health 2>/dev/null)
    [ -n "$HTTP" ] && ok "HTTP :8080 响应: $HTTP" || fail "HTTP :8080 无响应"
    # 测试 mobileconfig 下载
    MC=$(curl -s --max-time 5 http://localhost:8080/api/vpn/download 2>/dev/null | head -c 50)
    echo "$MC" | grep -q "plist" && ok "mobileconfig 下载正常" || warn "mobileconfig 下载异常: $MC"
else
    fail "后端服务未运行"
    journalctl -u vpn-location -n 10 --no-pager 2>/dev/null | sed 's/^/  /'
fi

# ── 最近连接日志 ──────────────────────────────────────────────────────────────
section "最近 IKE 连接日志（最后30条）"
journalctl -u strongswan-starter -n 50 --no-pager 2>/dev/null \
    | grep -E "initiating|IKE_AUTH|ESTABLISHED|DESTROY|COOKIE|error|fail|certificate|EAP" \
    | tail -30 | sed 's/^/  /'

echo ""
echo "========================================"
echo " 诊断完成"
echo "========================================"
