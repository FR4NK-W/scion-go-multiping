package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PingResult struct {
	SrcSCIONAddr    string
	DstSCIONAddr    string
	Success         bool
	RTT             float64
	Fingerprint     string
	PingTime        string
	SuccessfulPings int
	MaxPings        int
}

type IPPingResult struct {
	SrcAddr  string
	DstAddr  string
	Success  bool
	RTT      float64
	PingTime string
}

type PathStatistics struct {
	SrcSCIONAddr   string
	DstSCIONAddr   string
	Paths          string
	Fingerprints   string
	Success        bool
	MinRTT         float64
	MaxRTT         float64
	MinHops        int
	MaxHops        int
	LookupTime     string
	ActivePaths    int
	ProbedPaths    int
	AvailablePaths int
}

func processSQLiteFile(filePath string, pgDB *gorm.DB, batchSize int) error {
	// Connect to SQLite database
	sqliteDB, err := gorm.Open(sqlite.Open(filePath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open SQLite database %s: %w", filePath, err)
	}

	// Migrate SQLite tables to match the GORM models
	if err := sqliteDB.AutoMigrate(&PingResult{}, &IPPingResult{}, &PathStatistics{}); err != nil {
		return fmt.Errorf("failed to migrate SQLite tables: %w", err)
	}

	// Process PingResult in batches
	var offset int
	for {
		var pingResults []PingResult
		err := sqliteDB.Order("ping_time").Limit(batchSize).Offset(offset).Find(&pingResults).Error
		if err != nil {
			return fmt.Errorf("failed to read PingResults batch: %w", err)
		}
		if len(pingResults) == 0 {
			break
		}
		if err := pgDB.Create(&pingResults).Error; err != nil {
			return fmt.Errorf("failed to insert PingResults batch into PostgreSQL: %w", err)
		}
		fmt.Println("Inserted", len(pingResults), "PingResults")
		offset += batchSize
	}

	// Process IPPingResult in batches
	offset = 0
	for {
		var ipPingResults []IPPingResult
		err := sqliteDB.Order("ping_time").Limit(batchSize).Offset(offset).Find(&ipPingResults).Error
		if err != nil {
			return fmt.Errorf("failed to read IPPingResults batch: %w", err)
		}
		if len(ipPingResults) == 0 {
			break
		}
		if err := pgDB.Create(&ipPingResults).Error; err != nil {
			return fmt.Errorf("failed to insert IPPingResults batch into PostgreSQL: %w", err)
		}
		fmt.Println("Inserted", len(ipPingResults), "IPPingResults")
		offset += batchSize
	}

	// Process PathStatistics in batches
	offset = 0
	for {
		var pathStats []PathStatistics
		err := sqliteDB.Order("lookup_time").Limit(batchSize).Offset(offset).Find(&pathStats).Error
		if err != nil {
			return fmt.Errorf("failed to read PathStatistics batch: %w", err)
		}
		if len(pathStats) == 0 {
			break
		}
		if err := pgDB.Create(&pathStats).Error; err != nil {
			return fmt.Errorf("failed to insert PathStatistics batch into PostgreSQL: %w", err)
		}
		fmt.Println("Inserted", len(pathStats), "PathStatistics")
		offset += batchSize
	}

	// Rename the SQLite file to .db.done
	donePath := strings.Replace(filePath, ".db", ".db.done", 1)
	if err := os.Rename(filePath, donePath); err != nil {
		return fmt.Errorf("failed to rename file to %s: %w", donePath, err)
	}

	log.Printf("Processed and renamed %s to %s", filePath, donePath)
	return nil
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <folder_path> <batch_size>", os.Args[0])
	}

	folderPath := os.Args[1]
	batchSize, err := strconv.Atoi(os.Args[2])
	if err != nil || batchSize <= 0 {
		log.Fatalf("Invalid batch size: %s", os.Args[2])
	}

	// Connect to PostgreSQL database
	pgDSN := "host=timescaledb user=postgres password=ephee9iechahwaehoosh6Eiz9mohr7Mu dbname=postgres port=5432 sslmode=disable"
	pgDB, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	fmt.Println("Connected to PostgreSQL")

	err = pgDB.AutoMigrate(&PingResult{}, &IPPingResult{}, &PathStatistics{})
	if err != nil {
		log.Fatalf("Failed to migrate PostgreSQL tables: %v", err)
	}

	for {
		// Recursively walk through the folder and process .db files
		err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".db") {
				if err := processSQLiteFile(path, pgDB, batchSize); err != nil {
					log.Printf("Error processing file %s: %v", path, err)
				}
			}
			return nil
		})

		if err != nil {
			log.Fatalf("Error walking the directory: %v", err)
		}

		log.Println("Processing completed.")
		time.Sleep(10 * time.Second)

	}

}
