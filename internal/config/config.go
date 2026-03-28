package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server ServerConfig
	DB     DBConfig
	JWT    JWTConfig
	CORS   CORSConfig
	MinIO  MinIOConfig
}

type ServerConfig struct {
	Port    string
	GinMode string
}

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

type MinIOConfig struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	BucketProjects string
	BucketDesign   string
	BucketCutting  string
	BucketAvatars  string
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env file not found, using environment variables")
	}

	accessTTL, _    := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	refreshTTL, _   := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))
	connLifetime, _ := time.ParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "5m"))

	rawOrigins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3001")
	origins := []string{}
	for _, o := range strings.Split(rawOrigins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}

	return &Config{
		Server: ServerConfig{
			Port:    getEnv("PORT", "8181"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", "jevon_crm"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: connLifetime,
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", ""),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
			AccessTTL:     accessTTL,
			RefreshTTL:    refreshTTL,
		},
		CORS: CORSConfig{
			AllowedOrigins: origins,
		},
		MinIO: MinIOConfig{
			Endpoint:       getEnv("MINIO_ENDPOINT", "172.20.40.6:9000"),
			AccessKey:      getEnv("MINIO_ACCESS_KEY", "jevon_backend"),
			SecretKey:      getEnv("MINIO_SECRET_KEY", "JevonBackend@2026"),
			UseSSL:         getEnv("MINIO_USE_SSL", "false") == "true",
			BucketProjects: getEnv("MINIO_BUCKET_PROJECTS", "jevon-projects"),
			BucketDesign:   getEnv("MINIO_BUCKET_DESIGN", "jevon-design"),
			BucketCutting:  getEnv("MINIO_BUCKET_CUTTING", "jevon-cutting"),
			BucketAvatars:  getEnv("MINIO_BUCKET_AVATARS", "jevon-avatars"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
