# VPN虚拟定位项目部署指南

## 项目架构分析

### 组件说明
- **前端**: Vue.js + Vite，静态文件
- **后端**: Go + Gin框架，提供HTTP API(8080) + MITM代理(8081)
- **数据库**: MySQL
- **特殊功能**: 
  - MITM HTTPS代理拦截Apple定位请求
  - IPsec VPN配置
  - 动态生成.mobileconfig文件
  - 自定义CA证书管理

### 部署限制
- ❌ **Cloudflare Pages**: 仅支持静态前端，无法运行后端
- ❌ **GitHub Pages**: 仅支持静态网站，无法运行后端
- ❌ **Cloudflare Workers**: 不支持原始TCP连接和TLS拦截
- ✅ **VPS/云服务器**: 完全支持所有功能

## 推荐部署方案

### 方案1: VPS/云服务器（推荐）
**适用场景**: 完整功能部署，完全控制权

**服务器要求**:
- 操作系统: Linux (Ubuntu 20.04+ / CentOS 7+)
- CPU: 2核+
- 内存: 2GB+
- 带宽: 10Mbps+
- 公网IP: 必需

**软件依赖**:
- Go 1.19+
- Node.js 16+
- MySQL 8.0+
- Nginx
- StrongSwan (IPsec)

### 方案2: 分离部署
**适用场景**: 前端静态托管 + 后端VPS

**前端部署**:
- Cloudflare Pages / Vercel / Netlify
- 仅托管静态Vue文件

**后端部署**:
- VPS / Railway / Render / Fly.io
- 运行Go后端 + 数据库

### 方案3: 混合部署（成本最低）
**前端**: Cloudflare Pages (免费)
**后端**: 低价VPS ($5-10/月)

## 详细部署步骤（方案1）

### 1. 服务器准备

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装基础依赖
sudo apt install -y git curl wget nginx mysql-server strongswan-starter libcharon-extra-plugins

# 安装Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 安装Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs
```

### 2. 数据库配置

```bash
# 启动MySQL
sudo systemctl start mysql
sudo systemctl enable mysql

# 创建数据库和用户
sudo mysql -u root -p
```

```sql
CREATE DATABASE vpn_location CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'vpn_user'@'localhost' IDENTIFIED BY 'your_secure_password';
GRANT ALL PRIVILEGES ON vpn_location.* TO 'vpn_user'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

### 3. 项目部署

```bash
# 克隆项目
cd /opt
sudo git clone <your-repo-url> vpn-location-spoofer
cd vpn-location-spoofer

# 配置后端
cd backend
cp config.yaml config.yaml.backup
# 编辑config.yaml，修改数据库连接、服务器地址等
nano config.yaml
```

**config.yaml关键配置**:
```yaml
server:
  port: 8080
  mode: release  # 生产环境改为release

database:
  host: 127.0.0.1
  port: 3306
  username: vpn_user
  password: your_secure_password
  database: vpn_location

vpn:
  server_address: your-server-ip  # 改为服务器公网IP
  ipsec_bin: /usr/sbin/ipsec
  ipsec_secrets: /etc/ipsec.secrets

cert:
  root_ca_path: /etc/ipsec.d/cacerts/caCert.pem
  root_key_path: /etc/ipsec.d/private/caKey.pem

auth:
  jwt_secret: your_random_jwt_secret_key
```

### 4. 证书配置

```bash
# 创建证书目录
sudo mkdir -p /etc/ipsec.d/cacerts
sudo mkdir -p /etc/ipsec.d/private

# 复制证书文件
sudo cp vpn-config/caCert.pem /etc/ipsec.d/cacerts/
sudo cp vpn-config/caKey.pem /etc/ipsec.d/private/
sudo chmod 600 /etc/ipsec.d/private/caKey.pem
```

### 5. 构建和启动后端

```bash
# 构建后端
go build -o backend

# 测试运行
./backend

# 如果正常运行，创建systemd服务
sudo nano /etc/systemd/system/vpn-location.service
```

**systemd服务配置**:
```ini
[Unit]
Description=VPN Location Service
After=network.target mysql.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vpn-location-spoofer/backend
ExecStart=/opt/vpn-location-spoofer/backend/backend
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable vpn-location
sudo systemctl start vpn-location
sudo systemctl status vpn-location
```

### 6. 构建和部署前端

```bash
cd /opt/vpn-location-spoofer/frontend

# 安装依赖
npm install

# 构建生产版本
npm run build

# 配置Nginx
sudo nano /etc/nginx/sites-available/vpn-location
```

**Nginx配置**:
```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端静态文件
    location / {
        root /opt/vpn-location-spoofer/frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    # API代理
    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # 搜索API代理
    location /search {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # 代理PAC文件
    location /proxy.pac {
        proxy_pass http://localhost:8080;
    }
}
```

```bash
# 启用配置
sudo ln -s /etc/nginx/sites-available/vpn-location /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### 7. 防火墙配置

```bash
# 开放必要端口
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 500/udp   # IPsec
sudo ufw allow 4500/udp  # IPsec NAT-T
sudo ufw allow 8080/tcp # 后端API（可选，通过Nginx代理）
sudo ufw allow 8081/tcp # MITM代理（可选，内网使用）

sudo ufw enable
```

### 8. SSL证书配置（推荐）

```bash
# 安装Certbot
sudo apt install certbot python3-certbot-nginx

# 获取SSL证书
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo certbot renew --dry-run
```

### 9. IPsec配置

```bash
# 配置IPsec
sudo nano /etc/ipsec.conf
```

**ipsec.conf配置**:
```config
config setup
    charondebug="ike 1, knf 1, cfg 0"
    uniqueids=no

conn ike
    keyexchange=ikev2
    ike=aes256-sha256-modp2048!
    esp=aes256-sha256!
    dpdaction=clear
    dpdinterval=30s
    dpdtimeout=120s
    left=%any
    leftid=@your-server-ip
    leftcert=serverCert.pem
    leftsendcert=always
    leftsubnet=0.0.0.0/0
    right=%any
    rightid=%any
    rightauth=eap-mschapv2
    rightdns=no
    rightmodecfgserver=no
    eap_identity=%any
    auto=add
```

```bash
# 重启IPsec服务
sudo ipsec restart
sudo systemctl enable strongswan-starter
```

## 监控和维护

### 日志查看
```bash
# 后端日志
sudo journalctl -u vpn-location -f

# Nginx日志
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log

# IPsec日志
sudo tail -f /var/log/syslog | grep ipsec
```

### 服务管理
```bash
# 重启后端
sudo systemctl restart vpn-location

# 重启Nginx
sudo systemctl restart nginx

# 重启IPsec
sudo ipsec restart
```

### 数据库备份
```bash
# 备份数据库
mysqldump -u vpn_user -p vpn_location > backup_$(date +%Y%m%d).sql

# 恢复数据库
mysql -u vpn_user -p vpn_location < backup_20240101.sql
```

## 安全建议

1. **修改默认密码**: 更改数据库密码、JWT密钥等
2. **配置防火墙**: 只开放必要端口
3. **使用SSL**: 启用HTTPS加密
4. **定期更新**: 保持系统和依赖更新
5. **监控日志**: 定期检查异常访问
6. **备份数据**: 定期备份数据库和配置文件

## 故障排查

### 后端无法启动
```bash
# 检查端口占用
sudo netstat -tlnp | grep 8080

# 检查日志
sudo journalctl -u vpn-location -n 50
```

### 数据库连接失败
```bash
# 测试数据库连接
mysql -u vpn_user -p -h 127.0.0.1 vpn_location

# 检查MySQL状态
sudo systemctl status mysql
```

### IPsec连接失败
```bash
# 检查IPsec状态
sudo ipsec status

# 检查防火墙
sudo ufw status
```

## 成本估算

### VPS方案
- **低价VPS**: $5-10/月 (DigitalOcean, Linode, Vultr)
- **中配VPS**: $20-40/月 (更好的性能)
- **域名**: $10-15/年
- **SSL证书**: 免费 (Let's Encrypt)

### 分离部署方案
- **前端**: 免费 (Cloudflare Pages)
- **后端VPS**: $5-10/月
- **域名**: $10-15/年

## 总结

由于项目需要MITM代理、IPsec VPN等系统级功能，**必须使用VPS或云服务器部署**。Cloudflare Pages和GitHub Pages只能部署前端静态文件，无法满足完整功能需求。

推荐使用方案1（VPS完整部署）以获得最佳性能和功能完整性。
