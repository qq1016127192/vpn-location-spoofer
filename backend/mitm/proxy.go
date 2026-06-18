package mitm

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/vpn-location/backend/mitm/pb"
	proto "google.golang.org/protobuf/proto"
)

// 默认位置（北京天安门）
const defaultLat = 39.90923
const defaultLon = 116.397428

// per-IP 位置表
var (
	locationMap   = make(map[string][2]float64) // clientIP -> [lat, lon]
	locationMapMu sync.RWMutex

	// 拦截统计：clientIP -> {次数, 上次时间}
	hitMap   = make(map[string]*HitStats)
	hitMapMu sync.RWMutex

	globalProxy *LocationProxy
	proxyOnce   sync.Once
)

// HitStats 拦截统计
type HitStats struct {
	Count     int64  `json:"count"`
	LastTime  int64  `json:"last_time"`  // unix seconds
	LastLat   float64 `json:"last_lat"`
	LastLon   float64 `json:"last_lon"`
}

// PersistentData 持久化数据结构
type PersistentData struct {
	Locations map[string][2]float64 `json:"locations"`
	Hits      map[string]*HitStats   `json:"hits"`
}



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

// SetSpoofLocation 为指定客户端 IP 设置虚拟位置
func (p *LocationProxy) SetSpoofLocation(clientIP string, lat, lon float64) {
	locationMapMu.Lock()
	defer locationMapMu.Unlock()
	locationMap[clientIP] = [2]float64{lat, lon}
	log.Printf("[MITM] 用户 %s 虚拟位置: %.6f, %.6f", clientIP, lat, lon)
}

// GetSpoofLocation 获取指定客户端 IP 的虚拟位置，未设置则返回默认
func (p *LocationProxy) GetSpoofLocation(clientIP string) (float64, float64) {
	locationMapMu.RLock()
	defer locationMapMu.RUnlock()
	if loc, ok := locationMap[clientIP]; ok {
		return loc[0], loc[1]
	}
	return defaultLat, defaultLon
}

// IsSpoofed 判断该 clientIP 是否已设置过虚拟位置
func (p *LocationProxy) IsSpoofed(clientIP string) bool {
	locationMapMu.RLock()
	defer locationMapMu.RUnlock()
	_, ok := locationMap[clientIP]
	return ok
}

// GetHitStats 获取指定客户端的拦截统计
func (p *LocationProxy) GetHitStats(clientIP string) *HitStats {
	hitMapMu.RLock()
	defer hitMapMu.RUnlock()
	if s, ok := hitMap[clientIP]; ok {
		return s
	}
	return &HitStats{}
}

// recordHit 记录一次拦截
func recordHit(clientIP string, lat, lon float64) {
	hitMapMu.Lock()
	defer hitMapMu.Unlock()
	if hitMap[clientIP] == nil {
		hitMap[clientIP] = &HitStats{}
	}
	hitMap[clientIP].Count++
	hitMap[clientIP].LastTime = time.Now().Unix()
	hitMap[clientIP].LastLat = lat
	hitMap[clientIP].LastLon = lon
}

// loadData 从文件加载数据
func loadData() {
	dataFile := filepath.Join(os.Getenv("HOME"), ".vpn-location", "mitm-data.json")
	if os.Getenv("DATA_DIR") != "" {
		dataFile = filepath.Join(os.Getenv("DATA_DIR"), "mitm-data.json")
	}
	if os.Getenv("MITM_DATA_FILE") != "" {
		dataFile = os.Getenv("MITM_DATA_FILE")
	}

	data, err := os.ReadFile(dataFile)
	if err != nil {
		log.Printf("[MITM] 数据文件不存在或读取失败: %v", err)
		return
	}

	var pd PersistentData
	if err := json.Unmarshal(data, &pd); err != nil {
		log.Printf("[MITM] 数据文件解析失败: %v", err)
		return
	}

	locationMapMu.Lock()
	hitMapMu.Lock()
	defer locationMapMu.Unlock()
	defer hitMapMu.Unlock()

	locationMap = pd.Locations
	hitMap = pd.Hits
	if locationMap == nil {
		locationMap = make(map[string][2]float64)
	}
	if hitMap == nil {
		hitMap = make(map[string]*HitStats)
	}

	log.Printf("[MITM] 数据加载成功: %d 个位置, %d 个统计", len(locationMap), len(hitMap))
}

// saveData 保存数据到文件
func saveData() {
	dataFile := filepath.Join(os.Getenv("HOME"), ".vpn-location", "mitm-data.json")
	if os.Getenv("DATA_DIR") != "" {
		dataFile = filepath.Join(os.Getenv("DATA_DIR"), "mitm-data.json")
	}
	if os.Getenv("MITM_DATA_FILE") != "" {
		dataFile = os.Getenv("MITM_DATA_FILE")
	}

	dir := filepath.Dir(dataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[MITM] 创建数据目录失败: %v", err)
		return
	}

	locationMapMu.RLock()
	hitMapMu.RLock()
	defer locationMapMu.RUnlock()
	defer hitMapMu.RUnlock()

	pd := PersistentData{
		Locations: locationMap,
		Hits:      hitMap,
	}

	data, err := json.MarshalIndent(pd, "", "  ")
	if err != nil {
		log.Printf("[MITM] 数据序列化失败: %v", err)
		return
	}

	if err := os.WriteFile(dataFile, data, 0644); err != nil {
		log.Printf("[MITM] 数据写入失败: %v", err)
		return
	}

	log.Printf("[MITM] 数据保存成功: %s", dataFile)
}

// GetAllLocations 返回所有用户位置（供调试）
func (p *LocationProxy) GetAllLocations() map[string][2]float64 {
	locationMapMu.RLock()
	defer locationMapMu.RUnlock()
	result := make(map[string][2]float64, len(locationMap))
	for k, v := range locationMap {
		result[k] = v
	}
	return result
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

// Start 启动 HTTP CONNECT 代理
func (p *LocationProxy) Start() error {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	loadData()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			saveData()
		}
	}()

	if p.caCert != nil {
		mitmAction := &goproxy.ConnectAction{
			Action:    goproxy.ConnectMitm,
			TLSConfig: goproxy.TLSConfigFromCA(p.caCert),
		}
		proxy.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(
			func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				hostname := strings.Split(host, ":")[0]
				if hostname == "gs-loc.apple.com" || 
					hostname == "gs-loc-cn.apple.com" || 
					hostname == "location.apple.com" || 
					hostname == "location-cn.apple.com" || 
					strings.Contains(hostname, "gs-loc") || 
					strings.Contains(hostname, "location") ||
					strings.Contains(hostname, ".ls.apple.com") ||
					strings.Contains(hostname, "gsp-ssl.ls.apple.com") ||
					strings.Contains(hostname, "ls-cn.apple.com") {
					return mitmAction, host
				}
				return goproxy.OkConnect, host
			},
		))
	}

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		host := strings.Split(req.Host, ":")[0]
		isAppleLocationService := host == "gs-loc.apple.com" || 
			host == "gs-loc-cn.apple.com" || 
			host == "location.apple.com" || 
			host == "location-cn.apple.com" || 
			strings.Contains(host, "gs-loc") || 
			strings.Contains(host, "location") ||
			strings.Contains(host, ".ls.apple.com") ||
			strings.Contains(host, "gsp-ssl.ls.apple.com") ||
			strings.Contains(host, "ls-cn.apple.com")
		
		if isAppleLocationService && req.Method == "POST" {
			// 从请求里取客户端 IP
			clientIP := extractClientIP(req)
			return p.handleLocationRequest(req, clientIP)
		}
		return req, nil
	})

	addr := fmt.Sprintf(":%d", p.port)
	log.Printf("[MITM] HTTPS MITM 代理启动 %s", addr)
	return http.ListenAndServe(addr, proxy)
}

// extractClientIP 从请求中提取客户端真实IP（VPN分配的10.10.0.x）
func extractClientIP(req *http.Request) string {
	// 优先取X-Forwarded-For，取最后一个（VPN客户端IP）
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	// 取RemoteAddr
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

func (p *LocationProxy) handleLocationRequest(req *http.Request, clientIP string) (*http.Request, *http.Response) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MITM] PANIC: %v", r)
		}
	}()

	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return req, nil
	}

	arpc := ArpcDeserialize(body)
	if arpc == nil {
		return req, nil
	}

	var wloc pb.AppleWLoc
	if err := proto.Unmarshal(arpc.Payload, &wloc); err != nil {
		return req, nil
	}

	// 按客户端 IP 取各自的位置
	lat, lon := p.GetSpoofLocation(clientIP)
	log.Printf("[MITM] 客户端 %s 替换位置 -> (%.6f, %.6f)", clientIP, lat, lon)
	recordHit(clientIP, lat, lon)

	iLat := IntFromCoord(lat)
	iLon := IntFromCoord(lon)

	ha := int64(10)
	va := int64(100)
	alt := int64(530)
	uv4 := int64(3)
	mat := int64(63)
	mac := int64(467)

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
		return req, nil
	}
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(respPayload)))

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
	return req, resp
}

// GetCACertPEM 返回 CA 证书 PEM
func (p *LocationProxy) GetCACertPEM() []byte {
	if p.caCert == nil {
		return nil
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.caCert.Certificate[0],
	})
}
