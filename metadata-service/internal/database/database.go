package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"metadata-service/internal/models"
)

var DB *gorm.DB

func ConnectDatabase() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second, // Slow SQL threshold
			LogLevel:      logger.Info, // Log level
			Colorful:      true,        // Disable color
		},
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection established")

	// Auto-migrate schema
	// This will create tables, missing foreign keys, constraints, columns and indexes.
	// It will NOT delete unneeded columns, to protect your data.
	err = DB.AutoMigrate(&models.EntityDefinition{}, &models.AttributeDefinition{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database schema: %v", err)
	}
	log.Println("Database schema migration completed.")
}

// GetDB returns the gorm database instance
func GetDB() *gorm.DB {
	return DB
}
