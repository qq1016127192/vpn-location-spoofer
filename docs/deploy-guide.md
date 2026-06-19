# VPN Location Spoofer 部署指南

## 项目概述

基于 Go + Vue 3 的 iOS VPN 定位篡改系统，支持多用户隔离、iOS 17~27 完整支持。

---

## 系统要求

- **操作系统**: Ubuntu 20.04+ / Debian 10+
- **Go 版本**: 1.19+
- **Node.js 版本**: 16+
- **内存**: 最低 2GB，推荐 4GB+
- **网络**: 公网 IP（需要开放 80、443、8080、8081 端口）

---

## 快速部署

### 方法1：使用部署脚本（推荐）

```bash
# 克隆项目
git clone https://github.com/qq1016127192/vpn-location-spoofer.git
cd vpn-location-spoofer

# 运行部署脚本
chmod +x deploy.sh
sudo ./deploy.sh
```

### 方法2：手动部署

#### 1. 安装依赖

```bash
# 安装 Go（如果未安装）
sudo apt update
sudo apt install -y golang-go

# 安装 Node.js
curl -fsSL https://deb.nodesource.com/setup_16.x | sudo -E bash -

# 安装 StrongSwan（IPsec VPN）
sudo apt install -y strongswan strongswan-pki libcharon-extra-modules

# 安装其他依赖
sudo apt install -y iptables-persistent
```

#### 2. 配置 StrongSwan VPN

```bash
# 复制 VPN 配置文件
sudo cp vpn-config/ipsec.conf /etc/ipsec.conf
sudo cp vpn-config/ipsec.secrets /etc/ipsec.secrets
sudo cp vpn-config/swanctl.conf /etc/swanctl.conf

# 设置权限
sudo chmod 600 /etc/ipsec.secrets

# 启动 StrongSwan
sudo systemctl restart strongswan
sudo systemctl enable strongswan
```

#### 3. 配置防火墙和 NAT

```bash
# 复制 NAT 配置
sudo cp vpn-nat.conf /etc/iptables/rules.v4

# 应用 NAT 规则
sudo iptables-restore < /etc/iptables/rules.v4
sudo iptables-persistent save

# 开启 IP 转发
echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward
```

#### 4. 启动后端服务

```bash
# 编译后端
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-extldflags=-static -s -w" -o vpn-server-linux .

# 创建数据目录
sudo mkdir -p /opt/vpn-data

# 复制可执行文件
sudo cp vpn-server-linux /opt/vpn-location/vpn-server
sudo chmod +x /opt/vpn-location/vpn-server

# 创建 systemd 服务
sudo tee /etc/systemd/system/vpn-location.service << 'EOF'
[Unit]
Description=VPN Location Server
After=network.target strongswan.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vpn-location
ExecStart=/opt/vpn-location/vpn-server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl start vpn-location
sudo systemctl enable vpn-location
```

#### 5. 配置 Nginx（可选）

```bash
# 安装 Nginx
sudo apt install -y nginx

# 配置 Nginx
sudo tee /etc/nginx/sites-available/vpn-location << 'EOF'
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
EOF

# 启用配置
sudo ln -s /etc/nginx/sites-available/vpn-location /etc/nginx/sites-enabled/
sudo systemctl reload nginx
```

---

## 配置说明

### 服务器配置

编辑 `backend/config.yaml`:

```yaml
vpn:
  server_address: "your-domain.com"  # 修改为你的域名或公网IP

cert:
  root_ca_path: "/opt/vpn-location/certs/rootCA.pem"
  root_key_path: "/opt/vpn-location/certs/rootCA.key"
  cert_path: "/opt/vpn-location/certs/serverCert.pem"
  key_path: "/opt/vpn-location/certs/serverKey.pem"

server:
  port: 8080
  host: "0.0.0.0"
```

### VPN 配置

编辑 `vpn-config/ipsec.conf`:

```conf
config setup
conn %default
    keyexchange=ikev2
    ike=aes256-sha1-modp2048!
    esp=aes256-sha1!
    left=%defaultroute
    leftid=@your-domain.com
    leftcert=/etc/ipsec.d/serverCert.pem
    leftsendcert=always
    leftsubnet=0.0.0.0/0
    leftfirewall=yes
    right=%any
    rightauth=eap-mschapv2
    rightca=%cert
    eap_identity=%identity
    auto=add
    dpdaction=clear
    dpddelay=40
    type=transport
```

---

## 功能特性

### 1. 多用户隔离

- 每个用户使用独立的 VPN 凭据（自动生成）
- 每个用户的定位状态完全隔离
- 每个用户的拦截统计独立

### 2. iOS 版本支持

- iOS 5~16: 支持 `gs-loc.apple.com`、`location.apple.com`
- iOS 17~27: 额外支持 `*.ls.apple.com`、`gsp-ssl.ls.apple.com`、`ls-cn.apple.com`

### 3. 数据持久化

- MITM 代理数据自动保存到 `/opt/vpn-data/mitm-data.json`
- 30秒自动保存一次
- 服务重启后自动恢复数据

### 4. 坐标系转换

- 前端支持 WGS84、GCJ02、BD09 坐标系
- 自动转换高德地图瓦片（GCJ02）到 WGS