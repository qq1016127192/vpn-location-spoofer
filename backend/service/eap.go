package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/vpn-location/backend/config"
)

var eapMu sync.Mutex

// AddEAPCredential 把用户 EAP 凭据追加到 ipsec.secrets 并热重载
func AddEAPCredential(username, password string) error {
	eapMu.Lock()
	defer eapMu.Unlock()

	secretsPath := config.Config.VPN.IpsecSecrets
	if secretsPath == "" {
		secretsPath = "/etc/ipsec.secrets" // 默认 Linux 路径
	}

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		return fmt.Errorf("读取 ipsec.secrets 失败: %v", err)
	}

	line := fmt.Sprintf("%s : EAP \"%s\"", username, password)

	// 已存在则更新，否则追加
	if strings.Contains(string(data), username+" : EAP") {
		lines := strings.Split(string(data), "\n")
		var newLines []string
		for _, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), username+" : EAP") {
				continue
			}
			newLines = append(newLines, l)
		}
		data = []byte(strings.Join(newLines, "\n"))
	}

	content := strings.TrimRight(string(data), "\n") + "\n" + line + "\n"
	if err := os.WriteFile(secretsPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("写入 ipsec.secrets 失败: %v", err)
	}

	// 热重载：用配置里的 ipsec 路径
	ipsecBin := config.Config.VPN.IpsecBin
	if ipsecBin == "" {
		ipsecBin = "/usr/sbin/ipsec"
	}
	if out, err := exec.Command("sudo", "-n", ipsecBin, "rereadsecrets").CombinedOutput(); err != nil {
		return fmt.Errorf("重载 secrets 失败: %v, output: %s", err, out)
	}

	return nil
}
