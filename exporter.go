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
	SrcSCIONAddr   string
	DstSCIONAddr   string
	Success        bool
	MinLatency     float64
	MaxLatency     float64
	MinHops        int
	MaxHops        int
	LookupTime     time.Time
	ActivePaths    int
	ProbedPaths    int
	AvailablePaths int
}

type DataExporter interface {
	Init() error
	WritePingResult(PingResult) error
	WritePathStatistic(PathStatistics) error
}
