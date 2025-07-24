import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
from scipy.ndimage import gaussian_filter1d
import numpy as np

def plot_with_weighted_average():
    """
    Loads data, filters based on valid hours, aggregates using a
    weighted average, and plots the RTT ratio.
    """
    # Load the CSV files, parsing the 'hour' column as dates
    scion_df = pd.read_csv("scion_pings_hour.csv", parse_dates=["hour"])
    ip_df = pd.read_csv("ip_pings_hour.csv", parse_dates=["hour"])
    valid_hours_df = pd.read_csv("ip_pings_valid_hours.csv", parse_dates=["hour"])

    # --- Filtering based on valid_hours_df ---
    valid_combinations = set(zip(valid_hours_df['hour'], valid_hours_df['src_addr']))

    # Filter IP DataFrame
    ip_df_filtered = ip_df[ip_df.apply(lambda row: (row['hour'], row['src_addr']) in valid_combinations, axis=1)].copy()

    # Filter SCION DataFrame
    scion_df['src_addr'] = scion_df['src_scion_addr'].str.split(',').str[1].str.split(':').str[0]
    scion_df_filtered = scion_df[scion_df.apply(lambda row: (row['hour'], row['src_addr']) in valid_combinations, axis=1)].copy()
    
    # --- Aggregation Section (Weighted Average) ---
    def weighted_average(df, value_col, weight_col):
        try:
            return np.average(df[value_col], weights=df[weight_col])
        except ZeroDivisionError:
            return np.nan

    # Group by hour and aggregate using weighted average for RTT
    scion_hourly = scion_df_filtered.groupby('hour').apply(
        lambda x: pd.Series({
            'avg_rtt_scion': weighted_average(x, 'avg_rtt', 'prcount'),
            'prcount_scion': x['prcount'].sum()
        })
    ).reset_index()

    ip_hourly = ip_df_filtered.groupby('hour').apply(
        lambda x: pd.Series({
            'avg_rtt_ip': weighted_average(x, 'avg_rtt', 'prcount'),
            'prcount_ip': x['prcount'].sum()
        })
    ).reset_index()

    # Merge the hourly aggregated dataframes
    merged_df_hourly = pd.merge(scion_hourly, ip_hourly, on="hour")
    
    # --- Plotting Section ---
    merged_df_hourly["rtt_ratio"] = merged_df_hourly["avg_rtt_scion"] / merged_df_hourly["avg_rtt_ip"]
    merged_df_hourly.dropna(subset=['rtt_ratio'], inplace=True)
    merged_df_hourly["smoothed_rtt_ratio"] = gaussian_filter1d(merged_df_hourly["rtt_ratio"], sigma=2)
    
    plt.figure(figsize=(6, 4.4))
    sns.lineplot(data=merged_df_hourly, x="hour", y="smoothed_rtt_ratio", label="SCION/IP RTT Ratio (Weighted Avg)", linestyle="-")
    
    plt.axhline(y=1, color='red', linestyle='--', label="IP Baseline")
    plt.xlabel("Date", fontsize=14)
    plt.ylabel("SCION/IP RTT Ratio", fontsize=14)
    plt.legend()
    plt.grid(True, linestyle="--", alpha=0.6)
    
    unique_days = merged_df_hourly["hour"].dt.normalize().unique()
    if len(unique_days) > 0:
        every_second_day = unique_days[::2]
        plt.xticks(ticks=every_second_day, labels=[d.strftime('%Y-%m-%d') for d in every_second_day], rotation=45)

    plt.tight_layout()
    plt.savefig("rtt_ratio_time_scaled_weighted_avg.pdf")
    plt.close() # Close the figure to free memory

def plot_with_regular_average():
    """
    Loads data, filters based on valid hours, aggregates using a
    regular (mean) average, and plots the RTT ratio.
    """
    # Load the CSV files
    scion_df = pd.read_csv("scion_pings_hour.csv", parse_dates=["hour"])
    ip_df = pd.read_csv("ip_pings_hour.csv", parse_dates=["hour"])
    valid_hours_df = pd.read_csv("ip_pings_valid_hours.csv", parse_dates=["hour"])

    # --- Filtering based on valid_hours_df ---
    valid_combinations = set(zip(valid_hours_df['hour'], valid_hours_df['src_addr']))

    # Filter IP DataFrame
    ip_df_filtered = ip_df[ip_df.apply(lambda row: (row['hour'], row['src_addr']) in valid_combinations, axis=1)].copy()

    # Filter SCION DataFrame
    scion_df['src_addr'] = scion_df['src_scion_addr'].str.split(',').str[1].str.split(':').str[0]
    scion_df_filtered = scion_df[scion_df.apply(lambda row: (row['hour'], row['src_addr']) in valid_combinations, axis=1)].copy()
    
    # Write scion_df_filtered and ip_df_filtered to CSV for debugging
    scion_df_filtered.to_csv("scion_filtered_debug.csv", index=False)
    ip_df_filtered.to_csv("ip_filtered_debug.csv", index=False)

    # --- Aggregation Section (Regular Average) ---
    # Group by hour and aggregate using mean for RTT and sum for prcount
    scion_hourly = scion_df_filtered.groupby('hour').agg(
        avg_rtt_scion=('avg_rtt', 'mean'),
        prcount_scion=('prcount', 'sum')
    ).reset_index()

    ip_hourly = ip_df_filtered.groupby('hour').agg(
        avg_rtt_ip=('avg_rtt', 'mean'),
        prcount_ip=('prcount', 'sum')
    ).reset_index()

    # Merge the hourly aggregated dataframes
    merged_df_hourly = pd.merge(scion_hourly, ip_hourly, on="hour")
    
    # --- Plotting Section ---
    merged_df_hourly["rtt_ratio"] = merged_df_hourly["avg_rtt_scion"] / merged_df_hourly["avg_rtt_ip"]
    merged_df_hourly.dropna(subset=['rtt_ratio'], inplace=True)
    merged_df_hourly["smoothed_rtt_ratio"] = gaussian_filter1d(merged_df_hourly["rtt_ratio"], sigma=2)
    
    plt.figure(figsize=(6, 4.4))
    sns.lineplot(data=merged_df_hourly, x="hour", y="smoothed_rtt_ratio", label="SCION/IP RTT Ratio", linestyle="-")
    
    plt.axhline(y=1, color='red', linestyle='--', label="IP Baseline")
    plt.xlabel("Date", fontsize=14)
    plt.ylabel("SCION/IP RTT Ratio", fontsize=14)
    plt.legend()
    plt.grid(True, linestyle="--", alpha=0.6)
    
    unique_days = merged_df_hourly["hour"].dt.normalize().unique()
    if len(unique_days) > 0:
        every_second_day = unique_days[::2]
        plt.xticks(ticks=every_second_day, labels=[d.strftime('%Y-%m-%d') for d in every_second_day], rotation=45)

    plt.tight_layout()
    plt.savefig("rtt_ratio_time_scaled_regular_avg.pdf")
    plt.close() # Close the figure

if __name__ == "__main__":
    # Generate the plot using the weighted average method
    # plot_with_weighted_average()
    
    # Generate the plot using the regular average (mean) method
    plot_with_regular_average()

    print("One plot file has been generated:")
    # print("1. rtt_ratio_time_scaled_weighted_avg.pdf")
    print("2. rtt_ratio_time_scaled_regular_avg.pdf")