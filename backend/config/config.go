// Package config 提供环境变量加载和应用配置管理
package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDSN                    string
	JWTSecret                string
	ServerPort               string
	LogLevel                 string
	BcryptCost               int
	MaxExchangeQty           int
	AutomationPythonBin      string
	AutomationScriptPath     string
	AutomationTimeoutSeconds int
	AutomationMaxRetries     int
}

// Load 从环境变量加载配置,缺失必填项时 panic
// 优先加载项目根目录的 .env 文件
func Load() *Config {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	return &Config{
		DBDSN:                    requireEnv("DB_DSN"),
		JWTSecret:                requireEnv("JWT_SECRET"),
		ServerPort:               envDefault("SERVER_PORT", "8080"),
		LogLevel:                 envDefault("LOG_LEVEL", "info"),
		BcryptCost:               envIntDefault("BCRYPT_COST", 12),
		MaxExchangeQty:           envIntDefault("MAX_EXCHANGE_QUANTITY", 50),
		AutomationPythonBin:      envDefault("AUTOMATION_PYTHON_BIN", "python3"),
		AutomationScriptPath:     envDefault("AUTOMATION_SCRIPT_PATH", "automation/yfjc_runner.py"),
		AutomationTimeoutSeconds: envIntDefault("AUTOMATION_TIMEOUT_SECONDS", 120),
		AutomationMaxRetries:     envIntDefault("AUTOMATION_MAX_RETRIES", 2),
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required env: " + key)
	}
	return v
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}
