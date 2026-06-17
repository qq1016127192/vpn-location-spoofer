package nat

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

// Setup 根据当前操作系统配置 IP 转发和 NAT，使 VPN 客户端流量能正常出网
func Setup(subnet string, logger *zap.Logger) {
	if os.Getuid() != 0 {
		logger.Warn("[NAT] 非 root 权限，跳过 NAT 配置（需要 sudo 运行）")
		return
	}

	switch runtime.GOOS {
	case "darwin":
		setupMacOS(subnet, logger)
	case "linux":
		setupLinux(subnet, logger)
	default:
		logger.Warn("[NAT] 不支持的系统，跳过 NAT 配置", zap.String("os", runtime.GOOS))
	}
}

// setupMacOS 配置 macOS pf NAT
func setupMacOS(subnet string, logger *zap.Logger) {
	// 开启 IP 转发
	if err := run("sysctl", "-w", "net.inet.ip.forwarding=1"); err != nil {
		logger.Warn("[NAT] 开启 IP 转发失败", zap.Error(err))
	} else {
		logger.Info("[NAT] macOS IP 转发已开启")
	}

	// 找出口网卡（有默认路由的那个）
	outIf, err := getMacOutInterface()
	if err != nil {
		logger.Warn("[NAT] 找不到出口网卡", zap.Error(err))
		return
	}
	logger.Info("[NAT] 出口网卡", zap.String("interface", outIf))

	// 写 pf anchor 规则文件
	anchorFile := "/tmp/vpn-nat.conf"
	rule := fmt.Sprintf("nat on %s from %s to any -> (%s)\n", outIf, subnet, outIf)
	if err := os.WriteFile(anchorFile, []byte(rule), 0644); err != nil {
		logger.Warn("[NAT] 写规则文件失败", zap.Error(err))
		return
	}

	// 确保 pf 已启用
	_ = run("pfctl", "-e")

	// 加载 anchor
	if err := run("pfctl", "-a", "vpn-nat", "-f", anchorFile); err != nil {
		logger.Warn("[NAT] 加载 pf 规则失败", zap.Error(err))
		return
	}
	logger.Info("[NAT] macOS pf NAT 规则已加载", zap.String("rule", strings.TrimSpace(rule)))
}

// setupLinux 配置 Linux iptables NAT
func setupLinux(subnet string, logger *zap.Logger) {
	// 开启 IP 转发
	if err := run("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		logger.Warn("[NAT] 开启 IP 转发失败", zap.Error(err))
	} else {
		logger.Info("[NAT] Linux IP 转发已开启")
	}

	// 找出口网卡
	outIf, err := getLinuxOutInterface()
	if err != nil {
		logger.Warn("[NAT] 找不到出口网卡", zap.Error(err))
		return
	}
	logger.Info("[NAT] 出口网卡", zap.String("interface", outIf))

	// 先检查规则是否已存在，避免重复添加
	checkErr := run("iptables", "-t", "nat", "-C", "POSTROUTING",
		"-s", subnet, "-o", outIf, "-j", "MASQUERADE")
	if checkErr != nil {
		// 规则不存在，添加
		if err := run("iptables", "-t", "nat", "-A", "POSTROUTING",
			"-s", subnet, "-o", outIf, "-j", "MASQUERADE"); err != nil {
			logger.Warn("[NAT] 添加 iptables 规则失败", zap.Error(err))
			return
		}
	}
	logger.Info("[NAT] Linux iptables NAT 规则已就绪",
		zap.String("subnet", subnet), zap.String("outIf", outIf))
}

func getMacOutInterface() (string, error) {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("未找到默认路由接口")
}

func getLinuxOutInterface() (string, error) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "", err
	}
	// 格式: default via x.x.x.x dev eth0 ...
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", fmt.Errorf("未找到默认路由接口")
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// SetupDNS 配置 dnsmasq 劫持 gs-loc.apple.com 到服务器自身 IP
// Split Tunnel 模式下，iOS DNS 指向服务器，服务器把定位域名解析到自身
func SetupDNS(serverIP string, logger *zap.Logger) {
	// 检查 dnsmasq 是否安装
	if _, err := exec.LookPath("dnsmasq"); err != nil {
		logger.Warn("[DNS] dnsmasq 未安装，跳过 DNS 劫持配置",
			zap.String("hint", "brew install dnsmasq (macOS) 或 apt install dnsmasq (Linux)"))
		return
	}

	confDir := dnsmasqConfDir()
	confFile := confDir + "/vpn-location-spoof.conf"

	content := fmt.Sprintf("# 劫持 Apple 定位服务到本机，由 vpn-server 自动生成\n"+
		"address=/gs-loc.apple.com/%s\n"+
		"address=/gs-loc-cn.apple.com/%s\n", serverIP, serverIP)

	if err := os.MkdirAll(confDir, 0755); err != nil {
		logger.Warn("[DNS] 创建 dnsmasq 配置目录失败", zap.Error(err))
		return
	}
	if err := os.WriteFile(confFile, []byte(content), 0644); err != nil {
		logger.Warn("[DNS] 写入 dnsmasq 配置失败", zap.Error(err))
		return
	}
	logger.Info("[DNS] dnsmasq 劫持规则已写入", zap.String("file", confFile))

	// 重载 dnsmasq
	if err := run("pkill", "-HUP", "dnsmasq"); err != nil {
		// dnsmasq 可能未运行，尝试启动
		if runtime.GOOS == "darwin" {
			_ = run("brew", "services", "restart", "dnsmasq")
		} else {
			_ = run("systemctl", "restart", "dnsmasq")
		}
	}
	logger.Info("[DNS] dnsmasq 已重载", zap.String("serverIP", serverIP))
}

func dnsmasqConfDir() string {
	if runtime.GOOS == "darwin" {
		// Homebrew dnsmasq
		return "/opt/homebrew/etc/dnsmasq.d"
	}
	return "/etc/dnsmasq.d"
}
