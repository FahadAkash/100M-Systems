package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port          string
	BaseURL       string
	StorageMode   string
	DatabaseURL   string
	RedisURL      string
	RedisHost     string
	RedisPort              string
	RedisPassword          string
	RateLimitEnabled       bool
	RateLimitMaxRequests   int
	RateLimitWindowSeconds int
	KafkaEnabled           bool
	KafkaBrokers           []string
	KafkaTopic             string
	KafkaGroupID           string
}

func Load() Config {
	// Try loading .env file if available
	loadEnvFile(".env")

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		dbUser := getEnv("DB_USER", getEnv("POSTGRES_USER", "postgres"))
		dbPass := getEnv("DB_PASSWORD", getEnv("POSTGRES_PASSWORD", ""))
		dbHost := getEnv("DB_HOST", "127.0.0.1")
		dbPort := getEnv("DB_PORT", getEnv("POSTGRES_PORT", "5432"))
		dbName := getEnv("DB_NAME", getEnv("POSTGRES_DB", "appdb"))

		if dbPass != "" {
			dbURL = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
		} else {
			dbURL = fmt.Sprintf("postgresql://%s@%s:%s/%s?sslmode=disable", dbUser, dbHost, dbPort, dbName)
		}
	}

	redisURL := strings.TrimSpace(os.Getenv("REDIS_URL"))
	redisHost := getEnv("REDIS_HOST", "127.0.0.1")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPass := getEnv("REDIS_PASSWORD", "")

	if redisURL == "" && redisHost != "" {
		if redisPass != "" {
			redisURL = fmt.Sprintf("redis://:%s@%s:%s/0", redisPass, redisHost, redisPort)
		} else {
			redisURL = fmt.Sprintf("redis://%s:%s/0", redisHost, redisPort)
		}
	}

	rateLimitEnabled := getEnv("RATE_LIMIT_ENABLED", "true") != "false"
	rateLimitMax, err := strconv.Atoi(getEnv("RATE_LIMIT_MAX_REQUESTS", "60"))
	if err != nil || rateLimitMax <= 0 {
		rateLimitMax = 60
	}
	rateLimitWindow, err := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW_SECONDS", "60"))
	if err != nil || rateLimitWindow <= 0 {
		rateLimitWindow = 60
	}

	kafkaEnabled := getEnv("KAFKA_ENABLED", "false") == "true"
	kafkaBrokersRaw := getEnv("KAFKA_BROKERS", "127.0.0.1:9092")
	var kafkaBrokers []string
	for _, b := range strings.Split(kafkaBrokersRaw, ",") {
		b = strings.TrimSpace(b)
		if b != "" {
			kafkaBrokers = append(kafkaBrokers, b)
		}
	}
	if len(kafkaBrokers) == 0 {
		kafkaBrokers = []string{"127.0.0.1:9092"}
	}
	kafkaTopic := getEnv("KAFKA_TOPIC", "url-clicks")
	kafkaGroupID := getEnv("KAFKA_GROUP_ID", "url-shortener-analytics")

	return Config{
		Port:                   getEnv("PORT", "8080"),
		BaseURL:                strings.TrimRight(getEnv("BASE_URL", "http://localhost:8080"), "/"),
		StorageMode:            strings.ToLower(getEnv("STORAGE_MODE", "memory")),
		DatabaseURL:            dbURL,
		RedisURL:               redisURL,
		RedisHost:              redisHost,
		RedisPort:              redisPort,
		RedisPassword:          redisPass,
		RateLimitEnabled:       rateLimitEnabled,
		RateLimitMaxRequests:   rateLimitMax,
		RateLimitWindowSeconds: rateLimitWindow,
		KafkaEnabled:           kafkaEnabled,
		KafkaBrokers:           kafkaBrokers,
		KafkaTopic:             kafkaTopic,
		KafkaGroupID:           kafkaGroupID,
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func (c Config) PortNumber() (int, error) {
	return strconv.Atoi(c.Port)
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
				val = val[1 : len(val)-1]
			}
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, val)
			}
		}
	}
}
