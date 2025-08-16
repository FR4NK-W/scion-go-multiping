# \#908 - SIGCOMM '25: Scaling SCIERA: A Journey Through the Deployment of a Next-Generation Network

## Setup instructions to recreate plots

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

## Data Processing and Importing
We provide a tool called [postgres-importer](./tools/original-measurement-import/posgres-importer/main.go) that we used to process the data and aggregate the measurement data from multiple measurement points into a single postgres database. As extension for postgres we deployed a `timescaledb` that provides extended functionality to evaluate time series. Our postgres and importer deployment can be found [here](./tools/original-measurement-import/postgres/docker-compose.yaml).


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
We provide a compressed dump of the aggregated postgres database that can be directly imported. It can be downloaded from [here](https://polybox.ethz.ch/index.php/s/9DMrYiXBwCT4KBH).
For a permanent reference of the dataset, refer to [doi.org/10.5281/zenodo.15672256](https://doi.org/10.5281/zenodo.15672256).

## Processing the Dataset
A mostly automated ready-to-use analysis environment is provided via Docker Compose.
Follow these steps to set up and process the dataset:

1. **Prepare directories**
   Navigate to `sigcomm-results/tools/analysis` and create the necessary directories:

   ```bash
   mkdir -p ./exports
   sudo chown -R 1000:1000 dump exports

   mkdir -p dump
   ```

   > The `exports` directory will store generated CSV files. The `dump` directory will hold the dataset dump file.

2. **Add the dataset**
   Download the `.dump` file from previous section containing the measurements and place it in:

   ```
   dump/selected.dump
   ```

3. **Start the analysis 'stack'**
   Run:

   ```bash
   docker-compose up
   ```

   This will:

   * Start a TimescaleDB container
   * Import the measurements into the database
   * Execute the export SQL scripts to produce `.csv` files

4. **Wait for completion**
   When you see:

   ```
   All done
   ```

   printed by the timescaledb container, stop the containers and run the following **outside** the container to copy the results:

   ```bash
   ./copy_exports.sh
   ```

All exports are now in their respective figure folders.

## Recreating Figures
All figures can be generated by running `bash generate-plots.sh` in the `figures` directory.

In addition to the above scripts, we provide detailed manual instructions for each figure on how to extract the data out of the postgres database and generate the respective plot. 
- [Figure 5: Ping Latency](./figures/figure_5_ping_latency/Readme.md)
- [Figure 6: RTT CDF](./figures/figure_6_rtt_cdf/Readme.md)
- [Figure 7: RTT diff over time](./figures/figure_7_rtt_diff_over_time/Readme.md)
- [Figure 8: Path Statistics Heatmap](./figures/figure_8_9_heatmap/Readme.md)
- [Figure 9: Median Diff Heatmap](./figures/figure_8_9_heatmap/Readme.md)
- [Figure 10a: Path Stretch](./figures/figure_10a_path_stretch/Readme.md)
- [Figure 10b: Path Disjointness](./figures/figure_10b_path_disjointness/Readme.md)
- [Figure 10c: Link Failures](./figures/figure_10c_link_failures/Readme.md)
