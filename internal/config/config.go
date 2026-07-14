package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost     string
	DBHostAddr string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBURL      string
	LOCALDBURL string
}

func LoadConfig() Config {

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	sslMode := os.Getenv("DB_SSLMODE")
	localDbURL := os.Getenv("LOCAL_DB_URL")

	if sslMode == "" {
		sslMode = "require"
	}

	// Use host for SSL verification, hostaddr forces IPv4
	dbURL := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbName, sslMode,
	)

	return Config{
		DBHost:     host,
		DBPort:     port,
		DBUser:     user,
		DBPassword: password,
		DBName:     dbName,
		DBSSLMode:  sslMode,
		DBURL:      dbURL,
		LOCALDBURL: localDbURL,
	}
}

