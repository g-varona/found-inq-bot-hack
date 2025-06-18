package storage

import (
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the database connection and runs migrations
func InitDB(dbPath string) (*gorm.DB, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// Configure GORM
	config := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true, // Enable prepared statements for better performance
	}

	// Open database connection with pragma settings for SQLite
	dsn := dbPath + "?cache=shared&mode=rwc&_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=1"
	db, err := gorm.Open(sqlite.Open(dsn), config)
	if err != nil {
		return nil, err
	}

	// Configure SQLite specific settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// Run auto migrations in specific order
	if err := db.AutoMigrate(&Inquiry{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&SearchResult{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&ReactionEvent{}); err != nil {
		return nil, err
	}

	return db, nil
}
