package model

import (
	"fmt"
	"log"

	"github.com/vpn-location/backend/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() error {
	cfg := config.Config.Database

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	log.Println("数据库连接成功")

	// 自动迁移表结构
	err = AutoMigrate()
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %v", err)
	}

	return nil
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate() error {
	return DB.AutoMigrate(&User{})
}
