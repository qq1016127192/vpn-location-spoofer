# macOS 本地部署教程

在 macOS 上本地部署 VPN 虚拟定位系统，用于开发调试或局域网内 iOS 设备使用。

---

## 一、环境要求

| 项 | 要求 |
|---|---|
| 系统 | macOS 12+（推荐 macOS 13+） |
| 芯片 | Apple Silicon / Intel 均可 |
| 内存 | 4GB+ |
| StrongSwan | `brew install strongswan` |
| Go | 1.23+（`brew install go`） |
| Node.js | 18+（`brew install node`） |
| OpenSSL | 系统自带或 `brew install openssl` |
| 网络 | macOS 与 iOS 设备在同一局域网，或 macOS 有公网 IP |

### 安装依赖

```bash
brew install strongswan go node openssl
```

---

## 二、关键端口

| 端口 | 协议 | 用途 |
|---|---|---|
| 500 | UDP | IKEv2 VPN（macOS 上会被 racoon 抢占，需 pf 重定向到 5500） |
| 4500 | UDP | IKEv2 NAT-T（同上，重定向到 5501） |
| 5500 | UDP | StrongSwan charon 实际监听 |
| 5501 | UDP | StrongSwan charon 实际监听 |
| 8080 | TCP | 后端 HTTP API + 前端静态文件 |
| 8081 | TCP | MITM HTTPS 代理（拦截 gs-loc.apple.com） |

---

## 三、部署步骤

### Step 1 — 获取本机 IP

```bash
# 局域网 IP（iOS 设备将访问这个 IP）
ipconfig getifaddr en0
# 假设输出: 192.168.10.7
```

记下这个 IP，后续所有命令都用 `SERVER_IP=192.168.10.7`（替换成你的实际 IP）。

### Step 2 — 生成证书 + 部署到 StrongSwan

项目自带的 `generate-certs.sh` 会一次性完成：
1. 生成 CA 私钥/证书、服务器私钥/证书（带 IP SAN，iOS 16+ 必须）
2. 同步 CA 到 `vpn.mobileconfig`
3. 部署证书到 StrongSwan 目录（`/opt/homebrew/etc/ipsec.d/` 和 `swanctl/`）
4. 替换 `ipsec.conf` / `swanctl.conf` 里的 `leftid` 为你的 IP
5. 配置 pf 重定向规则（绕过 racoon 抢占 500 端口）

```bash
cd vpn-location-spoofer
SERVER_IP=192.168.10.7 bash vpn-config/generate-certs.sh
```

> **需要 sudo**：脚本会写 `/etc/pf.anchors/` 和 `/etc/pf.conf`，会提示输入密码。

### Step 3 — 启动 StrongSwan

```bash
sudo /opt/homebrew/bin/ipsec start
sleep 2

# 验证
sudo /opt/homebrew/bin/ipsec statusall | grep -E "ikev2-eap|Listening"
```

正常输出应包含：
```
Listening on IP address 192.168.10.7 port 500
Listening on IP address 192.168.10.7 port 4500
Connections:
        ikev2-eap:  ...
```

### Step 4 — 启动后端

```bash
cd backend
go run .
```

后端启动后会：
- 在 `:8080` 启动 HTTP API + 托管前端
- 在 `:8081` 启动 MITM HTTPS 代理
- 尝试配置 NAT（需要 sudo，非 root 会跳过，不影响 VPN 连接，只影响客户端出网）

### Step 5 — 启动前端（开发模式）

```bash
cd frontend
npm install
npm run dev
```

Vite 跑在 `http://localhost:3000`，API 自动反代到 8080。

### Step 6 — 构建前端（让 iOS 直接访问 8080）

如果要让 iOS 设备直接访问 `http://192.168.10.7:8080` 看到完整页面：

```bash
cd frontend
npm run build
```

构建产物在 `frontend/dist/`，后端会自动托管。

---

## 四、iOS 设备配置

### 1. 下载描述文件

iOS Safari 打开：
```
http://192.168.10.7:8080/api/vpn/download
```

Safari 提示「此网站正尝试下载一个配置描述文件」→ 点允许。

### 2. 安装描述文件

```
设置 → 通用 → VPN与设备管理 → 找到描述文件 → 安装
```

### 3. 信任根证书（关键步骤，不做则 MITM 失败）

```
设置 → 通用 → 关于本机 → 证书信任设置 → 开启 "VPN Root CA" 完全信任
```

> 安装描述文件后才会出现「证书信任设置」选项。

### 4. 连接 VPN

```
设置 → VPN → 选择 "wzgo.cc 虚拟定位" → 连接
```

VPN 账号：`vpnuser` / 密码：`vpnpass2024`

### 5. 设置虚拟定位

1. iOS Safari 打开 `http://192.168.10.7:8080`
2. 进入地图页，准星对准目标位置
3. 点「锁定此位置」
4. 设置 → 隐私 → 定位服务 → 关闭再打开（刷新定位）
5. 打开地图 App 验证位置已改变

---

## 五、日常启动顺序（每次重启 Mac 后）

```bash
# 1. 启动 StrongSwan（需要 sudo）
sudo /opt/homebrew/bin/ipsec start

# 2. 启动后端（终端 1）
cd backend && go run .

# 3. 启动前端（终端 2，可选，开发时用）
cd frontend && npm run dev
```

停止服务：

```bash
# 停 StrongSwan
sudo /opt/homebrew/bin/ipsec stop

# 停后端：Ctrl+C

# 停前端：Ctrl+C
```

---

## 六、验证与诊断

### 一键诊断脚本

```bash
# 检查 StrongSwan 状态
sudo /opt/homebrew/bin/ipsec statusall

# 检查端口监听
sudo lsof -iUDP:500 -iUDP:4500 -iUDP:5500 -iUDP:5501

# 检查 pf 规则
sudo pfctl -s nat | grep -E "5500|5501"

# 检查后端健康
curl http://localhost:8080/health

# 检查 mobileconfig 下载
curl -o /tmp/test.mobileconfig http://localhost:8080/api/vpn/download
ls -la /tmp/test.mobileconfig
```

### 证书一致性检查（关键！）

三处 CA 指纹必须一致，否则 VPN 连不上：

```bash
echo "=== 后端 CA ==="
openssl x509 -in vpn-config/caCert.pem -noout -fingerprint -sha256

echo "=== StrongSwan CA ==="
openssl x509 -in /opt/homebrew/etc/ipsec.d/cacerts/caCert.pem -noout -fingerprint -sha256

echo "=== mobileconfig 里的 CA ==="
python3 -c "
import re, base64, subprocess
with open('vpn.mobileconfig') as f: c = f.read()
m = re.search(r'<key>PayloadContent</key>\s*<data>([^<]+)</data>', c)
open('/tmp/mc.der','wb').write(base64.b64decode(m.group(1)))
subprocess.run(['openssl','x509','-inform','DER','-in','/tmp/mc.der','-noout','-fingerprint','-sha256'])
"
```

三个指纹必须完全相同。

---

## 七、常见问题

### Q1: VPN 连接失败，提示「IKE_SA 建立失败」

**原因 1：StrongSwan 未启动**

```bash
sudo /opt/homebrew/bin/ipsec statusall
# 如果无输出，说明没启动
sudo /opt/homebrew/bin/ipsec start
```

**原因 2：500 端口被 racoon 抢占**

macOS 自带的 `racoon` 会抢占 UDP 500 端口，导致 StrongSwan 收不到 IKE 请求。

```bash
# 检查 500 端口占用
sudo lsof -iUDP:500

# 如果是 racoon 占用，需要 pf 重定向（generate-certs.sh 已自动配置）
sudo pfctl -f /etc/pf.conf
sudo pfctl -e

# 验证重定向规则
sudo pfctl -s nat | grep -E "5500|5501"
```

**原因 3：CA 证书不一致**

三处 CA（后端、StrongSwan、mobileconfig）指纹必须一致。重新跑：
```bash
SERVER_IP=192.168.10.7 bash vpn-config/generate-certs.sh
```

### Q2: VPN 连上了，但定位没变

**原因 1：iOS 没信任 CA 证书**

```
设置 → 通用 → 关于本机 → 证书信任设置 → 开启完全信任
```

**原因 2：没刷新定位服务**

```
设置 → 隐私 → 定位服务 → 关闭 → 重新打开
```

**原因 3：MITM 代理没拦截到**

检查后端日志，应该有 `[MITM] 客户端 10.10.0.x 替换位置 -> (lat, lon)` 输出。如果没有：
- 确认 iOS 走的是 VPN 流量（不是 WiFi 直连）
- 确认 PAC 文件生效：iOS Safari 访问 `http://192.168.10.7:8080/proxy.pac` 应返回 JS

### Q3: VPN 连上了，但 iOS 无法上网

**原因：NAT 未配置**

后端非 root 运行时会跳过 NAT 配置，VPN 客户端无法出网。

```bash
# 方案 1: sudo 启动后端
cd backend && sudo go run .

# 方案 2: 手动配置 pf NAT
sudo tee /tmp/vpn-nat.conf > /dev/null << EOF
nat on en0 from 10.10.0.0/16 to any -> (en0)
EOF
sudo pfctl -a vpn-nat -f /tmp/vpn-nat.conf
```

### Q4: 描述文件安装后「VPN」选项里看不到配置

**原因：描述文件未正确安装或 CA 未信任**

1. 设置 → 通用 → VPN与设备管理 → 确认描述文件已安装
2. 设置 → 通用 → 关于本机 → 证书信任设置 → 开启信任
3. 重新下载安装描述文件

### Q5: iOS 16+ 连接失败

**原因：服务器证书缺少 SAN 扩展**

iOS 16+ 强制要求证书包含 Subject Alternative Name。`generate-certs.sh` 已自动添加 IP SAN，但如果手动生成证书，需加 `-extfile`：

```bash
cat > /tmp/server_ext.cnf << EOF
subjectAltName = IP:192.168.10.7
extendedKeyUsage = serverAuth
EOF

openssl x509 -req -in serverReq.pem \
  -CA caCert.pem -CAkey caKey.pem -CAcreateserial \
  -out serverCert.pem -days 3650 \
  -extfile /tmp/server_ext.cnf
```

### Q6: 修改了 SERVER_IP 后连不上

IP 变了需要重新生成证书（CN 和 SAN 都绑定了 IP）：

```bash
SERVER_IP=新IP bash vpn-config/generate-certs.sh
sudo /opt/homebrew/bin/ipsec restart
# iOS 删除旧描述文件，重新下载安装
```

---

## 八、目录结构说明

```
vpn-location-spoofer/
├── backend/                    # Go 后端
│   ├── main.go                 # 入口：启动 MITM + HTTP
│   ├── config.yaml             # 配置（SERVER_IP、证书路径）
│   ├── mitm/                   # MITM 代理核心
│   │   ├── proxy.go            # HTTPS 拦截 + Protobuf 篡改
│   │   ├── arpc.go             # Apple ARPC 协议解析
│   │   └── pb/                 # Protobuf 定义
│   ├── nat/setup.go            # macOS pf / Linux iptables NAT
│   └── service/
│       ├── mobileconfig.go     # 生成 iOS 描述文件
│       └── cert.go             # CA 证书管理
├── frontend/                   # Vue 3 前端
│   └── src/views/MapView.vue   # 地图选点页
├── vpn-config/                 # StrongSwan 配置 + 证书
│   ├── generate-certs.sh       # 一键生成证书 + 部署
│   ├── ipsec.conf              # IKEv2 配置
│   ├── ipsec.secrets           # EAP 凭据
│   └── swanctl.conf            # swanctl 配置
├── vpn.mobileconfig            # iOS 描述文件（CA 自动同步）
└── local-dev.sh                # 本地开发启动脚本
```

---

## 九、安全注意事项

1. **VPN 账号是共享的**：`vpnuser/vpnpass2024` 硬编码在 `ipsec.secrets`，任何拿到描述文件的人都能连。生产环境请修改。
2. **CA 私钥是核心**：`vpn-config/caKey.pem` 泄露 = 任何人都能伪造你的 HTTPS 证书。权限设 600，不要提交到 git。
3. **仅限学习研究**：本方案通过 MITM 拦截 Apple 定位服务，仅供技术测试使用，禁止用于违法违规场景。

---

## 十、快速启动 Cheatsheet

```bash
# 一键启动（每次重启 Mac 后执行）
SERVER_IP=192.168.10.7 bash vpn-config/generate-certs.sh  # 首次或换 IP 时
sudo /opt/homebrew/bin/ipsec start                          # 启动 VPN
cd backend && go run . &                                    # 启动后端
cd frontend && npm run dev &                                # 启动前端（可选）

# iOS 访问
# http://192.168.10.7:8080/api/vpn/download  下载描述文件
# http://192.168.10.7:8080                    地图选点

# 一键停止
sudo /opt/homebrew/bin/ipsec stop
pkill -f "go run ." ; pkill -f "vite"
```
