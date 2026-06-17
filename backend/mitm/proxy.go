package mitm

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/elazarl/goproxy"
	"github.com/vpn-location/backend/mitm/pb"
	proto "google.golang.org/protobuf/proto"
)

var (
	spoofLat float64 = 39.90923
	spoofLon float64 = 116.397428
	spoofMu  sync.RWMutex

	globalProxy *LocationProxy
	proxyOnce   sync.Once
)

type LocationProxy struct {
	port   int
	caCert *tls.Certificate
}

func NewLocationProxy(port int) *LocationProxy {
	return &LocationProxy{port: port}
}

func GetLocationProxy(port int) *LocationProxy {
	proxyOnce.Do(func() {
		globalProxy = &LocationProxy{port: port}
	})
	return globalProxy
}

func (p *LocationProxy) SetSpoofLocation(lat, lon float64) {
	spoofMu.Lock()
	defer spoofMu.Unlock()
	spoofLat = lat
	spoofLon = lon
	log.Printf("[MITM] 虚拟位置: %.6f, %.6f", lat, lon)
}

func (p *LocationProxy) GetSpoofLocation() (float64, float64) {
	spoofMu.RLock()
	defer spoofMu.RUnlock()
	return spoofLat, spoofLon
}

// LoadCA 从文件加载 CA 证书用于 MITM TLS 解密
func (p *LocationProxy) LoadCA(certPath, keyPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("读取CA证书失败: %v", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("读取CA私钥失败: %v", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("解析CA密钥对失败: %v", err)
	}
	if cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0]); err != nil {
		return fmt.Errorf("解析CA叶证书失败: %v", err)
	}

	p.caCert = &cert
	log.Printf("[MITM] CA 证书加载成功: %s", cert.Leaf.Subject.CommonName)
	return nil
}

// Start 启动 HTTP CONNECT 代理（Full Tunnel + PAC 模式）
// PAC 只把 gs-loc.apple.com 路由到这里，其他域名返回 DIRECT
func (p *LocationProxy) Start() error {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	if p.caCert != nil {
		mitmAction := &goproxy.ConnectAction{
			Action:    goproxy.ConnectMitm,
			TLSConfig: goproxy.TLSConfigFromCA(p.caCert),
		}
		proxy.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(
			func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				hostname := strings.Split(host, ":")[0]
				if hostname == "gs-loc.apple.com" || hostname == "gs-loc-cn.apple.com" {
					log.Printf("[MITM] 拦截 HTTPS 连接: %s", host)
					return mitmAction, host
				}
				return goproxy.OkConnect, host
			},
		))
	}

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		host := strings.Split(req.Host, ":")[0]
		if (host == "gs-loc.apple.com" || host == "gs-loc-cn.apple.com") &&
			req.URL.Path == "/clls/wloc" && req.Method == "POST" {
			return p.handleLocationRequest(req)
		}
		return req, nil
	})

	addr := fmt.Sprintf(":%d", p.port)
	log.Printf("[MITM] HTTPS MITM 代理启动 %s (CA已加载: %v)", addr, p.caCert != nil)
	return http.ListenAndServe(addr, proxy)
}

func (p *LocationProxy) handleLocationRequest(req *http.Request) (*http.Request, *http.Response) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MITM] PANIC in handleLocationRequest: %v", r)
		}
	}()

	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		log.Printf("[MITM] 读取请求体失败: %v", err)
		return req, nil
	}

	log.Printf("[MITM] 收到 /clls/wloc 请求 %d bytes", len(body))

	arpc := ArpcDeserialize(body)
	if arpc == nil {
		log.Printf("[MITM] ARPC 解析失败")
		return req, nil
	}

	var wloc pb.AppleWLoc
	if err := proto.Unmarshal(arpc.Payload, &wloc); err != nil {
		log.Printf("[MITM] Protobuf 解析失败: %v", err)
		return req, nil
	}

	spoofMu.RLock()
	lat, lon := spoofLat, spoofLon
	spoofMu.RUnlock()

	iLat := IntFromCoord(lat)
	iLon := IntFromCoord(lon)
	ha := int64(39)
	va := int64(1000)
	alt := int64(530)
	uv4 := int64(3)
	mat := int64(63)
	mac := int64(467)

	log.Printf("[MITM] 替换 %d WiFi + %d CellTower 位置 -> (%.6f, %.6f)",
		len(wloc.WifiDevices), len(wloc.CellTowerResponse), lat, lon)

	// 替换 WiFi 设备位置
	for _, device := range wloc.WifiDevices {
		if device.Location == nil {
			device.Location = &pb.Location{}
		}
		device.Location.Latitude = &iLat
		device.Location.Longitude = &iLon
		device.Location.HorizontalAccuracy = &ha
		device.Location.VerticalAccuracy = &va
		device.Location.Altitude = &alt
		device.Location.UnknownValue4 = &uv4
		device.Location.MotionActivityType = &mat
		device.Location.MotionActivityConfidence = &mac
	}

	// 替换基站位置（LTE 区域定位主要靠基站，必须一并替换）
	for _, tower := range wloc.CellTowerResponse {
		if tower.Location == nil {
			tower.Location = &pb.Location{}
		}
		tower.Location.Latitude = &iLat
		tower.Location.Longitude = &iLon
		tower.Location.HorizontalAccuracy = &ha
		tower.Location.VerticalAccuracy = &va
		tower.Location.Altitude = &alt
	}

	wloc.NumCellResults = nil
	wloc.NumWifiResults = nil
	wloc.DeviceType = nil

	initialBytes, _ := hex.DecodeString("0001000000010000")
	respPayload, err := proto.Marshal(&wloc)
	if err != nil {
		log.Printf("[MITM] 序列化失败: %v", err)
		return req, nil
	}
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(respPayload)))

	// 用预分配避免 append 嵌套导致的 slice 底层数组复用问题
	responseBytes := make([]byte, 0, len(initialBytes)+2+len(respPayload))
	responseBytes = append(responseBytes, initialBytes...)
	responseBytes = append(responseBytes, lenBuf...)
	responseBytes = append(responseBytes, respPayload...)

	resp := &http.Response{
		Request:       req,
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(responseBytes)),
		ContentLength: int64(len(responseBytes)),
	}
	resp.Header.Set("Content-Type", "application/octet-stream")
	log.Printf("[MITM] 返回伪造响应 %d bytes", len(responseBytes))
	return req, resp
}

// GetCACertPEM 返回 CA 证书 PEM（供前端下载信任）
func (p *LocationProxy) GetCACertPEM() []byte {
	if p.caCert == nil {
		return nil
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.caCert.Certificate[0],
	})
}
