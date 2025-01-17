package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/sqlite" // Sqlite driver based on CGO
	// "github.com/glebarez/sqlite" // Pure go SQLite driver, checkout https://github.com/glebarez/sqlite for details
	"gorm.io/gorm"
)

type SQLiteExporter struct {
	DbPath              string
	db                  *gorm.DB
	scionPings          []PingResult
	pathStatistics      []PathStatistics
	ipPings             []IPPingResult
	scionMutex          sync.Mutex
	ipMutex             sync.Mutex
	pathStatisticsMutex sync.Mutex
	batchSize           int
}

func NewSQLiteExporter() *SQLiteExporter {
	exporter := &SQLiteExporter{
		batchSize: 1,
	}
	sqlitePath := os.Getenv("EXPORTER_SQLITE_DB_PATH")
	if sqlitePath == "" {
		sqlitePath = "pingmetrics.db"
	}

	sqliteBatchSize := os.Getenv("EXPORTER_SQLITE_DB_BATCH_SIZE")
	if sqliteBatchSize != "" {
		str, err := strconv.Atoi(sqliteBatchSize)
		if err == nil {
			exporter.batchSize = str
		}
	}

	exporter.DbPath = sqlitePath
	return exporter
}

func (exporter *SQLiteExporter) InitDaily() error {
	exporter.pathStatisticsMutex.Lock()
	exporter.scionMutex.Lock()
	exporter.ipMutex.Lock()
	defer exporter.ipMutex.Unlock()
	defer exporter.scionMutex.Unlock()
	defer exporter.pathStatisticsMutex.Unlock()

	if exporter.db != nil {
		err := exporter.Close()
		if err != nil {
			return err
		}
	}

	exporter.DbPath = strings.ReplaceAll(exporter.DbPath, ".db", "_"+time.Now().UTC().Format("2006-01-02")+".db")
	Log.Info("Connecting to database ", exporter.DbPath)
	// Create the sqlite file if it's not available
	if _, err := os.Stat(exporter.DbPath); err != nil {
		if _, err = os.Create(exporter.DbPath); err != nil {
			return err
		}
	}

	db, err := gorm.Open(sqlite.Open(exporter.DbPath), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&PingResult{}, &PathStatistics{}, &IPPingResult{})
	if err != nil {
		return err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return err
	}

	// We mutex our selves, this ensures no locking in the driver level
	sqlDb.SetMaxOpenConns(1)

	_, err = sqlDb.Exec("PRAGMA synchronous=OFF")
	if err != nil {
		return err
	}

	Log.Info("Database connection established to file ", exporter.DbPath)

	exporter.db = db
	return nil
}

func (exporter *SQLiteExporter) Close() error {
	sqlDB, _ := exporter.db.DB()
	// Close
	return sqlDB.Close()
}

func (exporter *SQLiteExporter) WritePathStatistic(statistic PathStatistics) error {
	exporter.pathStatisticsMutex.Lock()
	defer exporter.pathStatisticsMutex.Unlock()
	fmt.Printf("fingerprints: %s, paths: %s\n", statistic.Fingerprints, statistic.Paths)

	if exporter.batchSize == 1 {
		dbResult := exporter.db.Create(&statistic)
		if dbResult.Error != nil {
			return dbResult.Error
		}
		return nil
	}

	exporter.pathStatistics = append(exporter.pathStatistics, statistic)
	if len(exporter.pathStatistics) >= exporter.batchSize {
		dbResult := exporter.db.Create(&exporter.pathStatistics)
		exporter.pathStatistics = nil // Clear the slice after flushing
		if dbResult.Error != nil {
			return dbResult.Error
		}
	}

	return nil
}

func (exporter *SQLiteExporter) WritePingResult(result PingResult) error {

	fmt.Printf("fingerprints: %s\n", result.Fingerprint)
	exporter.scionMutex.Lock()
	defer exporter.scionMutex.Unlock()

	if exporter.batchSize == 1 {
		dbResult := exporter.db.Create(&result)
		if dbResult.Error != nil {
			return dbResult.Error
		}
		return nil
	}

	exporter.scionPings = append(exporter.scionPings, result)
	if len(exporter.scionPings) >= exporter.batchSize {
		dbResult := exporter.db.Create(&exporter.scionPings)
		exporter.scionPings = nil // Clear the slice after flushing
		if dbResult.Error != nil {
			return dbResult.Error
		}
	}

	return nil
}

func (exporter *SQLiteExporter) WriteIPPingResult(result IPPingResult) error {
	exporter.ipMutex.Lock()
	defer exporter.ipMutex.Unlock()

	if exporter.batchSize == 1 {
		dbResult := exporter.db.Create(&result)
		if dbResult.Error != nil {
			return dbResult.Error
		}
		return nil
	}

	exporter.ipPings = append(exporter.ipPings, result)
	if len(exporter.ipPings) >= exporter.batchSize {
		dbResult := exporter.db.Create(&exporter.ipPings)
		if dbResult.Error != nil {
			return dbResult.Error
		}
		exporter.ipPings = nil
	}

	return nil
}
