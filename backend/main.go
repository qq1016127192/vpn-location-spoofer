package main

import (
	"github.com/gin-gonic/gin"
	"github.com/vpn-location/backend/config"
	"github.com/vpn-location/backend/mitm"
	"github.com/vpn-location/backend/model"
	"github.com/vpn-location/backend/nat"
	"github.com/vpn-location/backend/router"
	"go.uber.org/zap"
)

func main() {
	// 初始化配置
	config.Init()

	// 初始化数据库
	if err := model.InitDB(); err != nil {
		panic("数据库初始化失败: " + err.Error())
	}

	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 配置 IP 转发 + NAT，确保 VPN 客户端流量能正常出网（地图等）
	nat.Setup("10.10.0.0/24", logger)

	// 启动 MITM HTTP 代理（Full Tunnel + PAC 模式）
	// PAC 把 gs-loc.apple.com 路由到 :8081，其他流量直连
	// NAT 已在上面配置好，直连流量能从 en0 正常出网
	locationProxy := mitm.GetLocationProxy(8081)
	if err := locationProxy.LoadCA(
		config.Config.Cert.RootCAPath,
		config.Config.Cert.RootKeyPath,
	); err != nil {
		logger.Fatal("MITM CA 加载失败", zap.Error(err))
	}
	go func() {
		logger.Info("Starting MITM proxy on :8081")
		if err := locationProxy.Start(); err != nil {
			logger.Error("Failed to start MITM proxy", zap.Error(err))
		}
	}()

	// 设置Gin模式
	if config.Config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := gin.Default()
	router.SetupRoutes(r)

	// 启动服务器
	addr := ":" + config.Config.Server.Port
	logger.Info("Server starting", zap.String("addr", addr))
	r.Run(addr)
}
