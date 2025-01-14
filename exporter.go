package main

import (
	"time"

	"gorm.io/gorm"
)

type PingResult struct {
	gorm.Model
	SrcSCIONAddr    string
	DstSCIONAddr    string
	Success         bool
	Latency         float64
	PingTime        time.Time
	SuccessfulPings int
	MaxPings        int
}

type PathStatistics struct {
	gorm.Model
	SrcSCIONAddr string
	DstSCIONAddr string
	Success      bool
	MinLatency   float64
	MaxLatency   float64
	MinHops      float64
	MaxMaxHops   float64
	LookupTime   time.Time
	ActivePaths  int
	MaxPaths     int
}

type DataExporter interface {
	WritePingResult(PingResult) error
	WritePathStatistic(PathStatistics) error
}
