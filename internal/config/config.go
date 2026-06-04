package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseHost       string
	DatabasePort       string
	DatabaseUser       string
	DatabasePassword   string
	DatabaseName       string
	MongoURI           string
	MongoDatabase      string
	Port               string
	SchemaName         string
	UserAuthServiceURL string
	JWTSecret          string
	JWTRefreshSecret   string
	CorsAllowedOrigins string
	AccessTokenMinutes int
	RefreshTokenDays   int
	RedisURL           string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, using system environment variables")
	}

	return &Config{
		DatabaseHost:       getEnv("DB_HOST", ""),
		DatabasePort:       getEnv("DB_PORT", ""),
		DatabaseUser:       getEnv("DB_USER", ""),
		DatabasePassword:   getEnv("DB_PASSWORD", ""),
		DatabaseName:       getEnv("DB_NAME", ""),
		MongoURI:           getEnv("MONGO_URI", ""),
		MongoDatabase:      getEnv("MONGO_DB", "chat_db"),
		Port:               getEnv("PORT", "3000"),
		SchemaName:         getEnv("SCHEMA_NAME", "chat_service"),
		UserAuthServiceURL: getEnv("USER_AUTH_SERVICE_URL", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTRefreshSecret:   getEnv("JWT_REFRESH_SECRET", ""),
		CorsAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3002,http://localhost:8080,https://chat.katgen.pro"),
		AccessTokenMinutes: getEnvAsInt("ACCESS_TOKEN_MINUTES", 15),
		RefreshTokenDays:   getEnvAsInt("REFRESH_TOKEN_DAYS", 30),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	strValue := getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return fallback
}
