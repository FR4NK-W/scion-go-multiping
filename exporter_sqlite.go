package main

import (
	"os"

	"gorm.io/driver/sqlite" // Sqlite driver based on CGO
	// "github.com/glebarez/sqlite" // Pure go SQLite driver, checkout https://github.com/glebarez/sqlite for details
	"gorm.io/gorm"
)

type SQLiteExporter struct {
	DbPath string
	db     *gorm.DB
}

func NewSQLiteExporter() *SQLiteExporter {
	exporter := &SQLiteExporter{}
	sqlitePath := os.Getenv("EXPORTER_SQLITE_DB_PATH")
	if sqlitePath == "" {
		sqlitePath = "pingmetrics.db"
	}
	exporter.DbPath = sqlitePath
	return exporter
}

func (exporter *SQLiteExporter) Init() error {

	// Create the sqlite file if it's not available
	if _, err := os.Stat(exporter.DbPath); err != nil {
		if _, err = os.Create(exporter.DbPath); err != nil {
			return err
		}
	}

	db, err := gorm.Open(sqlite.Open(exporter.DbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&PingResult{}, &PathStatistics{}, &IPPingResult{})
	if err != nil {
		return err
	}

	exporter.db = db
	return nil
}

func (exporter *SQLiteExporter) WritePathStatistic(statistic PathStatistics) error {
	dbResult := exporter.db.Create(&statistic)
	if dbResult.Error != nil {
		return dbResult.Error
	}

	return nil
}

func (exporter *SQLiteExporter) WritePingResult(result PingResult) error {
	dbResult := exporter.db.Create(&result)
	if dbResult.Error != nil {
		return dbResult.Error
	}
	return nil
}

func (exporter *SQLiteExporter) WriteIPPingResult(result IPPingResult) error {
	dbResult := exporter.db.Create(&result)
	if dbResult.Error != nil {
		return dbResult.Error
	}
	return nil
}
