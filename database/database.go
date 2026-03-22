package database

import (
	"log"

	"transit-server/config"
	"transit-server/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database connection.
var DB *gorm.DB

// Connect initializes the SQLite database and runs migrations.
func Connect() {
	var err error

	DB, err = gorm.Open(sqlite.Open(config.AppConfig.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Enable WAL mode for better concurrent read/write performance
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	_, err = sqlDB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Fatalf("Failed to enable WAL mode: %v", err)
	}
	_, err = sqlDB.Exec("PRAGMA foreign_keys=ON;")
	if err != nil {
		log.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Auto-migrate — creates tables and indexes defined in model struct tags
	err = DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	log.Println("Database connected and migrated successfully")
}
