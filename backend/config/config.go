package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var Config *AppConfig

type AppConfig struct {
	Server struct {
		Port string `yaml:"port"`
		Mode string `yaml:"mode"`
	} `yaml:"server"`
	Database struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		Charset  string `yaml:"charset"`
	} `yaml:"database"`
	VPN struct {
		APIKey        string `yaml:"api_key"`
		SecurityCode  string `yaml:"security_code"`
		ServerAddress string `yaml:"server_address"`
		SearchKey     string `yaml:"search_key"`
		IpsecBin      string `yaml:"ipsec_bin"`      // StrongSwan ipsec 可执行文件路径
		IpsecSecrets  string `yaml:"ipsec_secrets"`  // ipsec.secrets 文件路径
	} `yaml:"vpn"`
	Cert struct {
		RootCAPath  string `yaml:"root_ca_path"`
		RootKeyPath string `yaml:"root_key_path"`
	} `yaml:"cert"`
	Auth struct {
		JWTSecret  string `yaml:"jwt_secret"`
		ExpireHours int  `yaml:"expire_hours"`
	} `yaml:"auth"`
}

func Init() {
	// 支持环境变量 CONFIG_PATH 指定配置文件路径
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("读取配置文件失败: %v", err))
	}
	
	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		panic(fmt.Sprintf("解析配置文件失败: %v", err))
	}
}
