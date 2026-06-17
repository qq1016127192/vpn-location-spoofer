# VPN证书统一管理

## 问题背景

之前出现mobileconfig中的CA证书和StrongSwan服务端CA证书不匹配的问题，导致VPN连接失败。为防止此类问题再次发生，现在使用统一证书管理方案。

## 统一证书生成脚本

使用 `generate-unified-certs.sh` 脚本生成统一证书，确保证书一致性。

### 使用方法

```bash
cd vpn-config
SERVER_IP=192.168.10.7 bash generate-unified-certs.sh
```

### 脚本功能

1. **生成统一CA证书**
   - 创建CA私钥和证书
   - 用于StrongSwan服务端和mobileconfig

2. **生成服务器证书**
   - 使用统一CA签名服务器证书
   - 自动配置SAN扩展（iOS 16+要求）
   - 支持IP和DNS格式

3. **自动更新配置**
   - 更新backend/config.yaml证书路径
   - 更新ipsec.conf中的服务器地址
   - 验证证书指纹一致性

4. **权限设置**
   - 自动设置正确的文件权限
   - 私钥文件600权限
   - 证书文件644权限

## 证书文件说明

```
vpn-config/
├── caCert.pem          # CA证书（统一用于StrongSwan和mobileconfig）
├── caKey.pem           # CA私钥
├── serverCert.pem      # 服务器证书
├── serverKey.pem       # 服务器私钥
├── serverReq.pem       # 服务器证书签名请求
├── ipsec.conf          # StrongSwan配置
├── ipsec.secrets       # StrongSwan密钥配置
└── generate-unified-certs.sh  # 统一证书生成脚本
```

## 部署步骤

### 1. 生成证书

```bash
cd vpn-config
SERVER_IP=你的服务器IP bash generate-unified-certs.sh
```

### 2. 部署到StrongSwan服务端

```bash
# Linux服务器
sudo cp serverCert.pem /etc/ipsec.d/certs/
sudo cp serverKey.pem  /etc/ipsec.d/private/
sudo cp caCert.pem     /etc/ipsec.d/cacerts/
sudo cp ipsec.conf     /etc/ipsec.conf
sudo cp ipsec.secrets  /etc/ipsec.secrets
sudo ipsec restart
```

### 3. 验证证书一致性

```bash
# 检查StrongSwan服务端CA指纹
openssl x509 -in /etc/ipsec.d/cacerts/caCert.pem -noout -fingerprint -sha256

# 检查本地CA指纹
openssl x509 -in vpn-config/caCert.pem -noout -fingerprint -sha256

# 下载mobileconfig并检查CA指纹
curl -X POST http://your-server:8080/api/vpn/download \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test123"}' \
  -o test.mobileconfig

# 提取并检查mobileconfig中的CA证书指纹
# (需要手动解码base64部分)
```

### 4. 重启后端服务

```bash
cd backend
./vpn-server
```

## 证书验证

### 检查证书信息

```bash
# 查看CA证书详情
openssl x509 -in caCert.pem -text -noout

# 查看服务器证书详情
openssl x509 -in serverCert.pem -text -noout

# 验证证书链
openssl verify -CAfile caCert.pem serverCert.pem
```

### 检查证书指纹

```bash
# SHA256指纹
openssl x509 -in caCert.pem -noout -fingerprint -sha256

# SHA1指纹
openssl x509 -in caCert.pem -noout -fingerprint -sha1
```

## 故障排查

### 问题1：证书指纹不匹配

**症状：** VPN连接在IKE_AUTH阶段失败

**解决：**
1. 使用统一证书脚本重新生成证书
2. 确保StrongSwan服务端和后端使用相同的CA证书
3. 验证mobileconfig中的CA指纹与服务端一致

### 问题2：iOS 16+无法连接

**症状：** iOS 16+设备连接失败

**解决：**
1. 确保服务器证书包含SAN扩展
2. 统一证书脚本已自动配置SAN
3. 验证证书中的subjectAltName字段

### 问题3：证书过期

**症状：** 证书过期导致连接失败

**解决：**
1. 重新生成证书：`bash generate-unified-certs.sh`
2. 部署新证书到StrongSwan服务端
3. 更新mobileconfig

## 配置文件说明

### backend/config.yaml

```yaml
cert:
  root_ca_path: ./vpn-config/caCert.pem    # 统一CA证书路径
  root_key_path: ./vpn-config/caKey.pem    # 统一CA私钥路径
```

### ipsec.conf

```bash
leftid=192.168.10.7    # 必须与服务器证书CN一致
leftcert=serverCert.pem
```

## 安全建议

1. **保护私钥文件**
   - CA私钥和服务器私钥权限设置为600
   - 不要将私钥提交到版本控制

2. **定期更新证书**
   - 证书有效期10年
   - 建议定期检查证书过期时间

3. **备份证书**
   - 生成证书后立即备份
   - 安全存储CA私钥

## 维护脚本

### 重新生成证书

```bash
cd vpn-config
SERVER_IP=新服务器IP bash generate-unified-certs.sh
```

### 清理旧证书

```bash
cd vpn-config
rm -f caKey.pem caCert.pem serverKey.pem serverCert.pem serverReq.pem caCert.srl
```

## 注意事项

1. **证书一致性**
   - 始终使用统一证书脚本生成证书
   - 不要手动修改证书文件
   - 确保所有组件使用相同的CA证书

2. **服务器地址变更**
   - 修改SERVER_IP环境变量
   - 重新生成证书
   - 更新所有配置文件

3. **跨平台部署**
   - 脚本支持macOS和Linux
   - 自动适配不同系统的sed命令
   - 证书格式兼容StrongSwan
