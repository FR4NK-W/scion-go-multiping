#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXPORTS_DIR="$BASE_DIR/exports"
FIGURES_DIR="$BASE_DIR/../../figures"

# Mapping of SQL filenames to target figure subfolders
declare -A mapping=(
  [path_stretch.csv]="figure_10a_path_stretch"
  [disjointness.csv]="figure_10b_path_disjointness"
  [path_disjointness.csv]="figure_10c_link_failures"
  [ip_pings_hour_cdf.csv]="figure_6_rtt_cdf/input"
  [ip_pings_hour.csv]="figure_7_rtt_diff_over_time/input"
  [ip_pings_valid_hours.csv]="figure_5_ping_latency/input"
  [new_ip_pings_per_hour.csv]="figure_5_ping_latency/input"
  [new_scion_pings_per_hour.csv]="figure_5_ping_latency/input"
  [path_stats_clean_median_avg.csv]="figure_8_9_heatmap"
  [scion_pings_cdf_hour.csv]="figure_6_rtt_cdf/input"
  [scion_pings_hour.csv]="figure_7_rtt_diff_over_time/input"
)

echo "Copying CSV exports to figure folders..."
for csv_file in "$EXPORTS_DIR"/*.csv; do
  csv_name="$(basename "$csv_file" .csv)"

  # Resolve mapping
  target_figure="${mapping[${csv_name}.csv]:-}"
  if [[ -z "$target_figure" ]]; then
    echo "WARNING: No mapping for ${csv_name}.csv, skipping."
    continue
  fi

  dest_dir="$FIGURES_DIR/$target_figure"
  mkdir -p "$dest_dir"

  echo " -> $csv_file → $dest_dir/"
  cp "$csv_file" "$dest_dir/"
done

echo "All done."
