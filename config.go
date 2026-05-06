package main

import "os"

const (
	defaultAdminUser     = "admin"
	defaultAdminPassword = "mSm-4Qb9-vY2-Zt7-Kp5"
)

type AppConfig struct {
	AdminUser     string
	AdminPassword string
}

func DefaultAppConfig() AppConfig {
	return AppConfig{
		AdminUser:     valueOr(os.Getenv("ADMIN_USER"), defaultAdminUser),
		AdminPassword: valueOr(os.Getenv("ADMIN_PASSWORD"), defaultAdminPassword),
	}
}
