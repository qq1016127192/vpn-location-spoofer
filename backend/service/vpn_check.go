package service

import (
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

// VPNCheckService VPN连接检测服务
type VPNCheckService struct {
	mu              sync.RWMutex
	connectedIPs    map[string]bool   // 已连接的 VPN IP（虚拟 IP）
	vpnSubnets      []string          // VPN 分配的网段
	serverPublicIP  string            // 服务器公网 IP（固定配置）
}

var vpnCheckService *VPNCheckService
var vpnOnce sync.Once

// GetVPNCheckService 获取VPN检测服务单例
func GetVPNCheckService() *VPNCheckService {
	vpnOnce.Do(func() {
		vpnCheckService = &VPNCheckService{
			connectedIPs:   make(map[string]bool),
			vpnSubnets:     make([]string, 0),
			serverPublicIP: "116.62.147.43", // 固定服务器公网IP
		}
		vpnCheckService.AddVPNSubnet("10.10.0.0/16")
	})
	return vpnCheckService
}

// AddVPNSubnet 添加 VPN 网段
func (s *VPNCheckService) AddVPNSubnet(cidr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vpnSubnets = append(s.vpnSubnets, cidr)
}

// ipInSubnet 检查IP是否在网段内，简单的前缀匹配
func ipInSubnet(ip, subnet string) bool {
	// 解析 subnet，如 10.10.0.0/16
	parts := strings.Split(subnet, "/")
	if len(parts) != 2 {
		return false
	}
	network := parts[0]
	prefixLen := parts[1]

	// 获取要比较的前缀部分
	switch prefixLen {
	case "8":
		return strings.HasPrefix(ip, strings.Split(network, ".")[0]+".")
	case "16":
		netOctets := strings.Split(network, ".")
		if len(netOctets) < 2 {
			return false
		}
		prefix := netOctets[0] + "." + netOctets[1] + "."
		return strings.HasPrefix(ip, prefix)
	case "24":
		netOctets := strings.Split(network, ".")
		if len(netOctets) < 3 {
			return false
		}
		prefix := netOctets[0] + "." + netOctets[1] + "." + netOctets[2] + "."
		return strings.HasPrefix(ip, prefix)
	}
	return false
}

// CheckConnection 检查 VPN 连接状态
// clientIP 为请求来源 IP
func (s *VPNCheckService) CheckConnection(clientIP string) bool {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return false
	}

	// 1. 检查是否在 VPN 网段内（10.10.0.x）
	s.mu.RLock()
	subnets := s.vpnSubnets
	s.mu.RUnlock()
	for _, subnet := range subnets {
		if ipInSubnet(clientIP, subnet) {
			return true
		}
	}

	// 2. 实时查询 ipsec statusall（ipsec 执行很快，约10ms）
	connPublicIPs, hasActive := s.queryIPSecConnections()

	// 3. 源IP是VPN客户端公网IP
	for _, ip := range connPublicIPs {
		if ip == clientIP {
			return true
		}
	}

	// 4. NAT回环检测：iOS客户端访问服务器自身IP时，
	//    源IP变成服务器公网IP，此时若有活跃连接则视为已连接
	s.mu.RLock()
	serverPublicIP := s.serverPublicIP
	s.mu.RUnlock()
	if hasActive && serverPublicIP != "" && clientIP == serverPublicIP {
		return true
	}

	return false
}

// queryIPSecConnections 查询 ipsec statusall 获取活跃连接信息
// 返回：客户端公网IP列表, 是否有活跃连接
func (s *VPNCheckService) queryIPSecConnections() ([]string, bool) {
	cmd := exec.Command("ipsec", "statusall")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, false
	}

	// ikev2-eap[3]: ESTABLISHED 3 minutes ago, 172.16.29.124[map.iosr.cn]...39.182.37.31[192.168.10.9]
	//                                                                              ↑客户端公网IP在这里
	// 匹配客户端公网IP（...IP[...] 格式）
	reClientIP := regexp.MustCompile(`\.\.\.(\d+\.\d+\.\d+\.\d+)\[`)

	var publicIPs []string

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if matches := reClientIP.FindStringSubmatch(line); len(matches) > 1 {
			publicIPs = append(publicIPs, matches[1])
		}
	}

	return publicIPs, len(publicIPs) > 0
}

// AddConnectedIP 添加已连接的 VPN IP
func (s *VPNCheckService) AddConnectedIP(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connectedIPs[ip] = true
}

// RemoveConnectedIP 移除已连接的 VPN IP
func (s *VPNCheckService) RemoveConnectedIP(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.connectedIPs, ip)
}

// GetClientIP 从请求中获取客户端 IP（处理代理链）
func GetClientIP(ipStr string) string {
	if strings.Contains(ipStr, ",") {
		ips := strings.Split(ipStr, ",")
		return strings.TrimSpace(ips[len(ips)-1])
	}
	return strings.TrimSpace(ipStr)
}
