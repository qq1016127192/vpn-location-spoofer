#!/bin/bash
# 本地打包脚本
# 用法: bash package.sh
set -e

OUTPUT="vpn-package.tar.gz"
SERVER_IP="${SERVER_IP:-101.37.76.102}"

echo "打包项目源码..."

# 编译 Linux amd64 二进制
echo "编译 Linux amd64 二进制..."
(cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../vpn-server .)
echo "编译完成"

tar -czf $OUTPUT \
  --no-xattrs \
  --exclude=backend/vendor \
  --exclude=backend/vpn-server \
  --exclude=backend/vpn-server-linux \
  --exclude=backend/vpn_server \
  --exclude=backend/backend \
  --exclude=frontend/node_modules \
  --exclude=frontend/dist \
  --exclude=.DS_Store \
  --exclude='*.tar.gz' \
  --exclude=vpn-config/caKey.pem \
  --exclude=vpn-config/caCert.pem \
  --exclude=vpn-config/serverCert.pem \
  --exclude=vpn-config/serverKey.pem \
  --exclude=vpn-config/serverReq.pem \
  --exclude=vpn-config/caCert.srl \
  backend \
  frontend/src \
  frontend/index.html \
  frontend/package.json \
  frontend/vite.config.js \
  frontend/postcss.config.js \
  frontend/tailwind.config.js \
  vpn-config/ipsec.conf \
  vpn-config/ipsec.secrets \
  vpn-config/swanctl.conf \
  server-setup.sh \
  check-deploy.sh \
  vpn-server

rm -f vpn-server

echo "打包完成: $OUTPUT ($(du -sh $OUTPUT | cut -f1))"
