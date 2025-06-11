# Reaching Escape Velocity for Layer-3 Innovation: Fueling Deployment of a Next-generation Internet Architecture - Setup instructions to recreate plots

In this folder we provide all the required information to recreate our plots contained in the submission. We explain how we collected the data, which transformation steps changed the data and how we aggregated the data.

## Data collection
We use the tool called [scion-go-multiping](https://github.com/FR4NK-W/scion-go-multiping) at commit `224628ab13e0ade24404e1c7681a1ff71a256fad` to send pings via SCION and BGP/IP between a subset of nodes in the SCIERA AS. All SCION and IP destinations can be obtained by the `remotes.json` in the project. The tool can be built with the proper Go version by running `go build`.

The `scion-go-multiping` generates one sqlite file per day which contains the following tables:

```go
type PingResult struct {
	SrcSCIONAddr    string    // SCION src
	DstSCIONAddr    string    // SCION dst
	Success         bool      // SuccessfulPings > 0
	RTT             float64   // min rtt across path probed
	Fingerprint     string    // Fingerprint of the path with the min rtt
	PingTime        time.Time // time ping result was stored
	SuccessfulPings int       // Ping replies count
	MaxPings        int       // Sent ping count
}

type IPPingResult struct {
	SrcAddr  string
	DstAddr  string
	Success  bool      // SuccessfulPings > 0
	RTT      float64   // min rtt across path probed
	PingTime time.Time // time ping result was stored
}

type PathStatistics struct {
	SrcSCIONAddr   string    // SCION src
	DstSCIONAddr   string    // SCION dst
	Paths          string    // interface description of the AvailablePaths, comma separated
	Fingerprints   string    // path fingerprints, comma separated
	Success        bool      // successCount > 0
	MinRTT         float64   // min rtt across all paths
	MaxRTT         float64   // max rtt across all paths
	MinHops        int       // min # of hops across all paths
	MaxHops        int       // max # of hops across all paths
	LookupTime     time.Time // time ping results were stored
	ActivePaths    int       // # of active paths (got echo reply)
	ProbedPaths    int       // # of probed paths (sent echo request)
	AvailablePaths int       // # of known paths
}
```

Pings over IP are sent over the path obtained by BGP, pings over SCION are sent over three (mostly disjoint) SCION paths. A single SCION ping entry is generated out of the three pings. Pings are generated every second, path statistics every minute.

## Data Processing amd Importing
We provide a tool called [postgres-importer](./tools/posgres-importer/main.go) that we used to process the data and import it into postgres. As extension for postgres we deployed a `timescaledb` that provides extended functionality to evaluate time series. Our postgres and importer deployment can be found [here](./tools/postgres/docker-compose.yaml).


## Database preparation
We noticed that some fields in the `gorm` setting did not have the proper postgres timestamp fields and ended up being stored as strings. To overcome this, we ran the following queries after importing all the data:

```sql
UPDATE ping_results
SET ping_time_new = ping_time::TIMESTAMPTZ WHERE ping_time_new IS NULL;

UPDATE ip_ping_results
SET ping_time_new = ping_time::TIMESTAMPTZ WHERE ping_time_new IS NULL;

UPDATE path_statistics
SET lookup_time_new = lookup_time::TIMESTAMPTZ WHERE lookup_time_new IS NULL;
```

## Current Dataset
We provide a compressed postgres database dump that can be imported. It can be downloaded over TODO: Add Link here...

## Recreating Figures
We provide detailed instructions for each figure to extract the data out of the postgres database and generate the respective plot:
- [Figure 5: Ping Latency](./figures/figure_5_ping_latency/Readme.md)
- [Figure 6: RTT CDF](./figures/figure_6_rtt_cdf/Readme.md)
- [Figure 7: RTT diff over time](./figures/figure_7_rtt_diff_over_time/Readme.md)
- [Figure 8: Path Statistics Heatmap](./figures/figure_8_heatmap/Readme.md)
- [Figure 9a: Path Stretch](./figures/figure_9a_path_stretch/Readme.md)
- [Figure 9b: Path Disjointness](./figures/figure_9b_path_disjointness/Readme.md)
- [Figure 9c: Link Failures](./figures/figure_9c_link_failures/Readme.md)