import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
from scipy.ndimage import gaussian_filter1d

def plot_rtt_ratio():
    # Load the CSV files
    scion_df = pd.read_csv("avg_scion_pings_pairs.csv", parse_dates=["minute"])
    ip_df = pd.read_csv("avg_ip_pings_pairs.csv", parse_dates=["minute"])
    
    # Merge datasets on timestamp
    merged_df = pd.merge(scion_df, ip_df, on="minute", suffixes=("_scion", "_ip"))
    
    # Calculate RTT ratio
    merged_df["rtt_ratio"] = merged_df["avg_rtt_scion"] / merged_df["avg_rtt_ip"]
    
    # Apply Gaussian smoothing
    merged_df["smoothed_rtt_ratio"] = gaussian_filter1d(merged_df["rtt_ratio"], sigma=2)
    
    # Plot settings
    plt.figure(figsize=(6, 4.4))
    sns.lineplot(x=merged_df["minute"], y=merged_df["smoothed_rtt_ratio"], label="SCION/IP RTT Ratio", linestyle="-")
    
    # Horizontal line at y=1 for IP reference
    plt.axhline(y=1, color='red', linestyle='--', label="IP Baseline")
    
    # Labels and title
    plt.xlabel("Date", fontsize=14)
    plt.ylabel("SCION/IP RTT Ratio ", fontsize=14)
    # plt.title("SCION/IP RTT Ratio", fontsize=16, fontweight='bold')
    plt.legend()
    plt.grid(True, linestyle="--", alpha=0.6)
    
    # Show only every second day on x-axis
    unique_days = merged_df["minute"].dt.normalize().unique()
    every_third_day = unique_days[::2]
    plt.xticks(every_third_day, rotation=45)

    plt.tight_layout()
    plt.savefig("rtt_ratio_time_scaled.pdf")
    
    # Show plot
    # plt.show()

if __name__ == "__main__":
    plot_rtt_ratio()
