# 部署教程 · IP 版

适用于直接用服务器公网 IP 部署，无需域名。

---

## 环境要求

- Ubuntu 22.04 / Debian 11
- 宝塔面板（已装 MySQL、Node.js 18+）
- Go 1.21+（宝塔或手动安装）
- 开放端口：`8080/tcp`、`500/udp`、`4500/udp`

---

## 第一步：上传项目并解压

把项目压缩包上传到服务器，SSH 执行：

```bash
cd /www/wwwroot
unzip "vpn-location-spoofer.zip" -d tmp_vpn
mv tmp_vpn/vpn-location-spoofer /www/wwwroot/vpn-app
rm -rf tmp_vpn
ls /www/wwwroot/vpn-app
# 确认有 backend/ frontend/ vpn-config/
```

---

## 第二步：安装 StrongSwan

```bash
apt-get update
apt-get install -y strongswan strongswan-pki libcharon-extra-plugins
ipsec version
```

---

## 第三步：生成证书

```bash
cd /www/wwwroot/vpn-app

# 替换为你的服务器公网 IP
SERVER_IP=103.231.59.141 bash vpn-config/generate-certs.sh

# 部署到 StrongSwan
mkdir -p /etc/ipsec.d/{certs,private,cacerts}
cp vpn-config/serverCert.pem /etc/ipsec.d/certs/
cp vpn-config/caCert.pem     /etc/ipsec.d/cacerts/
cp vpn-config/serverKey.pem  /etc/ipsec.d/private/
cp vpn-config/caKey.pem      /etc/ipsec.d/private/
chmod 600 /etc/ipsec.d/private/*.pem
```

---

## 第四步：配置 StrongSwan

> ⚠️ 云服务器 NAT 结构：`left` 填**内网 IP**，`leftid` 填**公网 IP**

```bash
# 查询内网 IP
INNER_IP=$(ip route get 1 | awk '{print $7;exit}')
echo "内网 IP: $INNER_IP"

cat > /etc/ipsec.conf << EOF
config setup
    charondebug="ike 2, knl 2, cfg 2, net 2, esp 2, tls 2, lib 2"
    uniqueids=no

conn %default
    keyexchange=ikev2
    ike=aes256-sha256-modp2048!
    esp=aes256-sha256!
    dpdaction=clear
    dpddelay=300s
    rekey=no
    mobike=yes
    forceencaps=yes
    fragmentation=yes

conn ikev2-eap
    left=$INNER_IP
    leftid=103.231.59.141
    leftcert=serverCert.pem
    leftsendcert=always
    leftsubnet=0.0.0.0/0
    right=%any
    rightid=%any
    rightauth=eap-mschapv2
    rightsourceip=10.10.0.0/24
    rightdns=8.8.8.8,8.8.4.4
    eap_identity=%any
    auto=add
EOF

cp vpn-config/ipsec.secrets /etc/ipsec.secrets
chmod 600 /etc/ipsec.secrets

# 禁用 half-open 连接限制
CHARON_CONF=$(find /etc/strongswan.d -name "charon.conf" | head -1)
cat > $CHARON_CONF << 'EOF'
charon {
    block_threshold = 0
    half_open_timeout = 15
    install_routes = yes
    crypto_test {}
    host_resolver {}
    processor { priority_threads {} }
    start-scripts {}
    stop-scripts {}
    tls {}
    x509 {}
}
EOF

# sudoers 允许后端热重载
echo "root ALL=(ALL) NOPASSWD: /usr/sbin/ipsec rereadsecrets" > /etc/sudoers.d/ipsec-reload
chmod 440 /etc/sudoers.d/ipsec-reload

# 启动
systemctl enable strongswan-starter
systemctl restart strongswan-starter
sleep 2
ipsec statusall | grep -E "ikev2-eap|local:|cert:"
```

---

## 第五步：IP 转发和 NAT

```bash
echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf
sysctl -w net.ipv4.ip_forward=1

OUTIF=$(ip route | grep default | awk '{print $5}' | head -1)
echo "出口网卡: $OUTIF"
iptables -t nat -A POSTROUTING -s 10.10.0.0/24 -o $OUTIF -j MASQUERADE

apt-get install -y iptables-persistent
netfilter-persistent save
```

---

## 第六步：UFW 防火墙

```bash
ufw allow 500/udp
ufw allow 4500/udp
ufw allow 8080/tcp
sed -i 's/DEFAULT_FORWARD_POLICY="DROP"/DEFAULT_FORWARD_POLICY="ACCEPT"/' /etc/default/ufw
ufw reload
```

---

## 第七步：修改 config.yaml

```bash
cd /www/wwwroot/vpn-app
JWT_SECRET=$(openssl rand -hex 32)

cat > backend/config.yaml << EOF
server:
  port: 8080
  mode: release

database:
  host: 127.0.0.1
  port: 3306
  username: 你的数据库用户名
  password: 你的数据库密码
  database: 你的数据库名
  charset: utf8mb4

vpn:
  server_address: 103.231.59.141
  ipsec_bin: /usr/sbin/ipsec
  ipsec_secrets: /etc/ipsec.secrets

cert:
  root_ca_path: /etc/ipsec.d/cacerts/caCert.pem
  root_key_path: /etc/ipsec.d/private/caKey.pem

auth:
  jwt_secret: ${JWT_SECRET}
  expire_hours: 24
EOF
```

数据库在宝塔面板建好后填入用户名/密码/库名，后端启动时自动建表。

---

## 第八步：构建前端

```bash
cd /www/wwwroot/vpn-app/frontend
rm -rf node_modules package-lock.json
npm install
npm run build
```

---

## 第九步：编译后端

```bash
cd /www/wwwroot/vpn-app/backend
go build -o vpn-server .
```

---

## 第十步：创建 systemd 服务

```bash
cat > /etc/systemd/system/vpn-location.service << 'EOF'
[Unit]
Description=VPN Location Server
After=network.target

[Service]
User=root
WorkingDirectory=/www/wwwroot/vpn-app/backend
ExecStart=/www/wwwroot/vpn-app/backend/vpn-server
Restart=always
RestartSec=5
Environment=FRONTEND_DIST=/www/wwwroot/vpn-app/frontend/dist

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable vpn-location
systemctl start vpn-location
sleep 3
curl http://127.0.0.1:8080/health
# 返回 {"status":"ok"} 即成功
```

---

## 验证

```bash
curl http://127.0.0.1:8080/health           # 后端健康
ipsec statusall | grep ikev2-eap            # VPN 配置加载
ss -tulnp | grep -E "8080|8081|500|4500"   # 端口监听
```

---

## iOS 使用步骤

1. Safari 打开 `http://103.231.59.141:8080`
2. 点击「下载 VPN 配置文件」→ 允许下载
3. **设置 → 通用 → VPN与设备管理** → 安装描述文件
4. **设置 → 通用 → 关于本机 → 证书信任设置** → 打开 VPN Root CA 信任
5. **设置 → VPN** → 连接

---

## 常见问题

**VPN 连接失败 / retransmit 死循环**
- `ipsec.conf` 的 `left` 必须填内网 IP，不能填公网 IP

**连接提示 NO_PROPOSAL_CHOSEN**
- 证书 CN 与 `leftid` 不一致，重新生成证书并重装 mobileconfig

**iOS 找不到证书信任设置**
- 描述文件未安装成功，先确认安装完成再去信任设置

**更新服务**
```bash
# 重新编译并重启
cd /www/wwwroot/vpn-app/backend
go build -o vpn-server .
systemctl restart vpn-location
```

**切换 IP 或换域名**
```bash
bash /www/wwwroot/vpn-app/switch-server.sh 新IP或域名
# 执行完后 iOS 删旧描述文件重新下载
```
