package main

import (
	"github.com/gin-gonic/gin"
	"github.com/vpn-location/backend/config"
	"github.com/vpn-location/backend/mitm"
	"github.com/vpn-location/backend/nat"
	"github.com/vpn-location/backend/router"
	"go.uber.org/zap"
)

func main() {
	config.Init()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 配置 IP 转发 + NAT
	nat.Setup("10.10.0.0/24", logger)

	// 启动 MITM 代理
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

	if config.Config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	router.SetupRoutes(r)

	addr := ":" + config.Config.Server.Port
	logger.Info("Server starting", zap.String("addr", addr))
	r.Run(addr)
}
