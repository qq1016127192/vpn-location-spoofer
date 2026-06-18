package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	mathrand "math/rand"
	"os"
	"os/exec"
	"strings"
)

// GenerateMobileConfig 生成iOS IKEv2 VPN描述文件（压缩混淆 + PKCS#7 签名）
// serverAddr: 服务器地址（IP或域名）
// username/password: EAP-MSCHAPv2 认证凭据
// caCertPEM: Root CA 证书（PEM格式）
func GenerateMobileConfig(serverAddr, username, password string, caCertPEM []byte) (string, error) {
	block, _ := pem.Decode(caCertPEM)
	if block == nil {
		return "", fmt.Errorf("解析CA证书失败，请确认传入的是PEM格式证书")
	}
	caCertB64 := base64.StdEncoding.EncodeToString(block.Bytes)

	vpnUUID := generateUUID()
	certUUID := generateUUID()
	profileUUID := generateUUID()

	// 生成原始配置（压缩版，无多余空格换行）
	configMin := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
<key>PayloadContent</key>
<array>
<dict>
<key>IKEv2</key>
<dict>
<key>AuthenticationMethod</key>
<string>None</string>
<key>ChildSecurityAssociationParameters</key>
<dict>
<key>EncryptionAlgorithm</key>
<string>AES-256</string>
<key>IntegrityAlgorithm</key>
<string>SHA2-256</string>
<key>DiffieHellmanGroup</key>
<integer>14</integer>
<key>LifeTimeInMinutes</key>
<integer>1440</integer>
</dict>
<key>DeadPeerDetectionRate</key>
<string>Medium</string>
<key>DisableMOBIKE</key>
<false/>
<key>DisableRedirect</key>
<false/>
<key>EnableCertificateRevocationCheck</key>
<false/>
<key>EnablePFS</key>
<true/>
<key>ExtendedAuthEnabled</key>
<true/>
<key>IKESecurityAssociationParameters</key>
<dict>
<key>EncryptionAlgorithm</key>
<string>AES-256</string>
<key>IntegrityAlgorithm</key>
<string>SHA2-256</string>
<key>DiffieHellmanGroup</key>
<integer>14</integer>
<key>LifeTimeInMinutes</key>
<integer>1440</integer>
</dict>
<key>RemoteAddress</key>
<string>%s</string>
<key>RemoteIdentifier</key>
<string>%s</string>
<key>ServerCertificateIssuerCommonName</key>
<string>wzgo.cc Virtual Location Root CA</string>
<key>AuthName</key>
<string>%s</string>
<key>AuthPassword</key>
<string>%s</string>
<key>UseConfigurationAttributeInternalIPSubnet</key>
<false/>
</dict>
<key>IPv4</key>
<dict>
<key>OverridePrimary</key>
<integer>0</integer>
</dict>
<key>PayloadDescription</key>
<string>wzgo.cc 虚拟定位</string>
<key>PayloadDisplayName</key>
<string>wzgo.cc 虚拟定位</string>
<key>PayloadIdentifier</key>
<string>com.vpn.managed.%s</string>
<key>PayloadOrganization</key>
<string>请勿用于违法犯罪</string>
<key>PayloadType</key>
<string>com.apple.vpn.managed</string>
<key>PayloadUUID</key>
<string>%s</string>
<key>PayloadVersion</key>
<integer>1</integer>
<key>Proxies</key>
<dict>
<key>ProxyAutoConfigEnable</key>
<integer>1</integer>
<key>ProxyAutoConfigURLString</key>
<string>http://%s:8080/proxy.pac</string>
</dict>
<key>UserDefinedName</key>
<string>wzgo.cc 虚拟定位</string>
<key>VPNType</key>
<string>IKEv2</string>
</dict>
<dict>
<key>PayloadCertificateFileName</key>
<string>root-ca.cer</string>
<key>PayloadContent</key>
<data>%s</data>
<key>PayloadDescription</key>
<string>wzgo.cc Virtual Location Root CA</string>
<key>PayloadDisplayName</key>
<string>wzgo.cc Virtual Location Root CA</string>
<key>PayloadIdentifier</key>
<string>com.vpn.certificate.root.%s</string>
<key>PayloadOrganization</key>
<string>请勿用于违法犯罪</string>
<key>PayloadType</key>
<string>com.apple.security.root</string>
<key>PayloadUUID</key>
<string>%s</string>
<key>PayloadVersion</key>
<integer>1</integer>
</dict>
</array>
<key>PayloadDescription</key>
<string>wzgo.cc 虚拟定位</string>
<key>PayloadDisplayName</key>
<string>wzgo.cc 虚拟定位</string>
<key>PayloadIdentifier</key>
<string>com.vpn.profile.%s</string>
<key>PayloadOrganization</key>
<string>请勿用于违法犯罪</string>
<key>PayloadRemovalDisallowed</key>
<false/>
<key>PayloadType</key>
<string>Configuration</string>
<key>PayloadUUID</key>
<string>%s</string>
<key>PayloadVersion</key>
<integer>1</integer>
</dict>
</plist>`,
		serverAddr,  // RemoteAddress
		serverAddr,  // RemoteIdentifier
		username,    // AuthName
		password,    // AuthPassword
		vpnUUID,     // VPN PayloadIdentifier suffix
		vpnUUID,     // VPN PayloadUUID
		serverAddr,  // PAC URL host
		caCertB64,   // CA cert DER base64
		certUUID,    // cert PayloadIdentifier suffix
		certUUID,    // cert PayloadUUID
		profileUUID, // profile PayloadIdentifier suffix
		profileUUID, // profile PayloadUUID
	)

	// 进一步压缩：移除多余空格和换行
	configMin = strings.ReplaceAll(configMin, "\t", "")
	configMin = strings.ReplaceAll(configMin, "\n\n", "\n")

	// XML 混淆：注入注释、Junk键值对、替换显示名称
	configMin = obfuscatePlist(configMin)

	// 尝试 PKCS#7 签名（用 openssl 命令，基于服务器上已有证书）
	// 如果签名失败，就直接返回压缩版
	signed, err := trySignMobileConfig(configMin)
	if err == nil && signed != "" {
		return signed, nil
	}

	return configMin, nil
}

// trySignMobileConfig 尝试用 openssl 对 mobileconfig 做 PKCS#7 签名
func trySignMobileConfig(xml string) (string, error) {
	// 1. 写临时文件
	tmpXml, err := os.CreateTemp("", "mobileconfig_*.xml")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpXml.Name())
	_, _ = tmpXml.WriteString(xml)
	_ = tmpXml.Close()

	// 2. 签名（先检查本地有证书，再检查服务器证书路径）
	certPath := "/opt/vpn-location/certs/serverCert.pem"
	keyPath := "/opt/vpn-location/certs/serverKey.pem"
	caCertPath := "/opt/vpn-location/certs/caCert.pem"
	// 本地开发环境路径
	if _, err := os.Stat("certs/serverCert.pem"); err == nil {
		certPath = "certs/serverCert.pem"
		keyPath = "certs/serverKey.pem"
		caCertPath = "certs/caCert.pem"
	}

	tmpSigned, err := os.CreateTemp("", "mobileconfig_signed_*.mobileconfig")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpSigned.Name())
	_ = tmpSigned.Close()

	cmd := exec.Command("openssl", "smime", "-sign",
		"-in", tmpXml.Name(),
		"-out", tmpSigned.Name(),
		"-signer", certPath,
		"-inkey", keyPath,
		"-certfile", caCertPath,
		"-outform", "DER",
		"-nodetach")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("openssl 签名失败: %v, %s", err, string(output))
	}

	signedData, err := os.ReadFile(tmpSigned.Name())
	if err != nil {
		return "", err
	}

	return string(signedData), nil
}

// generateUUID 生成标准 UUID v4
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// obfuscatePlist 对 plist只做安全的混淆
func obfuscatePlist(xml string) string {
	// 只注入XML注释，不会破坏XML结构
	xml = injectXMLComments(xml)
	return xml
}

// replaceDisplayNames 替换所有暴露用途的显示名称
func replaceDisplayNames(xml string) string {
	// 不再进行随机替换，因为所有显示名称已设置为用户自定义值
	// 这样可以确保iOS能正确识别和显示配置信息
	return xml
}

// injectXMLComments 在 plist 中注入随机 XML 注释，只在安全位置注入
func injectXMLComments(xml string) string {
	// 只在 <plist version="1.0"> 之后注入注释（安全位置）
	xml = strings.Replace(xml, `<plist version="1.0">`,
		`<plist version="1.0">`+"\n"+"<!-- "+randChars(40)+" -->", 1)

	// 只在 </plist> 之前注入注释（安全位置）
	xml = strings.Replace(xml, "</plist>",
		"<!-- "+randChars(40)+" -->\n</plist>", 1)

	return xml
}

// randChars 生成 n 个随机字符（字母数字 + 混淆字符）
func randChars(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = chars[mathrand.Intn(len(chars))]
	}
	return string(result)
}