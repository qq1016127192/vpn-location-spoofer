package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User 用户模型
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	VPNUser      string `json:"vpn_user"`
	VPNPass      string `json:"vpn_pass"`
	CreatedAt    int64  `json:"created_at"`
}

// UserStore JSON 文件存储的用户仓库
type UserStore struct {
	mu       sync.RWMutex
	filePath string
	users    map[string]*User // username -> User
}

var store *UserStore
var storeOnce sync.Once

// InitUserStore 初始化用户存储
func InitUserStore(dataDir string) {
	storeOnce.Do(func() {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			panic(fmt.Sprintf("创建数据目录失败: %v", err))
		}
		fp := filepath.Join(dataDir, "users.json")
		s := &UserStore{
			filePath: fp,
			users:    make(map[string]*User),
		}
		s.load()
		store = s
	})
}

func GetStore() *UserStore {
	return store
}

func (s *UserStore) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return // 文件不存在,空存储
	}
	var list []*User
	if err := json.Unmarshal(data, &list); err != nil {
		return
	}
	for _, u := range list {
		s.users[u.Username] = u
	}
}

func (s *UserStore) save() error {
	list := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

// CreateUser 注册新用户,返回用户对象
func (s *UserStore) CreateUser(username, password string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("用户名已存在")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成密码哈希失败: %v", err)
	}

	vpnUser := "vpn_" + randHex(8)
	vpnPass := randHex(12)

	user := &User{
		ID:           randHex(16),
		Username:     username,
		PasswordHash: string(hash),
		VPNUser:      vpnUser,
		VPNPass:      vpnPass,
		CreatedAt:    nowUnix(),
	}
	s.users[username] = user

	if err := s.save(); err != nil {
		delete(s.users, username)
		return nil, fmt.Errorf("保存用户失败: %v", err)
	}
	return user, nil
}

// Authenticate 验证登录
func (s *UserStore) Authenticate(username, password string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	if !ok {
		return nil, fmt.Errorf("用户不存在")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("密码错误")
	}
	return user, nil
}

// GetByUsername 按用户名查询
func (s *UserStore) GetByUsername(username string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[username]
	return u, ok
}

// GetAllUsers 返回所有用户(不含密码哈希)
func (s *UserStore) GetAllUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u)
	}
	return list
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func nowUnix() int64 {
	return time.Now().Unix()
}