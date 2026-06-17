#!/bin/bash

# 生成VPN服务器证书，并自动同步 CA 到 vpn.mobileconfig
# 用法: SERVER_IP=你的服务器IP bash generate-certs.sh
# 默认使用 192.168.10.7，修改 SERVER_IP 变量以匹配你的实际 IP

SERVER_IP="${SERVER_IP:-192.168.10.7}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MOBILECONFIG="$SCRIPT_DIR/../vpn.mobileconfig"
cd "$SCRIPT_DIR"

echo "使用服务器IP: $SERVER_IP"

# 生成CA私钥和证书
openssl genrsa -out caKey.pem 2048
openssl req -new -x509 -days 3650 -key caKey.pem -out caCert.pem \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=VPN Location Spoofer/OU=CA/CN=VPN Root CA"

# 生成服务器私钥
openssl genrsa -out serverKey.pem 2048

# 生成服务器证书签名请求
# CN 必须和 ipsec.conf leftid 以及 mobileconfig RemoteIdentifier 完全一致
openssl req -new -key serverKey.pem -out serverReq.pem \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=VPN Location Spoofer/OU=Server/CN=$SERVER_IP"

# 添加 SAN（iOS 16+ 要求证书包含 IP SAN）
# 判断是 IP 还是域名，SAN 格式不同
if echo "$SERVER_IP" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'; then
  SAN="IP:$SERVER_IP"
else
  SAN="DNS:$SERVER_IP"
fi

cat > /tmp/server_ext.cnf << EOF
subjectAltName = $SAN
extendedKeyUsage = serverAuth
EOF

# 用CA签名服务器证书，附加 SAN
openssl x509 -req -in serverReq.pem -CA caCert.pem -CAkey caKey.pem \
    -CAcreateserial -out serverCert.pem -days 3650 \
    -extfile /tmp/server_ext.cnf

# 设置权限
chmod 600 caKey.pem serverKey.pem
chmod 644 caCert.pem serverCert.pem serverReq.pem

# ── 自动同步 CA 到 vpn.mobileconfig ──────────────────────────────────────────
if [ ! -f "$MOBILECONFIG" ]; then
  echo "警告: 未找到 $MOBILECONFIG，跳过自动同步"
else
  # 提取 DER 格式的 base64（去掉 PEM 头尾和换行）
  NEW_CA_B64=$(grep -v 'BEGIN\|END' caCert.pem | tr -d '\n')

  # 用 Python 做精准替换（避免 sed 跨行问题）
  python3 - "$MOBILECONFIG" "$NEW_CA_B64" << 'PYEOF'
import sys, re

path    = sys.argv[1]
new_b64 = sys.argv[2]

with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

# mobileconfig 结构：<key>PayloadContent</key><data>...</data> 紧跟在证书 payload 块里
# 用包含 com.apple.security.root 的 dict 块来定位，再替换其中的 PayloadContent <data>
# 先找到包含 com.apple.security.root 的整个 <dict>...</dict> 块，再替换其中的 PayloadContent
pattern = r'(<dict>(?:(?!<dict>|</dict>).|\n)*?<string>com\.apple\.security\.root</string>(?:(?!<dict>|</dict>).|\n)*?<key>PayloadContent</key>\s*<data>)[^<]*(</data>)'
replacement = r'\g<1>' + new_b64 + r'\g<2>'

new_content, count = re.subn(pattern, replacement, content, flags=re.DOTALL)

if count == 0:
    # 备用方案：直接找 PayloadCertificateFileName 后的第一个 <data> 块
    pattern2 = r'(PayloadCertificateFileName.*?<key>PayloadContent</key>\s*<data>)[^<]*(</data>)'
    new_content, count = re.subn(pattern2, replacement, content, flags=re.DOTALL)

if count == 0:
    print("警告: mobileconfig 中未找到 CA PayloadContent，请手动更新")
    sys.exit(1)

with open(path, 'w', encoding='utf-8') as f:
    f.write(new_content)

print(f"已自动将新 CA 写入 {path}")
PYEOF

  if [ $? -eq 0 ]; then
    # 验证：打印新旧 MD5 供确认
    NEW_MD5=$(openssl x509 -in caCert.pem -outform DER | md5)
    echo "新 CA MD5: $NEW_MD5"
    echo "请重新分发 vpn.mobileconfig 给 iOS 设备并重新安装描述文件"
  fi
fi
# ─────────────────────────────────────────────────────────────────────────────

echo ""
echo "证书生成完成"

# ── 自动同步证书到 StrongSwan 并重载 ─────────────────────────────────────────
IPSEC_BIN=$(which ipsec 2>/dev/null || echo "")
IPSEC_BASE=$(dirname "$(dirname "$IPSEC_BIN")" 2>/dev/null)
IPSEC_CERTS="$IPSEC_BASE/etc/ipsec.d/certs"
IPSEC_PRIVATE="$IPSEC_BASE/etc/ipsec.d/private"
IPSEC_CACERTS="$IPSEC_BASE/etc/ipsec.d/cacerts"

if [ -z "$IPSEC_BIN" ] || [ ! -f "$IPSEC_BIN" ]; then
  echo "警告: 未找到 $IPSEC_BIN，跳过 StrongSwan 自动同步"
  echo "请手动执行:"
  echo "  cp serverCert.pem /opt/homebrew/etc/ipsec.d/certs/"
  echo "  cp serverKey.pem  /opt/homebrew/etc/ipsec.d/private/"
  echo "  cp caCert.pem     /opt/homebrew/etc/ipsec.d/cacerts/"
  echo "  /opt/homebrew/sbin/ipsec restart"
else
  echo "正在同步证书到 StrongSwan..."
  SWANCTL_X509="$IPSEC_BASE/etc/swanctl/x509"
  SWANCTL_X509CA="$IPSEC_BASE/etc/swanctl/x509ca"
  SWANCTL_PRIVATE="$IPSEC_BASE/etc/swanctl/private"
  sudo mkdir -p "$IPSEC_CERTS" "$IPSEC_PRIVATE" "$IPSEC_CACERTS" \
                "$SWANCTL_X509" "$SWANCTL_X509CA" "$SWANCTL_PRIVATE"

  # 清理旧 CA 和误放的文件，避免多套共存导致验证混乱
  sudo rm -f "$IPSEC_CACERTS"/*.pem "$IPSEC_CACERTS"/*.crt
  sudo rm -f "$IPSEC_PRIVATE"/caKey.pem
  sudo rm -f "$SWANCTL_X509CA"/*.pem "$SWANCTL_X509CA"/*.crt

  # 写入新证书（ipsec.d 和 swanctl 两处同步）
  sudo cp caCert.pem     "$IPSEC_CACERTS/caCert.pem"
  sudo cp serverCert.pem "$IPSEC_CERTS/serverCert.pem"
  sudo cp serverKey.pem  "$IPSEC_PRIVATE/serverKey.pem"
  sudo cp caCert.pem     "$SWANCTL_X509CA/caCert.pem"
  sudo cp serverCert.pem "$SWANCTL_X509/serverCert.pem"
  sudo cp serverKey.pem  "$SWANCTL_PRIVATE/serverKey.pem"
  sudo chmod 600 "$IPSEC_PRIVATE/serverKey.pem" "$SWANCTL_PRIVATE/serverKey.pem"
  sudo chmod 644 "$IPSEC_CERTS/serverCert.pem" "$IPSEC_CACERTS/caCert.pem" \
                 "$SWANCTL_X509/serverCert.pem" "$SWANCTL_X509CA/caCert.pem"
  echo "证书已复制到 StrongSwan 目录（ipsec.d + swanctl）"

  # 部署配置文件，并将 SERVER_IP 占位符替换为实际 IP
  # ipsec.conf: left= 和 leftid= 替换
  sed "s/left=192\.168\.10\.7/left=$SERVER_IP/g;
       s/leftid=192\.168\.10\.7/leftid=$SERVER_IP/g" \
    "$SCRIPT_DIR/ipsec.conf" | sudo tee "$IPSEC_BASE/etc/ipsec.conf" > /dev/null

  # ipsec.secrets: 直接复制（不含 IP）
  sudo cp "$SCRIPT_DIR/ipsec.secrets" "$IPSEC_BASE/etc/ipsec.secrets"

  # swanctl.conf: id = 替换
  sed "s/id = 192\.168\.10\.7/id = $SERVER_IP/g" \
    "$SCRIPT_DIR/swanctl.conf" | sudo tee "$IPSEC_BASE/etc/swanctl/swanctl.conf" > /dev/null

  echo "ipsec.conf + ipsec.secrets + swanctl.conf 已部署（SERVER_IP=$SERVER_IP）"

  # sudo 检查运行状态（不用 sudo 会 permission denied 误判）
  if sudo "$IPSEC_BIN" status > /dev/null 2>&1; then
    echo "StrongSwan 正在运行，重启以完全清除旧证书缓存..."
    sudo "$IPSEC_BIN" restart
    sleep 3
    echo "StrongSwan 已重启，当前加载的 CA:"
    sudo "$IPSEC_BIN" listcacerts 2>&1 | grep -E "subject:|serial:" | head -4
    echo "当前连接配置:"
    sudo "$IPSEC_BIN" statusall 2>&1 | grep -A3 "^Connections:"
  else
    echo "StrongSwan 未运行，跳过重载"
  fi
fi
# ─────────────────────────────────────────────────────────────────────────────

echo ""
echo "全部完成。下一步："
echo "  1. 将更新后的 vpn.mobileconfig 重新安装到 iOS 设备"
echo "  2. 在 iOS 设置中信任新 CA 证书"
echo "  3. 重新连接 VPN"

# ── 确保 macOS pf 重定向规则存在（绕过 racoon 抢占 500 端口）────────────────
if [ "$(uname)" = "Darwin" ]; then
  PF_ANCHOR="/etc/pf.anchors/vpn-location"
  # 写入最新规则（NAT + RDR，包含当前 SERVER_IP）
  sudo tee "$PF_ANCHOR" > /dev/null << EOF
# NAT：VPN 客户端出网
nat on en0 from 10.10.0.0/24 to any -> (en0)

# RDR：绕过 macOS racoon 占用 500/4500，转给 charon 5500/5501
rdr pass on en0 proto udp from any to $SERVER_IP port 500  -> $SERVER_IP port 5500
rdr pass on en0 proto udp from any to $SERVER_IP port 4500 -> $SERVER_IP port 5501
EOF

  # 确保 pf.conf 里有 rdr-anchor "vpn-location"
  if ! sudo grep -q 'rdr-anchor "vpn-location"' /etc/pf.conf; then
    sudo sed -i '' 's|rdr-anchor "com.apple/\*"|rdr-anchor "com.apple/*"\
rdr-anchor "vpn-location"|' /etc/pf.conf
    echo "pf.conf 已添加 rdr-anchor vpn-location"
  fi

  # 重载 pf 规则
  sudo pfctl -f /etc/pf.conf 2>/dev/null
  echo "pf 重定向规则已更新（500→5500, 4500→5501）"
fi
