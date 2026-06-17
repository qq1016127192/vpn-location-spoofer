package mitm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

// signCertForHost 用加载的 CA 动态签发指定域名的 TLS 证书
func signCertForHost(host string, ca *tls.Certificate) (*tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		DNSNames:     []string{host},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	caX509, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, err
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caX509, &priv.PublicKey, ca.PrivateKey)
	if err != nil {
		return nil, err
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
	}
	cert.Leaf, _ = x509.ParseCertificate(der)
	return cert, nil
}
