package api

import (
	"github.com/gin-gonic/gin"
	"github.com/vpn-location/backend/config"
	"github.com/vpn-location/backend/mitm"
	"github.com/vpn-location/backend/model"
	"github.com/vpn-location/backend/service"
)

// DownloadVPNConfig 下载VPN配置文件（需登录，使用用户独立凭据）
func DownloadVPNConfig(c *gin.Context) {
	// 从 session 获取用户
	username := c.GetString("username")
	if username == "" {
		c.JSON(401, gin.H{"error": "请先登录"})
		return
	}

	user, ok := model.GetStore().GetByUsername(username)
	if !ok {
		c.JSON(401, gin.H{"error": "用户不存在"})
		return
	}

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

	// 使用用户独立的 VPN 凭据
	mobileconf, err := service.GenerateMobileConfig(serverAddress, user.VPNUser, user.VPNPass, certPEM)
	if err != nil {
		c.JSON(500, gin.H{"error": "生成配置文件失败"})
		return
	}

	// 确保 ipsec.secrets 中有此用户的 EAP 凭据
	_ = service.AddEAPCredential(user.VPNUser, user.VPNPass)

	c.Header("Content-Type", "application/x-apple-aspen-config")
	c.Header("Content-Disposition", "attachment; filename=profile.mobileconfig")
	c.String(200, mobileconf)
}

// LockLocation 锁定位置（按 VPN 客户端 IP 隔离，无需注册）
func LockLocation(c *gin.Context) {
	var req struct {
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	// 优先使用参数中的client_ip，其次使用VPN客户端IP
	clientIP := c.Query("client_ip")
	if clientIP == "" {
		clientIP = c.GetHeader("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = c.GetHeader("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = c.ClientIP()
	}
	clientIP = service.GetClientIP(clientIP)

	// 锁定位置
	locationService := service.GetLocationService()
	err := locationService.LockLocation(clientIP, req.Latitude, req.Longitude)
	if err != nil {
		c.JSON(500, gin.H{"error": "位置锁定失败"})
		return
	}

	// 同步到 MITM 代理，让锁定位置立即生效
	proxy := mitm.GetLocationProxy(8081)
	proxy.SetSpoofLocation(clientIP, req.Latitude, req.Longitude)

	c.JSON(200, gin.H{
		"message":   "位置锁定成功",
		"latitude":  req.Latitude,
		"longitude": req.Longitude,
	})
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

	// 优先使用用户身份获取VPN IP，确保多用户隔离
	vpnService := service.GetVPNCheckService()
	clientIP := ""

	// 尝试从session获取用户名
	username := c.GetString("username")
	if username != "" {
		// 使用用户名查找对应的VPN IP
		if vpnIP := vpnService.GetVPNClientIPByUsername(username); vpnIP != "" {
			clientIP = vpnIP
		}
	}

	// 如果没有通过用户身份获取到IP，使用传统IP映射方式
	if clientIP == "" {
		clientIP = c.Query("client_ip")
		if clientIP == "" {
			clientIP = c.GetHeader("X-Real-IP")
		}
		if clientIP == "" {
			clientIP = c.GetHeader("X-Forwarded-For")
		}
		if clientIP == "" {
			clientIP = c.ClientIP()
		}
		clientIP = service.GetClientIP(clientIP)

		// 如果获取到的是公网IP，尝试查找对应的VPN客户端IP
		if vpnIP := vpnService.GetVPNClientIP(clientIP); vpnIP != "" {
			clientIP = vpnIP
		}
	}

	proxy := mitm.GetLocationProxy(8081)
	proxy.SetSpoofLocation(clientIP, req.Latitude, req.Longitude)

	c.JSON(200, gin.H{
		"message":   "虚拟位置设置成功",
		"latitude":  req.Latitude,
		"longitude": req.Longitude,
		"client_ip": clientIP,
	})
}

// GetMapStatus 获取地图状态：是否已设置虚拟位置 + MITM 拦截统计
func GetMapStatus(c *gin.Context) {
	// 优先使用用户身份获取VPN IP，确保多用户隔离
	vpnService := service.GetVPNCheckService()
	clientIP := ""

	// 尝试从session获取用户名
	username := c.GetString("username")
	if username != "" {
		// 使用用户名查找对应的VPN IP
		if vpnIP := vpnService.GetVPNClientIPByUsername(username); vpnIP != "" {
			clientIP = vpnIP
		}
	}

	// 如果没有通过用户身份获取到IP，使用传统IP映射方式
	if clientIP == "" {
		clientIP = c.Query("client_ip")
		if clientIP == "" {
			clientIP = c.GetHeader("X-Real-IP")
		}
		if clientIP == "" {
			clientIP = c.GetHeader("X-Forwarded-For")
		}
		if clientIP == "" {
			clientIP = c.ClientIP()
		}
		clientIP = service.GetClientIP(clientIP)

		// 如果获取到的是公网IP，尝试查找对应的VPN客户端IP
		if vpnIP := vpnService.GetVPNClientIP(clientIP); vpnIP != "" {
			clientIP = vpnIP
		}
	}

	proxy := mitm.GetLocationProxy(8081)
	hit := proxy.GetHitStats(clientIP)

	c.JSON(200, gin.H{
		"spoofed":   proxy.IsSpoofed(clientIP),
		"hit_count": hit.Count,
		"last_hit":  hit.LastTime,
		"last_lat":  hit.LastLat,
		"last_lon":  hit.LastLon,
		"client_ip": clientIP,
	})
}

// CheckVPNConnection 检查VPN连接状态
func CheckVPNConnection(c *gin.Context) {
	// 优先使用参数中的client_ip，其次使用VPN客户端IP
	clientIP := c.Query("client_ip")
	if clientIP == "" {
		clientIP = c.GetHeader("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = c.GetHeader("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = c.ClientIP()
	}
	// 使用VPN检测服务检查连接
	vpnService := service.GetVPNCheckService()
	connected := vpnService.CheckConnection(clientIP)

	c.JSON(200, gin.H{
		"connected": connected,
		"message":   connectedMessage(connected),
		"client_ip": clientIP,
	})
}

// connectedMessage 根据连接状态返回提示文案
func connectedMessage(connected bool) string {
	if connected {
		return "VPN 已连接，可以设置虚拟定位"
	}
	return "VPN 未连接，请先在 iOS 设置中连接 VPN"
}
