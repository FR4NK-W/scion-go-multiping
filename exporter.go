package main

import (
	"time"
)

type PingResult struct {
	SrcSCIONAddr    string     // SCION src
	DstSCIONAddr    string     // SCION dst
	Success         bool       // SuccessfulPings > 0
	RTT             float64    // min rtt across path probed
	RTT2            float64    // min rtt across path probed
	RTT3            float64    // min rtt across path probed
	Fingerprint     string     // Fingerprint of the path with the min rtt
	Fingerprint2    string     // Fingerprint of the path with the min rtt
	Fingerprint3    string     // Fingerprint of the path with the min rtt
	PingTime        *time.Time // time ping result was stored
	SuccessfulPings int        // Ping replies count
	MaxPings        int        // Sent ping count
}

type IPPingResult struct {
	SrcAddr      string
	DstAddr      string
	Success      bool       // SuccessfulPings > 0
	RTT          float64    // min rtt across path probed
	PingTime     *time.Time // time ping result was stored
	SrcSCIONAddr string     // SCION src for mapping
	DstSCIONAddr string     // SCION dst for mapping
}

type PathStatistics struct {
	SrcSCIONAddr   string     // SCION src
	DstSCIONAddr   string     // SCION dst
	Paths          string     // interface description of the AvailablePaths, comma separated
	Fingerprints   string     // path fingerprints, comma separated
	Success        bool       // successCount > 0
	MinRTT         float64    // min rtt across all paths
	MaxRTT         float64    // max rtt across all paths
	MinHops        int        // min # of hops across all paths
	MaxHops        int        // max # of hops across all paths
	LookupTime     *time.Time // time ping results were stored
	ActivePaths    int        // # of active paths (got echo reply)
	ProbedPaths    int        // # of probed paths (sent echo request)
	AvailablePaths int        // # of known paths
}

type PathMeasurement struct {
	SrcSCIONAddr string // SCION src
	DstSCIONAddr string // SCION dst
	Path         string // interface description of the path, comma separated
	Fingerprint  string
	LookupTime   *time.Time // time ping results were stored
	Success      bool
	RTT          float64
	Hops         int
}

type DataExporter interface {
	InitDaily() error
	Close() error
	WritePingResult(PingResult) error
	WriteIPPingResult(IPPingResult) error
	WritePathStatistic(PathStatistics) error
	WritePathMeasurement(PathMeasurement) error
}
