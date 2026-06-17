package service

import (
	"strings"
	"sync"
)

// VPNCheckService VPN连接检测服务
type VPNCheckService struct {
	mu           sync.RWMutex
	connectedIPs map[string]bool // IP -> connected status
	allowedIPs   map[string]bool // 允许的VPN IP段
}

var vpnCheckService *VPNCheckService
var vpnOnce sync.Once

// GetVPNCheckService 获取VPN检测服务单例
func GetVPNCheckService() *VPNCheckService {
	vpnOnce.Do(func() {
		vpnCheckService = &VPNCheckService{
			connectedIPs: make(map[string]bool),
			allowedIPs:   make(map[string]bool),
		}
		// 初始化允许的IP段（VPN服务器IP）
		vpnCheckService.initAllowedIPs()
	})
	return vpnCheckService
}

// initAllowedIPs 初始化允许的VPN IP段
func (s *VPNCheckService) initAllowedIPs() {
	// TODO: 从配置文件或数据库加载允许的VPN IP段
	// 这里添加一些示例IP段
	s.allowedIPs["10.0.0.0/8"] = true
	s.allowedIPs["172.16.0.0/12"] = true
	s.allowedIPs["192.168.0.0/16"] = true

	// 本地调试：添加本地回环地址
	s.allowedIPs["127.0.0.1"] = true
}

// CheckConnection 检查VPN连接状态
func (s *VPNCheckService) CheckConnection(clientIP string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查IP是否在允许的VPN IP段内
	for ipRange := range s.allowedIPs {
		if s.isIPInRange(clientIP, ipRange) {
			return true
		}
	}

	// 检查是否是已连接的VPN IP
	return s.connectedIPs[clientIP]
}

// isIPInRange 检查IP是否在指定范围内
func (s *VPNCheckService) isIPInRange(ip, ipRange string) bool {
	// 简化实现，实际应该使用net.ParseCIDR
	ipParts := strings.Split(ip, ".")
	if len(ipParts) != 4 {
		return false
	}

	// 检查私有IP段
	firstOctet := ipParts[0]
	switch firstOctet {
	case "10":
		return true
	case "172":
		secondOctet := ipParts[1]
		return secondOctet >= "16" && secondOctet <= "31"
	case "192":
		secondOctet := ipParts[1]
		return secondOctet == "168"
	}

	return false
}

// AddConnectedIP 添加已连接的VPN IP
func (s *VPNCheckService) AddConnectedIP(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connectedIPs[ip] = true
}

// RemoveConnectedIP 移除已连接的VPN IP
func (s *VPNCheckService) RemoveConnectedIP(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.connectedIPs, ip)
}

// GetClientIP 从请求中获取客户端IP
func GetClientIP(ipStr string) string {
	// 处理代理情况
	if strings.Contains(ipStr, ",") {
		ips := strings.Split(ipStr, ",")
		return strings.TrimSpace(ips[0])
	}
	return strings.TrimSpace(ipStr)
}
