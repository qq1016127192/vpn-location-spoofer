package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LocationService 位置服务
type LocationService struct {
	mu              sync.RWMutex
	lockedLocations map[string]Location // clientIP -> Location
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

var locationService *LocationService
var once sync.Once

// GetLocationService 获取位置服务单例
func GetLocationService() *LocationService {
	once.Do(func() {
		locationService = &LocationService{
			lockedLocations: make(map[string]Location),
		}
		locationService.loadData()
	})
	return locationService
}

// LockLocation 锁定位置（按 VPN 客户端 IP 隔离）
func (s *LocationService) LockLocation(clientIP string, lat, lng float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lockedLocations[clientIP] = Location{
		Latitude:  lat,
		Longitude: lng,
		Timestamp: getCurrentTimestamp(),
	}

	go s.saveData()
	return nil
}

// GetLockedLocation 获取指定 IP 锁定的位置
func (s *LocationService) GetLockedLocation(clientIP string) (Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	loc, exists := s.lockedLocations[clientIP]
	if !exists {
		return Location{}, fmt.Errorf("未找到锁定的位置")
	}

	return loc, nil
}

// UnlockLocation 解锁指定 IP 的位置
func (s *LocationService) UnlockLocation(clientIP string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.lockedLocations, clientIP)
	return nil
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

const locationDataFile = "/opt/vpn-data/location-data.json"

// loadData 从文件加载锁定位置数据
func (s *LocationService) loadData() {
	dataFile := locationDataFile
	if os.Getenv("DATA_DIR") != "" {
		dataFile = filepath.Join(os.Getenv("DATA_DIR"), "location-data.json")
	}
	if os.Getenv("LOCATION_DATA_FILE") != "" {
		dataFile = os.Getenv("LOCATION_DATA_FILE")
	}

	data, err := os.ReadFile(dataFile)
	if err != nil {
		return
	}

	var mapData map[string]Location
	if err := json.Unmarshal(data, &mapData); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.lockedLocations = mapData
	if s.lockedLocations == nil {
		s.lockedLocations = make(map[string]Location)
	}
}

// saveData 保存锁定位置数据到文件
func (s *LocationService) saveData() {
	dataFile := locationDataFile
	if os.Getenv("DATA_DIR") != "" {
		dataFile = filepath.Join(os.Getenv("DATA_DIR"), "location-data.json")
	}
	if os.Getenv("LOCATION_DATA_FILE") != "" {
		dataFile = os.Getenv("LOCATION_DATA_FILE")
	}

	dir := filepath.Dir(dataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.lockedLocations, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(dataFile, data, 0644)
}
