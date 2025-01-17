package main

import (
	"os"
	"testing"
	"time"
)

func TestSQLiteExporter_Init(t *testing.T) {
	exporter := NewSQLiteExporter()
	exporter.DbPath = "test_pingmetrics.db"
	defer os.Remove(exporter.DbPath) // Clean up test database after the test

	if err := exporter.InitDaily(); err != nil {
		t.Fatalf("Failed to initialize SQLiteExporter: %v", err)
	}

	if _, err := os.Stat(exporter.DbPath); os.IsNotExist(err) {
		t.Errorf("Expected database file %s to be created, but it does not exist", exporter.DbPath)
	}
}

func TestSQLiteExporter_WritePingResult(t *testing.T) {
	exporter := NewSQLiteExporter()
	exporter.DbPath = "test_pingmetrics.db"
	defer os.Remove(exporter.DbPath) // Clean up test database after the test

	if err := exporter.InitDaily(); err != nil {
		t.Fatalf("Failed to initialize SQLiteExporter: %v", err)
	}

	pingResult := PingResult{
		SrcSCIONAddr:    "1-ff00:0:110",
		DstSCIONAddr:    "1-ff00:0:111",
		Success:         true,
		RTT:             20.5,
		PingTime:        time.Now(),
		SuccessfulPings: 5,
		MaxPings:        10,
	}

	if err := exporter.WritePingResult(pingResult); err != nil {
		t.Errorf("Failed to write PingResult: %v", err)
	}

	var fetched PingResult
	if err := exporter.db.First(&fetched, "src_scion_addr = ? AND dst_scion_addr = ?", pingResult.SrcSCIONAddr, pingResult.DstSCIONAddr).Error; err != nil {
		t.Errorf("Failed to fetch PingResult from database: %v", err)
	}

	if fetched.RTT != pingResult.RTT {
		t.Errorf("Expected RTT %v, got %v", pingResult.RTT, fetched.RTT)
	}
}

func TestSQLiteExporter_WritePathStatistic(t *testing.T) {
	exporter := NewSQLiteExporter()
	exporter.DbPath = "test_pingmetrics.db"
	defer os.Remove(exporter.DbPath) // Clean up test database after the test

	if err := exporter.InitDaily(); err != nil {
		t.Fatalf("Failed to initialize SQLiteExporter: %v", err)
	}

	pathStatistic := PathStatistics{
		SrcSCIONAddr: "1-ff00:0:110",
		DstSCIONAddr: "1-ff00:0:111",
		Success:      true,
		MinRTT:       10.2,
		MaxRTT:       50.8,
		LookupTime:   time.Now(),
		ActivePaths:  3,
		ProbedPaths:  5,
	}

	if err := exporter.WritePathStatistic(pathStatistic); err != nil {
		t.Errorf("Failed to write PathStatistic: %v", err)
	}

	var fetched PathStatistics
	if err := exporter.db.First(&fetched, "src_scion_addr = ? AND dst_scion_addr = ?", pathStatistic.SrcSCIONAddr, pathStatistic.DstSCIONAddr).Error; err != nil {
		t.Errorf("Failed to fetch PathStatistics from database: %v", err)
	}

	if fetched.MinRTT != pathStatistic.MinRTT {
		t.Errorf("Expected MinRTT %v, got %v", pathStatistic.MinRTT, fetched.MinRTT)
	}
}
