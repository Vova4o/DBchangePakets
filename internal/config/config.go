package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config aggregates all application configuration groups.
type Config struct {
	Port   int
	DBType string // "postgres" or "mongodb"
	PG     PGConfig
	Mongo  MongoConfig
}

// PGConfig holds PostgreSQL connection-pool and timeout configurations.
type PGConfig struct {
	URI             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // in minutes
}

// MongoConfig holds MongoDB specific connection configurations.
type MongoConfig struct {
	URI    string
	DBName string
}

// Load compiles configuration values from environment variables with fallback flags.
func Load() (*Config, error) {
	portStr := flag.String("port", getEnv("PORT", "8080"), "HTTP server port")
	dbType := flag.String("db", getEnv("DB_TYPE", "mongodb"), "Database type ('postgres' or 'mongodb')")

	// Postgres specific configurations
	pgURI := flag.String("pg-uri", getEnv("PG_URI", "postgres://postgres:postgres@localhost:5432/demo?sslmode=disable"), "PostgreSQL URI")
	pgMaxOpen := flag.Int("pg-max-open", getEnvInt("PG_MAX_OPEN", 25), "PostgreSQL max open connections")
	pgMaxIdle := flag.Int("pg-max-idle", getEnvInt("PG_MAX_IDLE", 25), "PostgreSQL max idle connections")
	pgLifetime := flag.Int("pg-lifetime-min", getEnvInt("PG_LIFETIME_MIN", 5), "PostgreSQL connection max lifetime in minutes")

	// Mongo specific configurations
	mongoURI := flag.String("mongo-uri", getEnv("MONGO_URI", "mongodb://localhost:27017"), "MongoDB URI")
	mongoDBName := flag.String("mongo-dbname", getEnv("MONGO_DBNAME", "demo_db"), "MongoDB database name")

	flag.Parse()

	port, err := strconv.Atoi(*portStr)
	if err != nil {
		return nil, fmt.Errorf("config parse port: %w", err)
	}

	if *dbType != "postgres" && *dbType != "mongodb" {
		return nil, fmt.Errorf("config validate: db_type must be either 'postgres' or 'mongodb'")
	}

	return &Config{
		Port:   port,
		DBType: *dbType,
		PG: PGConfig{
			URI:             *pgURI,
			MaxOpenConns:    *pgMaxOpen,
			MaxIdleConns:    *pgMaxIdle,
			ConnMaxLifetime: *pgLifetime,
		},
		Mongo: MongoConfig{
			URI:    *mongoURI,
			DBName: *mongoDBName,
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}
