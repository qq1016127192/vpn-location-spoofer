# 部署文档

## 当前生产环境

| 项目 | 值 |
|---|---|
| 服务器 IP | 116.62.147.43 |
| 域名 | wz.iosr.cn |
| Web 面板 | http://wz.iosr.cn |
| VPN 地址 | 116.62.147.43 (UDP 500/4500) |
| VPN 账号 | vpnuser / vpnpass2024 |

---

## 一、首次部署新服务器

### 前置条件
- Ubuntu 22.04 / 24.04 服务器
- 阿里云安全组放行：TCP 22、TCP 80、TCP 8080、UDP 500、UDP 4500
- 本地安装：Go 1.23+、Node.js 18+、sshpass

### 步骤

**1. 本地打包**
```bash
bash package.sh
```
生成 `vpn-package.tar.gz`

**2. 上传到服务器**
```bash
scp vpn-package.tar.gz root@服务器IP:/root/
```

**3. 服务器上部署**
```bash
ssh root@服务器IP
tar -xzf vpn-package.tar.gz
SERVER_IP=服务器IP bash server-setup.sh
```
脚本自动完成：安装依赖、生成证书、构建前端、配置 StrongSwan、启动服务

**4. 清理残留文件**
```bash
rm -rf /root/vpn-package.tar.gz /root/backend /root/frontend /root/server-setup.sh /root/check-deploy.sh /root/vpn-config /root/vpn-server
```

**5. 配置 nginx 反代（可选，绑定域名）**
```bash
apt-get install -y nginx
mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled
cat > /etc/nginx/sites-available/vpn << 'EOF'
server {
    listen 80;
    server_name 你的域名;
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
EOF
ln -sf /etc/nginx/sites-available/vpn /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

---

## 二、更新代码（仅更新后端二进制）

本地执行：
```bash
# 打包（会自动编译 Linux 二进制）
bash package.sh

# 上传
scp vpn-package.tar.gz root@116.62.147.43:/root/

# 服务器上更新
ssh root@116.62.147.43
mkdir /tmp/update && tar -xzf /root/vpn-package.tar.gz -C /tmp/update
systemctl stop vpn-location
cp /tmp/update/vpn-server /opt/vpn-location/vpn-server
chmod +x /opt/vpn-location/vpn-server
systemctl start vpn-location

# 清理
rm -rf /root/vpn-package.tar.gz /tmp/update
```

---

## 三、iOS 使用流程

1. Safari 打开 `http://wz.iosr.cn`（或服务器 IP:8080）
2. 点击「下载 VPN 配置」
3. 设置 → 通用 → VPN与设备管理 → 安装描述文件
4. 设置 → 通用 → **关于本机** → **证书信任设置** → 开启 **VPN Root CA** 完全信任
5. 连接 VPN（账号 `vpnuser`，密码 `vpnpass2024`）
6. 在 Web 面板地图上选点 → 设置虚拟定位

> 每个用户的定位是独立的，互不干扰。

---

## 四、诊断排查

**检查所有服务状态**
```bash
bash check-deploy.sh
```

**StrongSwan 状态**
```bash
ipsec statusall
```

**后端日志**
```bash
journalctl -u vpn-location -f
```

**重启所有服务**
```bash
systemctl restart strongswan-starter vpn-location nginx
```

---

## 五、重新生成证书（证书过期或更换服务器IP时）

在服务器上执行：
```bash
cd /opt/vpn-location
# 重新生成证书
openssl genrsa -out certs/caKey.pem 2048
openssl req -new -x509 -days 3650 -key certs/caKey.pem -out certs/caCert.pem \
  -subj "/C=CN/O=VPN Location Spoofer/OU=CA/CN=VPN Root CA"
openssl genrsa -out certs/serverKey.pem 2048
openssl req -new -key certs/serverKey.pem -out /tmp/serverReq.pem \
  -subj "/C=CN/O=VPN Location Spoofer/OU=Server/CN=116.62.147.43"
cat > /tmp/ext.cnf << EOF
subjectAltName = IP:116.62.147.43
extendedKeyUsage = serverAuth
EOF
openssl x509 -req -in /tmp/serverReq.pem -CA certs/caCert.pem -CAkey certs/caKey.pem \
  -CAcreateserial -out certs/serverCert.pem -days 3650 -extfile /tmp/ext.cnf

# 部署到 strongSwan
cp certs/serverCert.pem /etc/ipsec.d/certs/
cp certs/serverKey.pem  /etc/ipsec.d/private/
cp certs/caCert.pem     /etc/ipsec.d/cacerts/
systemctl restart strongswan-starter
systemctl restart vpn-location
```

然后 iOS 删除旧描述文件，重新下载安装。
