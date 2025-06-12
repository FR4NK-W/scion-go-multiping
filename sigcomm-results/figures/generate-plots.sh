# !/bin/bash

# This script generates the plots for the SIGCOMM results
# Usage: ./generate-plots.sh
# Ensure the script is run from the correct directory

set -e

# Generate the plot for Figure 5: Ping Latency

cd figure_5_ping_latency
python3 plot_histogram.py
echo "Figure 5: Ping Latency plot generated to figure_5_ping_latency/sciera_hist_norm_cdf.pdf."
cd ..

# Generate the plot for Figure 6: RTT CDF
cd figure_6_rtt_cdf
python3 plot.py
echo "Figure 6: RTT CDF plot generated to figure_6_rtt_cdf/rtt_ratio_cdf.pdf."
cd ..

# Generate the plot for Figure 7: RTT CDF Diff over Time
cd figure_7_rtt_diff_over_time
python3 plot.py
echo "Figure 7: RTT CDF Diff over time plot generated to figure_7_rtt_diff_over_time/rtt_ratio_time_scaled.pdf."
cd ..

# Generate the plot for Figure 8: Path Statistics Heatmap
cd figure_8_heatmap
bash plot.sh
echo "Figure 8: Path Statistics Heatmap plot generated to figure_8_heatmap/heatmap_manual.pdf."
cd ..

# Generate the plot for Figure 9: Path Statistics Heatmap
cd figure_9a_path_stretch
python3 path_stretch.py
echo "Figure 9a: Path Stretch plot generated to figure_9a_path_stretch/path_stretch.pdf."
cd ..

# Generate the plot for Figure 9: Path Statistics Heatmap
cd figure_9b_path_disjointness
python3 path_disjointness.py
echo "Figure 9b: Path Disjointness plot generated to figure_9b_path_disjointness/path_disjointness.pdf."
cd ..

# Generate the plot for Figure 9: Path Statistics Heatmap
cd figure_9c_link_failures
python3 link_failures.py
echo "Figure 9c: Link Failures plot generated to figure_9c_link_failures/link_failures_as_connectivity.pdf."
cd ..


