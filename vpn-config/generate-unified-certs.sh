#!/bin/bash

# 统一VPN证书生成脚本
# 用法: SERVER_IP=你的服务器IP bash generate-unified-certs.sh
# 此脚本生成统一证书，确保StrongSwan服务端和mobileconfig使用相同的CA证书

set -e

SERVER_IP="${SERVER_IP:-192.168.10.7}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR"
BACKEND_CERT_DIR="$SCRIPT_DIR/../backend"
CONFIG_FILE="$BACKEND_CERT_DIR/config.yaml"

echo "=========================================="
echo "统一VPN证书生成脚本"
echo "=========================================="
echo "使用服务器IP: $SERVER_IP"
echo "证书目录: $CERT_DIR"
echo ""

# 清理旧证书
echo "清理旧证书..."
rm -f "$CERT_DIR/caKey.pem" "$CERT_DIR/caCert.pem" "$CERT_DIR/serverKey.pem" "$CERT_DIR/serverCert.pem" "$CERT_DIR/serverReq.pem" "$CERT_DIR/caCert.srl"

# 生成CA私钥和证书
echo "生成CA私钥和证书..."
openssl genrsa -out "$CERT_DIR/caKey.pem" 2048
openssl req -new -x509 -days 3650 -key "$CERT_DIR/caKey.pem" -out "$CERT_DIR/caCert.pem" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=VPN Location Spoofer/OU=CA/CN=VPN Root CA"

# 生成服务器私钥
echo "生成服务器私钥..."
openssl genrsa -out "$CERT_DIR/serverKey.pem" 2048

# 生成服务器证书签名请求
echo "生成服务器证书签名请求..."
openssl req -new -key "$CERT_DIR/serverKey.pem" -out "$CERT_DIR/serverReq.pem" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=VPN Location Spoofer/OU=Server/CN=$SERVER_IP"

# 添加 SAN（iOS 16+ 要求证书包含 IP SAN）
echo "配置证书扩展..."
if [[ "$SERVER_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  cat > /tmp/server_ext.cnf << EOF
subjectAltName = IP:$SERVER_IP
extendedKeyUsage = serverAuth
EOF
else
  cat > /tmp/server_ext.cnf << EOF
subjectAltName = DNS:$SERVER_IP
extendedKeyUsage = serverAuth
EOF
fi

# 用CA签名服务器证书，附加 SAN
echo "用CA签名服务器证书..."
openssl x509 -req -in "$CERT_DIR/serverReq.pem" -CA "$CERT_DIR/caCert.pem" -CAkey "$CERT_DIR/caKey.pem" \
    -CAcreateserial -out "$CERT_DIR/serverCert.pem" -days 3650 \
    -extfile /tmp/server_ext.cnf

# 设置权限
chmod 600 "$CERT_DIR/caKey.pem" "$CERT_DIR/serverKey.pem"
chmod 644 "$CERT_DIR/caCert.pem" "$CERT_DIR/serverCert.pem" "$CERT_DIR/serverReq.pem"

# 验证证书指纹
echo ""
echo "验证证书指纹..."
CA_FINGERPRINT=$(openssl x509 -in "$CERT_DIR/caCert.pem" -noout -fingerprint -sha256 | cut -d= -f2)
echo "CA证书指纹: $CA_FINGERPRINT"

SERVER_FINGERPRINT=$(openssl x509 -in "$CERT_DIR/serverCert.pem" -noout -fingerprint -sha256 | cut -d= -f2)
echo "服务器证书指纹: $SERVER_FINGERPRINT"

# 更新后端配置文件
echo ""
echo "更新后端配置文件..."
if [ -f "$CONFIG_FILE" ]; then
    # 使用sed更新证书路径
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s|root_ca_path:.*|root_ca_path: $CERT_DIR/caCert.pem|g" "$CONFIG_FILE"
        sed -i '' "s|root_key_path:.*|root_key_path: $CERT_DIR/caKey.pem|g" "$CONFIG_FILE"
    else
        # Linux
        sed -i "s|root_ca_path:.*|root_ca_path: $CERT_DIR/caCert.pem|g" "$CONFIG_FILE"
        sed -i "s|root_key_path:.*|root_key_path: $CERT_DIR/caKey.pem|g" "$CONFIG_FILE"
    fi
    echo "已更新: $CONFIG_FILE"
else
    echo "警告: 配置文件不存在: $CONFIG_FILE"
fi

# 更新ipsec.conf中的服务器地址
echo ""
echo "更新ipsec.conf配置..."
if [ -f "$CERT_DIR/ipsec.conf" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/leftid=.*/leftid=$SERVER_IP/g" "$CERT_DIR/ipsec.conf"
    else
        sed -i "s/leftid=.*/leftid=$SERVER_IP/g" "$CERT_DIR/ipsec.conf"
    fi
    echo "已更新: $CERT_DIR/ipsec.conf (leftid=$SERVER_IP)"
else
    echo "警告: ipsec.conf不存在: $CERT_DIR/ipsec.conf"
fi

echo ""
echo "=========================================="
echo "证书生成完成"
echo "=========================================="
echo "CA证书:     $CERT_DIR/caCert.pem"
echo "CA私钥:     $CERT_DIR/caKey.pem"
echo "服务器证书:  $CERT_DIR/serverCert.pem"
echo "服务器私钥:  $CERT_DIR/serverKey.pem"
echo ""
echo "证书指纹: $CA_FINGERPRINT"
echo ""
echo "部署步骤:"
echo "1. StrongSwan服务端部署:"
echo "   cp $CERT_DIR/serverCert.pem /etc/ipsec.d/certs/"
echo "   cp $CERT_DIR/serverKey.pem  /etc/ipsec.d/private/"
echo "   cp $CERT_DIR/caCert.pem     /etc/ipsec.d/cacerts/"
echo "   cp $CERT_DIR/ipsec.conf     /etc/ipsec.conf"
echo "   cp $CERT_DIR/ipsec.secrets  /etc/ipsec.secrets"
echo "   ipsec restart"
echo ""
echo "2. 后端配置已自动更新，重启后端服务"
echo "3. mobileconfig将自动使用统一的CA证书"
echo ""
echo "验证命令:"
echo "   openssl x509 -in $CERT_DIR/caCert.pem -noout -fingerprint -sha256"
echo "   openssl x509 -in /etc/ipsec.d/cacerts/caCert.pem -noout -fingerprint -sha256"
echo "=========================================="
