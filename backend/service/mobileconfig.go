package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// GenerateMobileConfig 生成iOS IKEv2 VPN描述文件
// serverAddr: 服务器IP，必须和 ipsec.conf leftid 及服务器证书 CN 完全一致
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

	// Split Tunnel 模式：
	// - DNS 推给服务器（服务器劫持 gs-loc.apple.com → 服务器自身 IP）
	// - 只有去往服务器 IP 的流量走 VPN 隧道
	// - 手机其他流量（高德地图、普通上网）走自己的网络，不受影响
	config := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>IKEv2</key>
			<dict>
				<key>AuthenticationMethod</key>
				<string>Certificate</string>
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
				<string>VPN Root CA</string>
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
				<integer>1</integer>
			</dict>
			<key>PayloadDescription</key>
			<string>wzgo.cc 网络加速</string>
			<key>PayloadDisplayName</key>
			<string>wzgo.cc 虚拟定位</string>
			<key>PayloadIdentifier</key>
			<string>cc.wzgo.vpn.managed.%s</string>
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
			<string>wzgo-ca.cer</string>
			<key>PayloadContent</key>
			<data>%s</data>
			<key>PayloadDescription</key>
			<string>wzgo.cc 根证书</string>
			<key>PayloadDisplayName</key>
			<string>wzgo.cc 根证书</string>
			<key>PayloadIdentifier</key>
			<string>cc.wzgo.certificate.root.%s</string>
			<key>PayloadType</key>
			<string>com.apple.security.root</string>
			<key>PayloadUUID</key>
			<string>%s</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
	</array>
	<key>PayloadDescription</key>
	<string>wzgo.cc 虚拟定位配置</string>
	<key>PayloadDisplayName</key>
	<string>wzgo.cc 虚拟定位</string>
	<key>PayloadIdentifier</key>
	<string>cc.wzgo.profile.%s</string>
	<key>PayloadOrganization</key>
	<string>wzgo.cc</string>
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

	return config, nil
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
