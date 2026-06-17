package service

import (
	"fmt"
	"sync"
	"time"
)

// LocationService 位置服务
type LocationService struct {
	mu            sync.RWMutex
	lockedLocations map[string]Location // userID -> Location
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
	})
	return locationService
}

// LockLocation 锁定用户位置
func (s *LocationService) LockLocation(userID string, lat, lng float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.lockedLocations[userID] = Location{
		Latitude:  lat,
		Longitude: lng,
		Timestamp: getCurrentTimestamp(),
	}
	
	return nil
}

// GetLockedLocation 获取用户锁定的位置
func (s *LocationService) GetLockedLocation(userID string) (Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	loc, exists := s.lockedLocations[userID]
	if !exists {
		return Location{}, fmt.Errorf("未找到锁定的位置")
	}
	
	return loc, nil
}

// UnlockLocation 解锁用户位置
func (s *LocationService) UnlockLocation(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.lockedLocations, userID)
	return nil
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
