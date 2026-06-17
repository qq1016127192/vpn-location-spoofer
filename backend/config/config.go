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
	VPN struct {
		APIKey        string `yaml:"api_key"`
		SecurityCode  string `yaml:"security_code"`
		ServerAddress string `yaml:"server_address"`
		SearchKey     string `yaml:"search_key"`
		IpsecBin      string `yaml:"ipsec_bin"`
		IpsecSecrets  string `yaml:"ipsec_secrets"`
	} `yaml:"vpn"`
	Cert struct {
		RootCAPath  string `yaml:"root_ca_path"`
		RootKeyPath string `yaml:"root_key_path"`
	} `yaml:"cert"`
}

func Init() {
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
