package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vpn-location/backend/model"
)

const sessionCookieName = "vpn_session"
const sessionDuration = 7 * 24 * time.Hour

// sessionStore 内存 session 存储: token -> username
var sessionStore = make(map[string]string)

// Register 注册新用户
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=32"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "用户名至少3位,密码至少6位"})
		return
	}
	// 只允许字母数字下划线
	for _, ch := range req.Username {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			c.JSON(400, gin.H{"error": "用户名只能包含字母数字和下划线"})
			return
		}
	}

	user, err := model.GetStore().CreateUser(req.Username, req.Password)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 注册后自动写入 ipsec.secrets
	_ = AddEAPCredential(user.VPNUser, user.VPNPass)

	// 自动登录
	token := createSession(user.Username)
	setSessionCookie(c, token)

	c.JSON(200, gin.H{
		"message":  "注册成功",
		"username": user.Username,
		"vpn_user": user.VPNUser,
	})
}

// Login 登录
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请填写用户名和密码"})
		return
	}

	user, err := model.GetStore().Authenticate(req.Username, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	token := createSession(user.Username)
	setSessionCookie(c, token)

	c.JSON(200, gin.H{
		"message":  "登录成功",
		"username": user.Username,
		"vpn_user": user.VPNUser,
	})
}

// Logout 登出
func Logout(c *gin.Context) {
	token, _ := c.Cookie(sessionCookieName)
	if token != "" {
		delete(sessionStore, token)
	}
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.JSON(200, gin.H{"message": "已登出"})
}

// GetProfile 获取当前用户信息
func GetProfile(c *gin.Context) {
	username := getUsername(c)
	if username == "" {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}
	user, ok := model.GetStore().GetByUsername(username)
	if !ok {
		c.JSON(401, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(200, gin.H{
		"username": user.Username,
		"vpn_user": user.VPNUser,
	})
}

// AuthRequired 认证中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := getUsername(c)
		if username == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "未登录"})
			return
		}
		c.Set("username", username)
		c.Next()
	}
}

// AuthOptional 可选认证中间件：如果用户已登录则设置username，未登录则继续
func AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := getUsername(c)
		if username != "" {
			c.Set("username", username)
		}
		c.Next()
	}
}

// --- 内部工具函数 ---

func createSession(username string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)
	sessionStore[token] = username
	return token
}

func setSessionCookie(c *gin.Context, token string) {
	c.SetCookie(sessionCookieName, token, int(sessionDuration.Seconds()), "/", "", false, true)
}

func getUsername(c *gin.Context) string {
	// 先从 gin context 取
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	// 再从 cookie 解析
	token, err := c.Cookie(sessionCookieName)
	if err != nil || token == "" {
		return ""
	}
	username, ok := sessionStore[token]
	if !ok {
		return ""
	}
	return username
}

// InitAuthFromIPSec 启动时从 ipsec.secrets 同步已有 EAP 用户到 session store
func InitAuthFromIPSec() {
	// 无需操作,eap.go 已有的用户通过 EAP 认证
	// 新注册的用户由 AddEAPCredential 写入 secrets
}
