# VPN虚拟定位修改系统

基于iOS VPN代理技术的虚拟定位解决方案，类似loc567.com

## 项目结构

```
vpn-location-spoofer/
├── frontend/          # Vue.js前端应用
├── backend/           # 后端API服务
├── vpn-config/        # VPN配置文件生成
└── docs/             # 项目文档
```

## 技术栈

### 前端
- Vue.js 3.x
- 高德地图API
- 坐标转换库
- TailwindCSS

### 后端
- Go/Gin框架
- VPN配置生成
- 证书管理
- API接口

### VPN技术
- iOS描述文件
- Root CA证书
- 网络代理
- GPS拦截

## 功能特性

- 用户注册登录
- VPN配置下载
- 地图选点定位
- 位置锁定修改
- 网络验证
- 设备检测

## 开发进度

- [x] 项目目录创建
- [ ] 前端项目初始化
- [ ] 后端项目初始化
- [ ] VPN配置生成
- [ ] 证书管理
- [ ] 地图集成
- [ ] API开发

## 使用说明

详细使用教程请参考 docs/ 目录下的文档。

## 注意事项

- 仅支持iOS设备
- 需要Wi-Fi连接
- 必须信任Root证书
- 仅供学习研究使用
