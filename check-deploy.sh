#!/bin/bash
# 服务器部署状态诊断脚本
# 在服务器上执行: bash check-deploy.sh

DEPLOY_DIR="/opt/vpn-location"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ok()   { echo -e "${GREEN}[✓]${NC} $*"; }
fail() { echo -e "${RED}[✗]${NC} $*"; }
warn() { echo -e "${YELLOW}[!]${NC} $*"; }

echo "========================================"
echo " VPN 部署状态诊断"
echo "========================================"

# ── 1. StrongSwan 运行状态 ────────────────────────────────────────────────────
echo ""
echo "[ StrongSwan ]"
if systemctl is-active --quiet strongswan-starter 2>/dev/null || systemctl is-active --quiet strongswan 2>/dev/null; then
  ok "StrongSwan 服务运行中"
else
  fail "StrongSwan 服务未运行"
  echo "    → 执行: systemctl restart strongswan-starter"
fi

# 检查连接配置是否加载
if ipsec statusall 2>/dev/null | grep -q "ikev2-eap"; then
  ok "ikev2-eap 连接配置已加载"
  ipsec statusall 2>/dev/null | grep -E "left=|leftid=|Listening" | head -5 | sed 's/^/    /'
else
  fail "ikev2-eap 连接配置未加载"
  echo "    → 检查 /etc/ipsec.conf"
fi

# ── 2. 证书一致性检查 ─────────────────────────────────────────────────────────
echo ""
echo "[ 证书一致性 ]"

IPSEC_CA="/etc/ipsec.d/cacerts/caCert.pem"
BACKEND_CA="$DEPLOY_DIR/certs/caCert.pem"

if [ ! -f "$IPSEC_CA" ]; then
  fail "StrongSwan CA 不存在: $IPSEC_CA"
else
  IPSEC_CA_FP=$(openssl x509 -in "$IPSEC_CA" -noout -fingerprint -sha256 2>/dev/null | cut -d= -f2)
  ok "StrongSwan CA: $IPSEC_CA_FP"
fi

if [ ! -f "$BACKEND_CA" ]; then
  fail "后端 CA 不存在: $BACKEND_CA"
else
  BACKEND_CA_FP=$(openssl x509 -in "$BACKEND_CA" -noout -fingerprint -sha256 2>/dev/null | cut -d= -f2)
  ok "后端 CA:       $BACKEND_CA_FP"
fi

if [ -n "$IPSEC_CA_FP" ] && [ -n "$BACKEND_CA_FP" ]; then
  if [ "$IPSEC_CA_FP" = "$BACKEND_CA_FP" ]; then
    ok "两套 CA 一致 ✓"
  else
    fail "CA 不一致！iOS 会验证失败"
    echo "    → 执行 bash server-setup.sh 重新生成"
  fi
fi

# 检查服务器证书 CN 和 SAN
echo ""
CERT="/etc/ipsec.d/certs/serverCert.pem"
if [ -f "$CERT" ]; then
  CN=$(openssl x509 -in "$CERT" -noout -subject 2>/dev/null | grep -o 'CN=[^,/]*' | cut -d= -f2)
  SAN=$(openssl x509 -in "$CERT" -noout -extensions san -text 2>/dev/null | grep -A1 "Subject Alternative" | tail -1 | xargs)
  ok "服务器证书 CN: $CN"
  ok "服务器证书 SAN: $SAN"
  # 检查 CA 签名
  if openssl verify -CAfile "$IPSEC_CA" "$CERT" 2>/dev/null | grep -q "OK"; then
    ok "服务器证书由当前 CA 签名 ✓"
  else
    fail "服务器证书不是由当前 CA 签名"
    echo "    → 需要重新生成证书"
  fi
fi

# ── 3. 后端服务 ───────────────────────────────────────────────────────────────
echo ""
echo "[ 后端服务 ]"
if systemctl is-active --quiet vpn-location; then
  ok "后端服务运行中"
  # 检查 config.yaml 里的 CA 路径
  CFG="$DEPLOY_DIR/config.yaml"
  if [ -f "$CFG" ]; then
    CA_PATH=$(grep "root_ca_path" "$CFG" | awk '{print $2}')
    KEY_PATH=$(grep "root_key_path" "$CFG" | awk '{print $2}')
    SVR_ADDR=$(grep "server_address" "$CFG" | awk '{print $2}')
    ok "config.yaml server_address: $SVR_ADDR"
    ok "config.yaml root_ca_path: $CA_PATH"
    [ -f "$CA_PATH" ] && ok "CA 文件存在" || fail "CA 文件不存在: $CA_PATH"
    [ -f "$KEY_PATH" ] && ok "CA 私钥存在" || fail "CA 私钥不存在: $KEY_PATH"
  fi
else
  fail "后端服务未运行"
  echo "    最近日志:"
  journalctl -u vpn-location -n 5 --no-pager 2>/dev/null | sed 's/^/    /'
fi

# ── 4. 网络和防火墙 ───────────────────────────────────────────────────────────
echo ""
echo "[ 网络 ]"
OUTIF=$(ip route show default | awk '{print $5}' | head -1)
INTERNAL_IP=$(ip addr show "$OUTIF" 2>/dev/null | grep 'inet ' | awk '{print $2}' | cut -d/ -f1)
PUBLIC_IP=$(curl -s --max-time 5 ifconfig.me 2>/dev/null)

ok "内网 IP: $INTERNAL_IP (网卡 $OUTIF)"
ok "公网 IP: $PUBLIC_IP"

if [ "$INTERNAL_IP" != "$PUBLIC_IP" ]; then
  warn "NAT 环境检测到：内网 IP ≠ 公网 IP"
  # 检查 ipsec.conf 的 left/leftid
  LEFT=$(grep "^\s*left=" /etc/ipsec.conf 2>/dev/null | head -1 | tr -d ' ' | cut -d= -f2)
  LEFTID=$(grep "^\s*leftid=" /etc/ipsec.conf 2>/dev/null | head -1 | tr -d ' ' | cut -d= -f2)
  FORCE=$(grep "forceencaps" /etc/ipsec.conf 2>/dev/null | head -1)
  [ "$LEFT" = "$INTERNAL_IP" ] && ok "ipsec.conf left=$LEFT ✓" || fail "ipsec.conf left=$LEFT (应为 $INTERNAL_IP)"
  [ "$LEFTID" = "$PUBLIC_IP" ] && ok "ipsec.conf leftid=$LEFTID ✓" || fail "ipsec.conf leftid=$LEFTID (应为 $PUBLIC_IP)"
  echo "$FORCE" | grep -q "yes" && ok "forceencaps=yes ✓" || fail "forceencaps 未启用（NAT 环境必须）"
fi

# 防火墙
iptables -C INPUT -p udp --dport 500  -j ACCEPT 2>/dev/null && ok "防火墙: UDP 500 放行" || fail "防火墙: UDP 500 未放行"
iptables -C INPUT -p udp --dport 4500 -j ACCEPT 2>/dev/null && ok "防火墙: UDP 4500 放行" || fail "防火墙: UDP 4500 未放行"
iptables -C INPUT -p tcp --dport 8080 -j ACCEPT 2>/dev/null && ok "防火墙: TCP 8080 放行" || fail "防火墙: TCP 8080 未放行"

# NAT
iptables -t nat -C POSTROUTING -s 10.10.0.0/24 -o "$OUTIF" -j MASQUERADE 2>/dev/null \
  && ok "NAT MASQUERADE 规则存在" || fail "NAT MASQUERADE 规则缺失"

echo ""
echo "========================================"
echo " 诊断完成"
echo "========================================"
