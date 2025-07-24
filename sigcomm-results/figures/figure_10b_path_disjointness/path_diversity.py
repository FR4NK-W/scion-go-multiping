import csv
import matplotlib.pyplot as plt
import numpy as np
import os
import matplotlib.ticker as mticker
from collections import Counter

def load_paths_from_csv(file_path):
    """
    Loads path data from a CSV file.

    The CSV file is expected to have the following columns:
    "src_scion_addr", "dst_scion_addr", "paths"

    Args:
        file_path (str): The path to the CSV file.

    Returns:
        list: A list of dictionaries, where each dictionary represents a single path
              with its source AS, destination AS, and hops. Returns an empty list
              if the file cannot be found.
    """
    if not os.path.exists(file_path):
        print(f"Error: The file '{file_path}' was not found.")
        return []

    all_paths = []
    with open(file_path, 'r', newline='') as f:
        reader = csv.reader(f)
        next(reader)  # Skip the header row
        for row in reader:
            src_scion_addr, dst_scion_addr, paths_str = row
            source_as = src_scion_addr.split(',')[0]
            destination_as = dst_scion_addr.split(',')[0]
            for path_str in paths_str.split(';'):
                if not path_str:
                    continue
                hops = []
                for hop_str in path_str.split('->'):
                    parts = hop_str.split('#')
                    if len(parts) == 2:
                        isd_as, interface_id = parts
                        hops.append({'isd_as': isd_as, 'interface': interface_id})
                if hops:
                    all_paths.append({
                        'source_as': source_as,
                        'destination_as': destination_as,
                        'hops': hops
                    })
    return all_paths

def extract_interfaces(hops):
    """
    Extracts a set of link tuples from a path's hops.

    Args:
        hops (list): A list of hop dictionaries for a path.

    Returns:
        set: A set of tuples representing the links in the path.
    """
    interfaces = set()
    for i in range(len(hops) - 1):
        # Create a frozenset for each hop pair to make the link order-independent
        link = frozenset([
            (hops[i]['isd_as'], hops[i]['interface']), 
            (hops[i+1]['isd_as'], hops[i+1]['interface'])
        ])
        interfaces.add(link)
    return interfaces

def calculate_common_link_diversity(paths):
    """
    Calculates a disjointness ratio for each AS pair.

    The ratio is defined as the number of total unique links divided by the number of
    links that are shared by at least two paths. Duplicate paths are filtered out first.

    Args:
        paths (list): A list of all paths.

    Returns:
        list: A list of disjointness ratios for AS pairs.
    """
    paths_by_as_pair = {}
    for path in paths:
        src_as = path['source_as']
        dest_as = path['destination_as']
        as_pair = tuple(sorted((src_as, dest_as)))
        
        if as_pair not in paths_by_as_pair:
            # Use a set to store unique paths (represented by frozensets of their interfaces)
            paths_by_as_pair[as_pair] = set()
        
        interfaces = extract_interfaces(path['hops'])
        # Add the frozenset of interfaces to the set to automatically handle duplicates
        paths_by_as_pair[as_pair].add(frozenset(interfaces))

    diversity_ratios = []
    
    for as_pair, unique_paths_frozensets in paths_by_as_pair.items():
        # Convert frozensets back to sets for processing
        path_interfaces_list = [set(fs) for fs in unique_paths_frozensets]
        
        num_unique_paths = len(path_interfaces_list)

        if num_unique_paths < 2:
            # If there's 0 or 1 path, no links can be shared, so we can skip to the next pair.
            # Or handle as a case of perfect diversity if needed.
            continue
        
        # --- MODIFIED LOGIC: Find links shared by at least two paths ---
        # 1. Flatten the list of all links from all paths into a single list
        all_links_flat = [link for path_set in path_interfaces_list for link in path_set]
        
        # 2. Count the occurrences of each link
        link_counts = Counter(all_links_flat)
        
        # 3. A link is "common" if it appears in 2 or more paths
        shared_links = {link for link, count in link_counts.items() if count >= 2}
        
        num_shared_links = len(shared_links)
        
        # Calculate the total number of unique links across all paths for the pair
        num_total_unique_links = len(set.union(*path_interfaces_list)) if path_interfaces_list else 0

        print(
            f"AS Pair: {as_pair}, Unique Paths: {num_unique_paths}, "
            f"Shared Links (in >=2 paths): {num_shared_links}, "
            f"Total Unique Links: {num_total_unique_links}"
        )

        if num_total_unique_links - num_shared_links > 0:
            # Ratio of all unique links vs. links that are shared
            ratio = (num_total_unique_links - num_shared_links) / num_total_unique_links
            diversity_ratios.append(ratio)
        else:
            # If no links are shared, the paths are perfectly disjoint.
            # We can assign the number of unique links as the score, representing high diversity.
            # This avoids division by zero.
            # diversity_ratios.append(num_total_unique_links)
            print(f"Warning: AS Pair {as_pair} has no shared links, skipping to avoid division by zero.")
            
    return diversity_ratios

def plot_cdf(data, xlabel, ylabel, output_filename):
    """
    Plots and saves a CDF of the given data.

    Args:
        data (list): A list of numerical data points.
        xlabel (str): The label for the X-axis.
        ylabel (str): The label for the Y-axis.
        output_filename (str): The path to save the output plot file.
    """
    if not data:
        print(f"No data to plot for '{output_filename}'. All pairs may have been excluded.")
        return

    plt.rcParams.update({'font.size': 14})
    plt.figure(figsize=(8, 5))

    sorted_data = np.sort(data)
    yvals = np.arange(1, len(sorted_data) + 1) / len(sorted_data)

    plt.plot(sorted_data, yvals, drawstyle='steps-post', linewidth=2, color='purple')

    plt.gca().yaxis.set_major_formatter(mticker.PercentFormatter(xmax=1.0))

    plt.xlabel(xlabel, fontsize=14)
    plt.ylabel(ylabel, fontsize=14)
    plt.tick_params(axis='both', which='major', labelsize=12)
    
    plt.xlim(left=0)
    plt.ylim(bottom=0, top=1)
    plt.grid(True, which='both', linestyle='--', linewidth=0.5)
    plt.tight_layout()
    
    plt.savefig(output_filename)
    print(f"CDF plot has been saved to '{output_filename}'")
    plt.close()

if __name__ == "__main__":
    csv_file = "disjointness.csv" 
    paths = load_paths_from_csv(csv_file)
    
    if paths:
        # Calculate the path diversity based on shared links
        path_diversity_ratios = calculate_common_link_diversity(paths)
        
        # Plot the single CDF for the new path diversity metric
        plot_cdf(
            data=path_diversity_ratios,
            xlabel="Path Diversity (Number of unique links / Number of total Links)",
            ylabel="CDF (Percentage of AS Pairs)",
            output_filename="cdf_link_disjointness.pdf"
        )