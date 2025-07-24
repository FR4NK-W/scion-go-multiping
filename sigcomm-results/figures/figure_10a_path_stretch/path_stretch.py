import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

def analyze_path_data(file_path):
    """
    Loads path data from a CSV, counts unique paths, calculates path stretch,
    and plots the CDF of the stretch.

    Args:
        file_path (str): The path to the input CSV file.
    """
    try:
        # Read the CSV data into a pandas DataFrame
        df = pd.read_csv(file_path)
    except FileNotFoundError:
        print(f"Error: The file '{file_path}' was not found.")
        return

    # --- Part 1: Count unique paths for each source/destination combination ---
    #print("--- Unique Path Counts per Source/Destination ---")
    
    # Group by source and destination addresses and count unique fingerprints
    path_counts = df.groupby(['src_scion_addr', 'dst_scion_addr'])['fingerprint'].nunique().reset_index()
    
    # Rename the fingerprint column to reflect the count
    path_counts.rename(columns={'fingerprint': 'unique_path_count'}, inplace=True)

    #print(path_counts)
    #print("\n" + "="*50 + "\n")

    # --- Part 2: Compute path stretch for each source/destination combination ---
    path_stretch_list = []
    
    # Group paths by source-destination pairs
    grouped_paths = df.groupby(['src_scion_addr', 'dst_scion_addr'])
    
    for _, group in grouped_paths:
        # For each src/dst pair, get the RTTs of unique paths
        # We consider each fingerprint as a unique path
        unique_path_rtts = group.drop_duplicates(subset=['fingerprint'])
        
        # We need at least two distinct paths to calculate stretch
        if len(unique_path_rtts) < 2:
            continue
        
        # Sort paths by average RTT to find the fastest and second fastest
        sorted_rtts = unique_path_rtts['avg'].sort_values()
        
        # Get the fastest (lowest RTT) and second fastest paths
        fastest_path_rtt = sorted_rtts.iloc[0]
        second_fastest_path_rtt = sorted_rtts.iloc[1]
        
        # Avoid division by zero
        if fastest_path_rtt > 0:
            stretch = second_fastest_path_rtt / fastest_path_rtt
            path_stretch_list.append(stretch)

    # --- Part 3: Plot the CDF of the computed path stretches ---
    if not path_stretch_list:
        print("Could not compute any path stretches. Are there any source/destination pairs with at least 2 paths?")
        return
        
    #print(f"--- Plotting CDF for {len(path_stretch_list)} Source/Destination Pairs ---")
    plot_stretch_cdf(path_stretch_list)
    #print("CDF plot saved to 'path_stretch_cdf.pdf'")


def plot_stretch_cdf(path_stretch_list):
    """
    Plots the Cumulative Distribution Function (CDF) of path stretch.

    Args:
        path_stretch_list (list): A list of path stretch values.
    """
    # Sort the path stretch values for plotting
    sorted_stretch = np.sort(path_stretch_list)
    
    # Compute CDF values (y-axis)
    cdf = np.arange(1, len(sorted_stretch) + 1) / len(sorted_stretch)
    
    # Create the plot
    plt.figure(figsize=(6, 4))
    plt.rcParams.update({'font.size': 14})
    
    plt.plot(sorted_stretch, cdf, linestyle='-')
    
    plt.xlabel('Latency Inflation', fontsize=14)
    plt.ylabel('CDF of Source/Destination Pairs', fontsize=14)
    plt.tick_params(axis='both', which='major', labelsize=12)
    plt.tight_layout()
    
    # Set plot limits and ticks similar to the example
    plt.xlim(0.5, 3)
    plt.xticks([0.5, 1, 1.5, 2, 2.5, 3])
    plt.grid(True)
    
    # Save the plot to a file
    plt.savefig("latency_inflation.pdf")
    # plt.show() # Uncomment to display the plot directly

if __name__ == "__main__":
    # The script assumes a CSV file named 'path_stretch.csv' is in the same directory.
    # Create a dummy CSV for demonstration if it doesn't exist.
    try:
        pd.read_csv("./path_stretch.csv")
    except FileNotFoundError:
        print("File not found.")

            
    analyze_path_data("./path_stretch.csv")