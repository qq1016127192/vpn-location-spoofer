package api

import (
	"github.com/gin-gonic/gin"
	"github.com/vpn-location/backend/config"
	"github.com/vpn-location/backend/mitm"
	"github.com/vpn-location/backend/service"
)

// DownloadVPNConfig 下载VPN配置文件（无需注册，使用共享凭据）
func DownloadVPNConfig(c *gin.Context) {
	// 加载 CA 证书
	certPath := config.Config.Cert.RootCAPath
	keyPath := config.Config.Cert.RootKeyPath
	certPEM, _, err := service.LoadCertificate(certPath, keyPath)
	if err != nil {
		var keyPEM []byte
		certPEM, keyPEM, err = service.GenerateRootCA()
		if err != nil {
			c.JSON(500, gin.H{"error": "生成证书失败"})
			return
		}
		if saveErr := service.SaveCertificate(certPath, keyPath, certPEM, keyPEM); saveErr != nil {
			c.JSON(500, gin.H{"error": "保存证书失败"})
			return
		}
	}

	serverAddress := config.Config.VPN.ServerAddress

	// 使用共享账号密码认证
	mobileconf, err := service.GenerateMobileConfig(serverAddress, "vpnuser", "vpnpass2024", certPEM)
	if err != nil {
		c.JSON(500, gin.H{"error": "生成配置文件失败"})
		return
	}

	c.Header("Content-Type", "application/x-apple-aspen-config")
	c.Header("Content-Disposition", "attachment; filename=vpn.mobileconfig")
	c.String(200, mobileconf)
}

// LockLocation 锁定位置
func LockLocation(c *gin.Context) {
	var req struct {
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	// TODO: 从JWT或session获取用户ID
	userID := "user_123" // 临时硬编码

	// 锁定位置
	locationService := service.GetLocationService()
	err := locationService.LockLocation(userID, req.Latitude, req.Longitude)
	if err != nil {
		c.JSON(500, gin.H{"error": "位置锁定失败"})
		return
	}

	c.JSON(200, gin.H{"message": "位置锁定成功"})
}

// SetSpoofLocation 设置虚拟定位位置（按客户端 VPN IP 隔离）
func SetSpoofLocation(c *gin.Context) {
	var req struct {
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	// 取客户端 IP（VPN 连接后是 10.10.0.x）
	clientIP := c.ClientIP()

	proxy := mitm.GetLocationProxy(8081)
	proxy.SetSpoofLocation(clientIP, req.Latitude, req.Longitude)

	c.JSON(200, gin.H{
		"message":   "虚拟位置设置成功",
		"client_ip": clientIP,
		"latitude":  req.Latitude,
		"longitude": req.Longitude,
	})
}

// CheckVPNConnection 检查VPN连接状态
func CheckVPNConnection(c *gin.Context) {
	// 获取客户端IP
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = c.GetHeader("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = c.GetHeader("X-Real-IP")
	}

	// 使用VPN检测服务检查连接
	vpnService := service.GetVPNCheckService()
	clientIP = service.GetClientIP(clientIP)
	connected := vpnService.CheckConnection(clientIP)

	c.JSON(200, gin.H{"connected": connected})
}
