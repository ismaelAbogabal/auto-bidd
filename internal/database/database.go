package database

import (
	"log"

	"github.com/ismaelfi/auto-bidd/internal/config"
	"github.com/ismaelfi/auto-bidd/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Silent
	if cfg.IsDev() {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`).Error
	if err != nil {
		return err
	}

	return db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.UserProfile{},
		&models.ToneExample{},
		&models.PortfolioItem{},
		&models.Bid{},
		&models.ChatMessage{},
		&models.Template{},
	)
}
