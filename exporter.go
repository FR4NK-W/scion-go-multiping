package main

import (
	"time"

	"gorm.io/gorm"
)

type PingResult struct {
	gorm.Model
	SrcSCIONAddr    string // SCION src
	DstSCIONAddr    string // SCION dst
	Success         bool // SuccessfulPings > 0
	RTT             float64 // min rtt across path probed
	PingTime        time.Time // time ping result was stored
	SuccessfulPings int // Ping replies count
	MaxPings        int // Sent ping count
}

type PathStatistics struct {
	gorm.Model
	SrcSCIONAddr   string    // SCION src
	DstSCIONAddr   string    // SCION dst
	Success        bool      // successCount > 0
	MinRTT         float64   // min rtt across all paths
	MaxRTT         float64   // min rtt across all paths
	MinHops        int       // min # of hops across all paths
	MaxHops        int       // max # of hops across all paths
	LookupTime     time.Time // time ping results were stored
	ActivePaths    int       // # of active paths (got echo reply)
	ProbedPaths    int       // # of probed paths (sent echo request)
	AvailablePaths int       // # of known paths
}

type DataExporter interface {
	Init() error
	WritePingResult(PingResult) error
	WritePathStatistic(PathStatistics) error
}
