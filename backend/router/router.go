package router

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/vpn-location/backend/api"
	"github.com/vpn-location/backend/config"
)

// distPath 返回前端 dist 目录的绝对路径
// 优先用环境变量 FRONTEND_DIST，其次相对可执行文件位置推断
func distPath() string {
	if p := os.Getenv("FRONTEND_DIST"); p != "" {
		return p
	}
	wd, _ := os.Getwd()
	// 尝试相对路径
	rel := filepath.Join(wd, "..", "frontend", "dist")
	if _, err := os.Stat(filepath.Join(rel, "index.html")); err == nil {
		return rel
	}
	// fallback 绝对路径
	return "/Users/yangyongxing/Desktop/jikeh/vpn-location-spoofer/frontend/dist"
}

func SetupRoutes(r *gin.Engine) {
	// CORS中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 前端配置接口：把需要的配置下发给前端
	r.GET("/api/config", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"amapKey":          config.Config.VPN.APIKey,
			"amapSecurityCode": config.Config.VPN.SecurityCode,
		})
	})
	// proxy.pac：只代理 Apple 定位服务到 8081，其余流量直连
	// 添加常用定位服务域名排除，确保第三方应用定位正常
	r.GET("/proxy.pac", func(c *gin.Context) {
		serverAddr := config.Config.VPN.ServerAddress
		pac := fmt.Sprintf(`function FindProxyForURL(url, host) {
    // Apple定位服务走MITM代理
    if (host == "gs-loc.apple.com" || host == "gs-loc-cn.apple.com") {
        return "PROXY %s:8081";
    }
    // 高德地图和定位服务直连
    if (host.includes("autonavi.com") || host.includes("amap.com") || 
        host.includes("webrd0") || host.includes("lbs.amap.com")) {
        return "DIRECT";
    }
    // 微信相关域名直连
    if (host.includes("weixin.qq.com") || host.includes("wx.qq.com") || 
        host.includes("api.weixin.qq.com") || host.includes("lbs.weixin.qq.com")) {
        return "DIRECT";
    }
    // 其他流量直连
    return "DIRECT";
}`, serverAddr)
		c.Header("Content-Type", "application/x-ns-proxy-autoconfig")
		c.String(200, pac)
	})

	// 地点搜索代理（Photon - 基于 OpenStreetMap，完全免费无需 Key）
	r.GET("/search", func(c *gin.Context) {
		q := c.Query("q")
		if q == "" {
			c.JSON(200, []interface{}{})
			return
		}
		url := fmt.Sprintf("https://photon.komoot.io/api/?q=%s&limit=6", q)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "VPNLocationApp/1.0")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// 搜索服务不可用时返回空结果，不影响地图使用
			c.JSON(200, []interface{}{})
			return
		}
		defer resp.Body.Close()
		c.DataFromReader(200, resp.ContentLength, "application/json", resp.Body, nil)
	})
	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/vpn/download", api.DownloadVPNConfig)
		apiGroup.POST("/vpn/download", api.DownloadVPNConfig)
		apiGroup.POST("/map/lock", api.LockLocation)
		apiGroup.POST("/map/spoof", api.SetSpoofLocation)
		apiGroup.GET("/check", api.CheckVPNConnection)
	}

	// 托管前端静态文件
	dist := distPath()
	r.Static("/assets", filepath.Join(dist, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(dist, "favicon.ico"))

	// Vue Router history 模式：所有非 API 路径返回 index.html
	indexHTML := filepath.Join(dist, "index.html")
	r.GET("/", func(c *gin.Context) {
		c.File(indexHTML)
	})
	r.NoRoute(func(c *gin.Context) {
		c.File(indexHTML)
	})
	_ = http.StatusOK
}
