package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	ServerPort     string
	JWTSecret      string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5431"),
		DBUser:         getEnv("DB_USER", "kanban_user"),
		DBPassword:     getEnv("DB_PASSWORD", "kanban_pass"),
		DBName:         getEnv("DB_NAME", "kanban_db"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "supersecretkey"),
	}
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
