# 部署教程 · 域名版

适用于已有域名，通过域名访问和连接 VPN。

> 域名版与 IP 版的唯一区别：证书 CN、`leftid`、`server_address` 全部填域名而不是 IP。
> VPN 协议（UDP 500/4500）直连域名，不经过 Nginx。Nginx 只负责 Web 页面的反代（可选）。

---

## 环境要求

- Ubuntu 22.04 / Debian 11
- 宝塔面板（已装 MySQL、Node.js 18+）
- Go 1.21+（宝塔或手动安装）
- 域名 DNS 已解析到服务器公网 IP（A 记录已生效）
- 开放端口：`80/tcp`、`8080/tcp`、`500/udp`、`4500/udp`

---

## 第一步：确认 DNS 解析

```bash
dig vpn.wzgo.cc
# ANSWER SECTION 应显示服务器公网 IP
# 如果未生效等 DNS 传播后再继续
```

---

## 第二步：上传项目并解压

```bash
cd /www/wwwroot
unzip "vpn-location-spoofer.zip" -d tmp_vpn
mv tmp_vpn/vpn-location-spoofer /www/wwwroot/vpn-app
rm -rf tmp_vpn
ls /www/wwwroot/vpn-app
# 确认有 backend/ frontend/ vpn-config/
```

---

## 第三步：安装 StrongSwan

```bash
apt-get update
apt-get install -y strongswan strongswan-pki libcharon-extra-plugins
ipsec version
```

---

## 第四步：生成域名证书

```bash
cd /www/wwwroot/vpn-app

# 用域名生成证书（CN 设为域名）
SERVER_IP=vpn.wzgo.cc bash vpn-config/generate-certs.sh

# 部署到 StrongSwan
mkdir -p /etc/ipsec.d/{certs,private,cacerts}
cp vpn-config/serverCert.pem /etc/ipsec.d/certs/
cp vpn-config/caCert.pem     /etc/ipsec.d/cacerts/
cp vpn-config/serverKey.pem  /etc/ipsec.d/private/
cp vpn-config/caKey.pem      /etc/ipsec.d/private/
chmod 600 /etc/ipsec.d/private/*.pem

# 验证证书 CN 是域名
openssl x509 -in /etc/ipsec.d/certs/serverCert.pem -noout -subject
# 应显示 CN = vpn.wzgo.cc
```

---

## 第五步：配置 StrongSwan

> ⚠️ 云服务器 NAT 结构：`left` 填**内网 IP**，`leftid` 填**域名**

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
    leftid=vpn.wzgo.cc
    leftcert=serverCert.pem
    leftsendcert=always
    leftsubnet=0.0.0.0/0
    right=%any
    rightid=%any
    rightauth=eap-mschapv2
    rightsourceip=10.10.0.0/16
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
# local 应显示 [vpn.wzgo.cc]，cert 应显示 CN=vpn.wzgo.cc
```

---

## 第六步：IP 转发和 NAT

```bash
echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf
sysctl -w net.ipv4.ip_forward=1

OUTIF=$(ip route | grep default | awk '{print $5}' | head -1)
echo "出口网卡: $OUTIF"
iptables -t nat -A POSTROUTING -s 10.10.0.0/16 -o $OUTIF -j MASQUERADE

apt-get install -y iptables-persistent
netfilter-persistent save
```

---

## 第七步：UFW 防火墙

```bash
ufw allow 80/tcp
ufw allow 8080/tcp
ufw allow 500/udp
ufw allow 4500/udp
sed -i 's/DEFAULT_FORWARD_POLICY="DROP"/DEFAULT_FORWARD_POLICY="ACCEPT"/' /etc/default/ufw
ufw reload
```

---

## 第八步：修改 config.yaml

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
  server_address: vpn.wzgo.cc
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

---

## 第九步：构建前端

```bash
cd /www/wwwroot/vpn-app/frontend
rm -rf node_modules package-lock.json
npm install
npm run build
```

---

## 第十步：编译后端

```bash
cd /www/wwwroot/vpn-app/backend
go build -o vpn-server .
```

---

## 第十一步：创建 systemd 服务

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

## 第十二步：Nginx 反代（推荐）

配置后用户通过 `http://vpn.wzgo.cc` 访问，不需要加 `:8080`。

宝塔面板操作：
1. **网站 → 添加站点**，域名填 `vpn.wzgo.cc`，不选 PHP
2. **站点设置 → 反向代理 → 添加反向代理**
   - 目标 URL：`http://127.0.0.1:8080`
   - 发送域名：`$host`

配置完访问 `http://vpn.wzgo.cc` 验证页面正常。

> 注意：Nginx 只反代 Web（TCP 80），VPN（UDP 500/4500）直连域名，不受 Nginx 影响。

---

## 验证

```bash
curl http://127.0.0.1:8080/health           # 后端健康
ipsec statusall | grep ikev2-eap            # VPN 配置加载
ss -tulnp | grep -E "8080|8081|500|4500"   # 端口监听
```

外网验证：
```bash
# 在本机执行
curl http://vpn.wzgo.cc/health
# 返回 {"status":"ok"}
```

---

## iOS 使用步骤

1. Safari 打开 `http://vpn.wzgo.cc`（配了 Nginx 反代）或 `http://vpn.wzgo.cc:8080`
2. 点击「下载 VPN 配置文件」→ 允许下载
3. **设置 → 通用 → VPN与设备管理** → 安装描述文件
4. **设置 → 通用 → 关于本机 → 证书信任设置** → 打开 VPN Root CA 信任（安装描述文件后才出现此选项）
5. **设置 → VPN** → 连接

---

## 常见问题

**VPN 连接停在 retransmit / 超时**
- 确认云控制台安全组放行了 UDP 出方向 500/4500
- 确认 `ipsec.conf` 的 `left` 是内网 IP 而不是公网 IP 或域名

**连接提示 NO_PROPOSAL_CHOSEN**
- 证书 CN 与 `leftid` 不一致，重新生成证书并重装 mobileconfig

**iOS 找不到证书信任设置**
- 描述文件必须先安装成功，安装后才会在信任设置里出现

**重新生成证书后 VPN 仍连不上**
- iOS 上必须删除旧描述文件，重新下载安装（旧文件里嵌的是旧 CA）

**更新服务**
```bash
cd /www/wwwroot/vpn-app/backend
go build -o vpn-server .
systemctl restart vpn-location
```

**切换域名或 IP**
```bash
bash /www/wwwroot/vpn-app/switch-server.sh vpn.wzgo.cc
# 执行完后 iOS 删旧描述文件重新下载
```
