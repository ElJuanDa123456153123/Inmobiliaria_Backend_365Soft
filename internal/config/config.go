package config

import (
	"os"
	"strconv"
	"time"
)

// Config - Configuración general de la aplicación
type Config struct {
	Server  ServerConfig
	DB      DatabaseConfig
	Redis   RedisConfig
	JWT     JWTConfig
	BancoQr BancoQrConfig
	R2      R2Config
}

// ServerConfig - Configuración del servidor
type ServerConfig struct {
	Port    string
	AppEnv  string
	FrontendURL string
}

// DatabaseConfig - Configuración de base de datos
type DatabaseConfig struct {
	DatabaseURL string
}

// RedisConfig - Configuración de Redis
type RedisConfig struct {
	URL string
}

// JWTConfig - Configuración de JWT
type JWTConfig struct {
	Secret string
}

// BancoQrConfig - Configuración del Banco MC4 QR
type BancoQrConfig struct {
	APIURL      string
	APIKey      string
	ServiceKey  string
	Username    string
	Password    string
	Timeout     time.Duration
	CallbackURL string
}

// R2Config - Configuración de Cloudflare R2
type R2Config struct {
	AccountID   string
	AccessKeyID string
	SecretKey   string
	BucketName  string
}

// LoadConfig carga la configuración desde variables de entorno
func LoadConfig() *Config {
	timeoutSec, _ := strconv.Atoi(getEnv("BANCO_QR_TIMEOUT", "30000"))

	return &Config{
		Server: ServerConfig{
			Port:       getEnv("PORT", "3001"),
			AppEnv:     getEnv("APP_ENV", "development"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
		DB: DatabaseConfig{
			DatabaseURL: getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
		},
		BancoQr: BancoQrConfig{
			APIURL:      getEnv("BANCO_QR_API_URL", ""),
			APIKey:      getEnv("BANCO_QR_API_KEY", ""),
			ServiceKey:  getEnv("BANCO_QR_SERVICE_KEY", ""),
			Username:    getEnv("BANCO_QR_USERNAME", ""),
			Password:    getEnv("BANCO_QR_PASSWORD", ""),
			Timeout:     time.Duration(timeoutSec) * time.Millisecond,
			CallbackURL: getEnv("BANCO_QR_CALLBACK_URL", ""),
		},
		R2: R2Config{
			AccountID:   getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID: getEnv("R2_ACCESS_KEY_ID", ""),
			SecretKey:   getEnv("R2_SECRET_ACCESS_KEY", ""),
			BucketName:  getEnv("R2_BUCKET_NAME", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
