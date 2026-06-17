package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// GenerateRootCA 生成Root CA证书
func GenerateRootCA() (certPEM, keyPEM []byte, err error) {
	// 生成RSA私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("生成私钥失败: %v", err)
	}

	// 设置证书模板
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"VPN Location Spoofer"},
			CommonName:   "VPN Location Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1年有效期
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// 自签名证书
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("创建证书失败: %v", err)
	}

	// 编码证书为PEM格式
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// 编码私钥为PEM格式
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

// SaveCertificate 保存证书到文件
func SaveCertificate(certPath, keyPath string, certPEM, keyPEM []byte) error {
	// 保存证书
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("保存证书失败: %v", err)
	}

	// 保存私钥
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("保存私钥失败: %v", err)
	}

	return nil
}

// LoadCertificate 加载证书
func LoadCertificate(certPath, keyPath string) (certPEM, keyPEM []byte, err error) {
	certPEM, err = os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("读取证书失败: %v", err)
	}

	keyPEM, err = os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("读取私钥失败: %v", err)
	}

	return certPEM, keyPEM, nil
}

// GenerateClientCert 生成客户端证书（PEM格式）
func GenerateClientCert(caCertPEM, caKeyPEM []byte, username string) (certPEM, keyPEM []byte, err error) {
	// 解析CA证书和私钥
	caCertBlock, _ := pem.Decode(caCertPEM)
	if caCertBlock == nil {
		return nil, nil, fmt.Errorf("解析CA证书失败")
	}

	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("解析CA证书内容失败: %v", err)
	}

	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil {
		return nil, nil, fmt.Errorf("解析CA私钥失败")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("解析CA私钥内容失败: %v", err)
	}

	// 生成客户端私钥
	clientPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("生成客户端私钥失败: %v", err)
	}

	// 设置客户端证书模板
	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return nil, nil, fmt.Errorf("生成序列号失败: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"VPN Location Spoofer"},
			CommonName:   username,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // 1年有效期
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		DNSNames:    []string{username},
		IPAddresses: nil,
	}

	// 使用CA签发客户端证书
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &clientPrivateKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("创建客户端证书失败: %v", err)
	}

	// 编码客户端证书为PEM格式
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// 编码客户端私钥为PEM格式
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(clientPrivateKey),
	})

	return certPEM, keyPEM, nil
}
