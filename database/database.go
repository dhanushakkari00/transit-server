package database

import (
	"context"
	"log"
	"net/url"
	"strings"
	"time"

	"transit-server/config"
	"transit-server/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database connection.
var DB *gorm.DB

const dbPingTimeout = 5 * time.Second

// Connect initializes the configured database and runs migrations.
func Connect() {
	var err error

	backend, target := connectionTarget()
	log.Printf("DB startup: using %s (%s)", backend, target)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	if config.AppConfig.DatabaseURL != "" {
		log.Printf("DB startup: opening PostgreSQL connection to %s", target)
		DB, err = gorm.Open(postgres.Open(config.AppConfig.DatabaseURL), gormConfig)
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
	} else {
		log.Printf("DB startup: opening SQLite database at %s", target)
		DB, err = gorm.Open(sqlite.Open(config.AppConfig.DBPath), gormConfig)
		if err != nil {
			log.Fatalf("Failed to connect to SQLite: %v", err)
		}
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	if config.AppConfig.DatabaseURL == "" {
		log.Println("DB startup: enabling SQLite pragmas")
		_, err = sqlDB.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			log.Fatalf("Failed to enable WAL mode: %v", err)
		}
		_, err = sqlDB.Exec("PRAGMA foreign_keys=ON;")
		if err != nil {
			log.Fatalf("Failed to enable foreign keys: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbPingTimeout)
	defer cancel()
	log.Printf("DB startup: pinging %s", backend)
	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Printf("DB startup: %s ping succeeded", backend)

	// Auto-migrate — creates tables and indexes defined in model struct tags
	log.Println("DB startup: running auto-migrations")
	err = DB.AutoMigrate(
		&models.User{},
		&models.Driver{},
		&models.Aggregator{},
		&models.DriverAggregatorMapping{},
		&models.Route{},
		&models.Trip{},
		&models.ActiveTrip{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
	log.Println("DB startup: auto-migrations complete")

	if config.AppConfig.DatabaseURL != "" {
		log.Println("PostgreSQL database connected and migrated successfully")
		return
	}

	log.Println("SQLite database connected and migrated successfully")
}

func connectionTarget() (string, string) {
	if config.AppConfig.DatabaseURL == "" {
		return "SQLite", config.AppConfig.DBPath
	}

	parsedURL, err := url.Parse(config.AppConfig.DatabaseURL)
	if err != nil {
		return "PostgreSQL", "DATABASE_URL"
	}

	target := parsedURL.Hostname()
	if port := parsedURL.Port(); port != "" {
		target += ":" + port
	}

	dbName := strings.TrimPrefix(parsedURL.Path, "/")
	if dbName != "" {
		target += "/" + dbName
	}

	return "PostgreSQL", target
}
