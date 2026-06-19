# VPN 定位篡改系统 - 完整部署指南

## 📋 项目概述

本项目是一个基于 IKEv2/EAP VPN + MITM 代理的 iOS 定位篡改系统，支持多用户隔离、数据持久化、iOS 17~27 全版本支持。

---

## 🏗️ 系统要求

### 服务器要求
- **操作系统**: Ubuntu 20.04+ / CentOS 7+
- **内存**: 至少 2GB RAM
- **CPU**: 1核及以上
- **公网IP**: 需要固定公网IP（用于 VPN 服务）
- **端口**: 80 (HTTP), 443 (HTTPS), 500 (IKEv2), 8080 (Web API), 8081 (MITM 代理)

### 客户端要求
- **iOS 版本**: iOS 17 ~ iOS 27
- **VPN 类型**: IKEv2/EAP
- **网络**: 支持 Wi-Fi 和蜂窝网络

---

## 📦 部署架构

```
┌─────────────────────────────────────────────────────────┐
│  iOS 设备                                      │
│  (IKEv2/EAP VPN)                                │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│  VPN 服务器 (Ubuntu)                              │
│  - StrongSwan (500/UDP)                         │
│  - iptables NAT                                   │
│  - Nginx (80/443)                              │
│  - vpn-server (Go)                                │
│    - Web API (8080)                              │
│    - MITM 代理 (8081)                            │
│    - 数据持久化 (/opt/vpn-data/)                   │
└─────────────────────────────────────────────────────────┘
```

---

## 🚀 快速部署

### 方式一：使用部署脚本（推荐）

```bash
# 克隆项目
git clone https://github.com/qq1016127192/vpn-location-spoofer.git
cd vpn-location-spoofer

# 运行部署脚本
chmod +x deploy.sh
sudo ./deploy.sh
```

### 方式二：手动部署

#### 1. 安装依赖

```bash
# 安装 StrongSwan
sudo apt update
sudo apt install -y strongswan-starter libcharon-extra-utils

# 安装 Nginx
sudo apt install -y nginx

# 安装 Go (如果未安装)
wget https://go.dev/dl/go1.22.7.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.7.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 2. 配置 StrongSwan

```bash
# 复制 VPN 配置
sudo cp vpn-config/ipsec.conf /etc/ipsec.conf
sudo cp vpn-config/ipsec.secrets /etc/ipsec.secrets
sudo cp vpn-config/swanctl.conf /etc/swanctl.conf

# 启动 StrongSwan
sudo systemctl restart strongswan-starter
sudo systemctl enable strongswan-starter
```

#### 3. 配置 Nginx

```bash
# 复制 NCA 配置
sudo cp nginx/vpn-location.conf /etc/nginx/sites-available/vpn-location

# 启用配置
sudo ln -s /etc/nginx/sites-available/vpn-location /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

#### 4. 配置 SSL 证书（可选）

```bash
# 使用 Let's Encrypt
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d map.iosiosr.cn --email your-email@example.com

# 或使用自签名证书
sudo cp vpn-config/*.pem /etc/nginx/ssl/
```

#### 5. 编译并启动后端

```bash
# 编译后端
cd backend
go build -o vpn-server

# 创建数据目录
sudo mkdir -p /opt/vpn-data

# 启动服务
sudo cp vpn-server /opt/vpn-location/vpn-server
sudo chmod +x /opt/vpn-location/vpn-server

# 创建 systemd 服务
sudo tee /etc/systemd/system/vpn-location.service > /dev/null <<EOF
[Unit]
Description=VPN Location Server
After=network.target

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
sudo systemctl enable vpn-location
sudo systemctl start vpn-location
```

#### 6. 配置前端

```bash
# 安装 Node.js 依赖
cd frontend
npm install

# 编译前端
npm run build

# 复制到 Nginx 目录
sudo rm -rf /opt/vpn-location/frontend/dist
sudo cp -r dist /opt/vpn-location/frontend/dist
```

---

## 🔧 配置说明

### 服务器配置文件

#### config.yaml (后端配置)

```yaml
vpn:
  server_address: "map.iosiosr.cn"  # VPN 服务器地址
  subnet: "10.10.0.0/16"              # VPN 分配的网段

cert:
  root_ca_path: "vpn-config/caCert.pem"
  root_key_path: "vpn-config/caKey.pem"
  server_cert_path: "vpn-config/serverCert.pem"

data:
  users_file: "/opt/vpn-data/users.json"
```

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|----------|
| `DATA_DIR` | 数据目录 | `/opt/vpn-data` |
| `MITM_DATA_FILE` | MITM 代理数据文件 | `{DATA_DIR}/mitm-data.json` |
| `LOCATION_DATA_FILE` | 位置服务数据文件 | `{DATA_DIR}/location-data.json` |
| `FRONTEND_DIST` | 前端静态文件目录 | `./frontend/dist` |

---

## 🔐 安全配置

### 1. VPN 凭据管理

```bash
# 生成 VPN 用户凭据
# 每个用户有独立的 VPN 用户名和密码

# 格式：
vpn_user: vpn_{8位随机字符}
vpn_pass: {12位随机字符}
```

### 2. API 认证

```bash
# 登录 API
POST /api/login
{
  "username": "your_username",
  "password": "your_password"
}

# 返回 JWT Token（可选）
{
  "token": "jwt_token_here",
  "vpn_user": "vpn_abc12345",
  "vpn_pass": "random123456789"
}
```

### 3. CORS 配置

```nginx
# Nginx 配置
add_header 'Access-Control-Allow-Origin' '*' always;
add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS, PUT, DELETE' always;
add_header 'Access-Control-Allow-Headers' 'Content-Type, Authorization, X-Real-IP, X-Forwarded-For' always;
```

---

## 📊 监控和日志

### 查看服务状态

```bash
# VPN 服务
sudo systemctl status strongswan-starter
sudo ipsec statusall

# Web 服务
sudo systemctl status vpn-location
sudo journalctl -u vpn-location -f

# Nginx 服务
sudo systemctl status nginx
sudo tail -f /var/log/nginx/access.log
```

### 查看数据

```bash
# MITM 代理数据
cat /opt/vpn-data/mitm-data.json

# 位置服务数据
cat /opt/vpn-data/location-data.json

# 用户数据
cat /opt/vpn-data/users.json
```

---

## 🔄 更新部署

### 增量更新（保留数据）

```bash
# 1. 编译新版本
cd backend
go build -o vpn-server-linux

cd ../frontend
npm run build

# 2. 停止服务
sudo systemctl stop vpn-location

# 3. 备份当前数据
sudo cp -r /opt/vpn-data /opt/vpn-data.backup.$(date +%Y%m%d%H%M%S)

# 4. 更新二进制文件
sudo cp backend/vpn-server-linux /opt/vpn-location/vpn-server
sudo chmod +x /opt/vpn-location/vpn-server

# 5. 更新前端
sudo rm -rf /opt/vpn-location/frontend/dist
sudo cp -r frontend/dist /opt/vpn-location/frontend/dist

# 6. 重启服务
sudo systemctl start vpn-location

# 7. 检查状态
sudo systemctl status vpn-location
```

### 回滚更新

```bash
# 如果更新失败，恢复数据
sudo systemctl stop vpn-location
sudo rm -rf /opt/vpn-data
sudo cp -r /opt/vpn-data.backup.* /opt/vpn-data
sudo systemctl start vpn-location
```

---

## 🐛 故障排查

### 问题1: VPN 连接失败

```bash
# 检查 StrongSwan 状态
sudo ipsec statusall

# 检查证书
sudo ipsec listcerts

# 检查日志
sudo journalctl -u strongswan -f -n 100
```

### 问题2: MITM 代理不工作

```bash
# 检查代理是否运行
curl -x http://127.0.0.1:8081/health

# 检查 CA 证书
ls -la vpn-config/*.pem

# 检查 PAC 文件
curl http://map.iosiosr.cn:8080/proxy.pac
```

### 问题3: 定位不生效

```bash
# 检查 MITM 拦截日志
sudo journalctl -u vpn-location -f | grep MITM

# 检查客户端 IP 隔离
curl http://map.iosiosr.cn:8080/api/check

# 检查位置状态
curl http://map.iosiosr.cn:8080/api/map/status
```

### 问题4: 多用户状态混淆

```bash
# 检查数据文件
cat /opt/vpn-data/mitm-data.json

# 检查客户端 IP 获取
curl -H "X-Forwarded-For: 10.10.0.1, 1.2.3.4" \
     http://map.iosiosr.cn:8080/api/check
```

---

## 📱 性能优化

### 1. 数据持久化优化

- MITM 代理数据每 30 秒自动保存
- 位置服务数据在修改后异步保存
- 使用 JSON 格式存储，便于调试

### 2. 连接池优化

```go
// 配置合适的连接池大小
http.DefaultTransport.MaxIdleConns = 100
http.DefaultTransport.MaxIdleConnsPerHost = 10
```

### 3. 日志级别

```go
// 生产环境使用 Info 级别
// 开发环境使用 Debug 级别
log.SetLevel(log.InfoLevel)
```

---

## 🔒 备份策略

### 自动备份脚本

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/opt/vpn-data/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 创建备份
mkdir -p $BACKUP_DIR

# 备份数据
cp -r /opt/vpn-data $BACKUP_DIR/$TIMESTAMP

# 保留最近 7 天
find $BACKUP_DIR -type d -mtime +7 -exec rm -rf {} \;

echo "Backup completed: $TIMESTAMP"
```

### 定时备份

```bash
# 添加到 crontab
crontab -e "0 2 * * * /opt/vpn-location/backup.sh"
```

---

## 📚 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| 后端 | Go | 1.22+ |
| 前端 | Vue.js | 3.x |
| 构建工具 | Vite | 5.x |
| VPN | StrongSwan | 5.9+ |
| Web 服务器 | Nginx | 1.18+ |
| 代理 | goproxy | 2.x |
| 地图 | Leaflet | 1.9+ |

---

## 📞 支持的 iOS 版本

| iOS 版本 | 状态 | 说明 |
|----------|------|------|
| iOS 17 | ✅ 支持 | 完全支持 |
| iOS 18 | ✅ 支持 | 完全支持 |
| iOS 19 | ✅ 支持 | 完全支持 |
| iOS 20 | ✅ 支持 | 完全支持 |
| iOS 21 | ✅ 支持 | 完全支持 |
| iOS 22 | ✅ 支持 | 完全支持 |
| iOS 23 | ✅ 支持 | 完全支持 |
| iOS 24 | ✅ 支持 | 完全支持 |
| iOS 25 | ✅ 支持 | 完全支持 |
| iOS 26 | ✅ 支持 | 完全支持 |
| iOS 27 | ✅ 支持 | 完全支持 |

---

## 🌐 支持的定位域名

### Apple 定位服务

- `gs-loc.apple.com`
- `gs-loc-cn.apple.com`
- `location.apple.com`
- `location-cn.apple.com`
- `*.ls.apple.com`
- `gsp-ssl.ls.apple.com`
- `ls-cn.apple.com`

### 排除的域名（直连）

- `*.autonavi.com` (高德地图)
- `*.amap.com` (高德地图)
- `*.weixin.qq.com` (微信)
- `*.wx.qq.com` (微信)

---

## 📝 API 文档

### 公开 API

#### 1. 检查 VPN 连接

```http
GET /api/check?client_ip=10.10.0.1

Response:
{
  "connected": true,
  "message": "VPN 已连接，可以设置虚拟定位",
  "client_ip": "10.10.0.1"
}
```

#### 2. 获取地图状态

```http
GET /api/map/status?client_ip=10.10.0.1

Response:
{
  "spoofed": true,
  "hit_count": 241,
  "last_hit": 1781801200,
  "last_lat": 39.90923,
  "last_lon": 116.397428,
  "client_ip": "10.10.0.1"
}
```

#### 3. 设置虚拟位置

```http
POST /api/map/spoof?client_ip=10.10.0.1

Request:
{
  "latitude": 39.90923,
  "longitude": 116.397428
}

Response:
{
  "message": "虚拟位置设置成功",
  "latitude": 39.90923,
  "longitude": 116.397428
}
```

#### 4. 下载 VPN 配置文件

```http
POST /api/vpn/download

Response:
Content-Type: application/x-apple-aspen-config
Content-Disposition: attachment; filename=profile.mobileconfig

# iOS .mobileconfig 文件内容
```

### 需要认证的 API

#### 1. 用户注册

```http
POST /api/register

Request:
{
  "username": "newuser",
  "password": "password123"
}

Response:
{
  "id": "user_id",
  "username": "newuser",
  "vpn_user": "vpn_abc12345",
  "vpn_pass": "random123456789",
  "created_at": 1781801200
}
```

#### 2. 用户登录

```http
POST /api/login

Request:
{
  "username": "newuser",
  "password": "password123"
}

Response:
{
  "id": "user_id",
  "username": "newuser",
  "vpn_user": "vpn_abc12345",
  "vpn_pass": "random123456789",
  "token": "jwt_token_here"
}
```

---

## 🔐 权限管理

### 目录权限

```bash
# 数据目录
sudo chown -R root:root /opt/vpn-data
sudo chmod -R 755 /opt/vpn-data

# 配置文件
sudo chown root:root /etc/ipsec.*
sudo chmod 600 /etc/ipsec.secrets
```

### 文件权限

```bash
# CA 证书
sudo chmod 644 vpn-config/*.pem

# 私钥文件
sudo chmod 600 vpn-config/*Key.pem

# 日志文件
sudo chmod 644 /var/log/vpn-location/*.log
```

---

## 🌍 测试

### 1. 功能测试

```bash
# 测试 VPN 连接
curl http://map.iosiosr.cn:8080/api/check

# 测试 Web API
curl http://map.iosiosr.cn:8080/health

# 测试 PAC 文件
curl http://map.iosiosr.cn:8080/proxy.pac
```

### 2. 性能测试

```bash
# 压力测试
ab -n 1000 -c 10 http://map.iosiosr.cn:8080/health

# 内存使用
free -h

# 磁盘使用
df -h
```

---

## 📞 维护

### 日常维护任务

```bash
# 1. 清理旧日志
find /var/log/vpn-location -name "*.log" -mtime +30 -delete

# 2. 清理临时文件
find /tmp -name "vpn-*" -mtime +1 -delete

# 3. 检查磁盘空间
df -h | grep -E "(/|Avail)"

# 4. 检查服务状态
systemctl is-active vpn-location
systemctl is-active strongswan-starter
systemctl is-active nginx
```

### 监控脚本

```bash
#!/bin/bash
# monitor.sh

while true; do
    # 检查 VPN 服务
    if ! systemctl is-active strongswan-starter; then
        echo "VPN service is down!"
        systemctl restart strongswan-starter
    fi
    
    # 检查 Web 服务
    if ! systemctl is-active vpn-location; then
        echo "Web service is down!"
        systemctl restart vpn-location
    fi
    
    # 检查磁盘空间
    USAGE=$(df /opt/vpn-data | tail -1 | awk '{print $5}' | sed 's/%//')
    if (( $(echo "$USAGE > 90") )); then
        echo "Disk usage is high: $USAGE%"
    fi
    
    sleep 60
done
```

---

## 📞 故障恢复

### 紧急恢复流程

```bash
# 1. 停止所有服务
sudo systemctl stop vpn-location
sudo systemctl stop strongswan-starter
sudo systemctl stop nginx

# 2. 恢复配置
sudo cp /opt/vpn-data/config.backup /etc/ipsec.conf
sudo cp /opt/vpn-data/secrets.backup /etc/ipsec.secrets

# 3. 重启服务
sudo systemctl start strongswan-starter
sleep 5
sudo systemctl start nginx
sleep 2
sudo systemctl start vpn-location

# 4. 验证服务
systemctl status strongswan-starter
systemctl status nginx
systemctl status vpn-location
```

---

## 📞 联系方式

### 技术支持

- GitHub Issues: https://github.com/qq1016127192/vpn-location-spoofer/issues
- 文档: https://github.com/qq1016127192/vpn-location-spoofer/wiki

### 社区资源

- 项目 Wiki: https://github.com/qq1016127192/vpn-location-spoofer/wiki
- 部署指南: https://github.com/qq1016127192/vpn-location-spoofer/blob/main/docs/deploy-full.md

---

## 🎯 最佳实践

### 1. 安全建议

- ✅ 使用强密码（至少