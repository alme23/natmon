package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort   string
	DatabasePath string
	SNMPTimeout  int
	SNMPRetries  int
}

func Load() *Config {
	return &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		DatabasePath: getEnv("DATABASE_PATH", "./natmon.db"),
		SNMPTimeout:  getEnvInt("SNMP_TIMEOUT", 2),
		SNMPRetries:  getEnvInt("SNMP_RETRIES", 1),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
