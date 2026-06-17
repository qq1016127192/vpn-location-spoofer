#!/bin/bash
# 本地打包脚本：只打包源码，不修改任何本地文件
# 用法: bash package.sh
# 输出: vpn-package.tar.gz
#
# 上传到服务器后执行:
#   tar -xzf vpn-package.tar.gz
#   SERVER_IP=103.231.59.141 bash server-setup.sh

set -e

OUTPUT="vpn-package.tar.gz"

echo "打包项目源码..."

tar -czf $OUTPUT \
  --no-xattrs \
  --exclude='backend/vendor' \
  --exclude='backend/*.exe' \
  --exclude='backend/vpn-server' \
  --exclude='backend/vpn-server-linux' \
  --exclude='backend/vpn_server' \
  --exclude='backend/backend' \
  --exclude='backend/*.txt' \
  --exclude='frontend/node_modules' \
  --exclude='frontend/dist' \
  --exclude='frontend/*.zip' \
  --exclude='.DS_Store' \
  --exclude='*.tar.gz' \
  --exclude='vpn-config/caKey.pem' \
  --exclude='vpn-config/caCert.pem' \
  --exclude='vpn-config/serverCert.pem' \
  --exclude='vpn-config/serverKey.pem' \
  --exclude='vpn-config/serverReq.pem' \
  --exclude='vpn-config/caCert.srl' \
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
  server-setup.sh

echo ""
echo "打包完成: $OUTPUT ($(du -sh $OUTPUT | cut -f1))"
echo ""
echo "上传到服务器:"
echo "  scp $OUTPUT root@103.231.59.141:/root/"
echo ""
echo "服务器上执行:"
echo "  tar -xzf vpn-package.tar.gz"
echo "  SERVER_IP=103.231.59.141 bash server-setup.sh"
