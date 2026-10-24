# VPN 虚拟定位系统
会飞的鱼 QQ 1016127192
基于 iOS IKEv2 VPN + MITM 代理技术的 GPS 虚拟定位方案。iOS 设备连接 VPN 后，系统自动拦截并替换 Apple 定位服务的响应，实现地图选点定位。

## 项目结构

```
vpn-location-spoofer/
├── backend/              # Go 后端（Gin）
│   ├── api/              # HTTP 接口
│   ├── mitm/             # MITM HTTPS 代理（拦截 gs-loc.apple.com）
│   ├── nat/              # IP 转发 / NAT 配置
│   ├── service/          # 业务逻辑（证书、mobileconfig、定位）
│   └── config.yaml       # 服务配置
├── frontend/             # Vue.js 3 前端
│   └── src/views/        # 页面：地图选点 / VPN下载 / 帮助
├── vpn-config/           # StrongSwan 配置模板
│   ├── ipsec.conf        # IKEv2 连接配置
│   ├── ipsec.secrets     # EAP 认证凭据
│   └── generate-certs.sh # 证书生成脚本
├── package.sh            # 本地打包脚本
├── server-setup.sh       # 服务器一键部署脚本
└── deploy.sh             # SSH 远程部署脚本（可选）
```

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue.js 3 + Vue Router + TailwindCSS + Leaflet 地图 |
| 后端 | Go 1.23 + Gin + GORM + MySQL |
| VPN | StrongSwan IKEv2 + EAP-MSCHAPv2 |
| 代理 | goproxy MITM + Protobuf 解析替换 |
| 证书 | OpenSSL 自签 CA + iOS 描述文件 |

## 工作原理

1. iOS 安装 `.mobileconfig` 描述文件，信任自签 CA，连接 IKEv2 VPN
2. PAC 文件将 `gs-loc.apple.com` 流量路由到后端 8081 端口
3. MITM 代理解密 HTTPS 流量，解析 Apple 定位 Protobuf 请求
4. 将所有 WiFi/基站位置数据替换为用户在地图上选取的坐标
5. 返回伪造响应，iOS 系统定位显示指定位置

## 快速部署（Linux VPS）

### 方式一：打包上传

```bash
# 本地打包（Mac）
bash package.sh
# 上传
scp vpn-package.tar.gz root@你的服务器IP:/root/
# 服务器上执行
tar -xzf vpn-package.tar.gz
SERVER_IP=你的服务器IP bash server-setup.sh
```

### 方式二：SSH 一键部署

```bash
SERVER_IP=你的服务器IP bash deploy.sh
```

### 服务器要求

- Debian 11 / Ubuntu 20.04+
- 2 核 / 2GB 内存 / 公网 IP
- 开放端口：`500/UDP`、`4500/UDP`、`8080/TCP`

## 本地开发（macOS）

```bash
# 生成证书并部署到本地 StrongSwan
SERVER_IP=192.168.10.7 bash vpn-config/generate-certs.sh

# 启动后端
cd backend && go run .

# 启动前端
cd frontend && npm install && npm run dev
```

> macOS 上 `racoon` 会抢占 UDP 500 端口，`generate-certs.sh` 已自动配置 `pf` 重定向规则（500→5500）绕过此问题。

## iOS 使用步骤

1. Safari 打开 `http://服务器IP:8080`
2. 点击「下载 VPN 配置」，安装描述文件
3. 设置 → 通用 → 关于本机 → 证书信任设置 → 信任新 CA
4. 连接 VPN（账号 `vpnuser` / 密码 `vpnpass2024`）
5. 在 Web 地图上选点，点击「设置虚拟定位」

## VPN 账号

| 账号 | 密码 |
|------|------|
| vpnuser | vpnpass2024 |

> 生产环境建议修改 `vpn-config/ipsec.secrets` 和 `backend/config.yaml` 中的凭据。

## 注意事项

- 仅支持 iOS 设备
- 需要与 VPN 服务器在同一网络，或服务器有公网 IP
- 必须完整信任 CA 证书（设置中启用完全信任）
- 仅供学习研究使用
