package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/vpn-location/backend/api"
	"github.com/vpn-location/backend/config"
	"github.com/vpn-location/backend/service"
)

// distPath 返回前端 dist 目录的绝对路径
// 优先用环境变量 FRONTEND_DIST，其次按常见部署结构逐级查找
func distPath() string {
	if p := os.Getenv("FRONTEND_DIST"); p != "" {
		return p
	}
	wd, _ := os.Getwd()
	// 按优先级尝试多个候选路径
	candidates := []string{
		filepath.Join(wd, "frontend", "dist"),          // 项目根/frontend/dist
		filepath.Join(wd, "..", "frontend", "dist"),    // backend/../frontend/dist
		filepath.Join(wd, "frontend"),                   // 项目根/frontend (dist内容直接散落)
		filepath.Join(wd, "..", "frontend"),             // backend/../frontend
		"/opt/vpn-location/frontend/dist",               // 服务器部署路径
		"/opt/vpn-location/frontend",                    // 服务器部署路径(dist内容)
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
			return dir
		}
	}
	return filepath.Join(wd, "frontend", "dist")
}

func SetupRoutes(r *gin.Engine) {
	// CORS中间件
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
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

	// 前端配置接口
	r.GET("/api/config", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"serverAddress": config.Config.VPN.ServerAddress,
		})
	})
	// proxy.pac：只代理 Apple 定位服务到 8081，其余流量直连
	// 添加常用定位服务域名排除，确保第三方应用定位正常
	r.GET("/proxy.pac", func(c *gin.Context) {
		serverAddr := config.Config.VPN.ServerAddress
		pac := fmt.Sprintf(`function FindProxyForURL(url, host) {
    // Apple定位服务走MITM代理（支持iOS 17~27）
    if (host == "gs-loc.apple.com" || 
        host == "gs-loc-cn.apple.com" || 
        host == "location.apple.com" || 
        host == "location-cn.apple.com" ||
        host.includes("gs-loc") ||
        host.includes("location") ||
        host.includes(".ls.apple.com") ||
        host.includes("gsp-ssl.ls.apple.com") ||
        host.includes("ls-cn.apple.com")) {
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

	// 逆地理编码（使用高德地图API）
	r.GET("/reverse", func(c *gin.Context) {
		lat := c.Query("lat")
		lon := c.Query("lon")
		if lat == "" || lon == "" {
			c.JSON(400, gin.H{"error": "缺少经纬度参数"})
			return
		}
		// 使用高德地图逆地理编码API
	url := fmt.Sprintf("https://restapi.amap.com/v3/geocode/regeo?output=json&location=%s,%s&key=9609abf363a3f7c1e377ef7cf04cde8b8aa0&radius=1000", lon, lat)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "VPNLocationApp/1.0")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(500, gin.H{"error": "逆地理编码服务不可用"})
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			c.JSON(500, gin.H{"error": "解析响应失败"})
			return
		}

		// 提取地址信息
		if status, ok := result["status"].(string); ok && status == "1" {
			if regeocode, ok := result["regeocode"].(map[string]interface{}); ok {
				if addressComponent, ok := regeocode["addressComponent"].(map[string]interface{}); ok {
					var parts []string
					if district, ok := addressComponent["district"].(string); ok && district != "" {
						parts = append(parts, district)
					}
					if city, ok := addressComponent["city"].(string); ok && city != "" {
						parts = append(parts, city)
					}
					if province, ok := addressComponent["province"].(string); ok && province != "" {
						parts = append(parts, province)
					}
					if formattedAddress, ok := regeocode["formatted_address"].(string); ok && formattedAddress != "" {
						c.JSON(200, gin.H{
							"display_name": formattedAddress,
							"parts":        parts,
						})
						return
					}
				}
			}
		}

		// 如果高德API失败，使用OSM Nominatim作为备选
		osmUrl := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%s&lon=%s&zoom=18", lat, lon)
		osmReq, _ := http.NewRequest("GET", osmUrl, nil)
		osmReq.Header.Set("User-Agent", "VPNLocationApp/1.0")
		osmResp, err := http.DefaultClient.Do(osmReq)
		if err != nil {
			c.JSON(500, gin.H{"error": "逆地理编码服务不可用"})
			return
		}
		defer osmResp.Body.Close()

		var osmResult map[string]interface{}
		if err := json.NewDecoder(osmResp.Body).Decode(&osmResult); err != nil {
			c.JSON(500, gin.H{"error": "解析响应失败"})
			return
		}

		if displayName, ok := osmResult["display_name"].(string); ok {
			c.JSON(200, gin.H{
				"display_name": displayName,
				"parts":        []string{},
			})
			return
		}

		c.JSON(404, gin.H{"error": "未找到地址信息"})
	})
	// 认证路由（公开）
	r.POST("/api/register", service.Register)
	r.POST("/api/login", service.Login)
	r.POST("/api/logout", service.Logout)
	r.GET("/api/profile", service.GetProfile)
	r.GET("/api/check", api.CheckVPNConnection)
	r.GET("/api/map/status", api.GetMapStatus)
	r.POST("/api/map/spoof", api.SetSpoofLocation)

	// 需要登录的 API 路由
	authGroup := r.Group("/api")
	authGroup.Use(service.AuthRequired())
	{
		authGroup.GET("/vpn/download", api.DownloadVPNConfig)
		authGroup.POST("/vpn/download", api.DownloadVPNConfig)
		authGroup.POST("/map/lock", api.LockLocation)
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
