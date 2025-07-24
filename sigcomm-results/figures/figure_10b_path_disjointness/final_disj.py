import csv
import matplotlib.pyplot as plt
import numpy as np
import os

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
            for path_str in paths_str.split(','):
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
    Extracts a set of (AS, interface) tuples from a path's hops.

    Args:
        hops (list): A list of hop dictionaries for a path.

    Returns:
        set: A set of tuples, where each tuple contains the AS and interface ID.
    """
    interfaces = set()
    for hop in hops:
        interfaces.add((hop['isd_as'], hop['interface']))
    return interfaces

def calculate_disjointness(interfaces1, interfaces2):
    """
    Calculates the disjointness between two paths based on their interfaces.
    Disjointness = 1 - (common_interfaces / total_interfaces)

    Args:
        interfaces1 (set): A set of interfaces from the first path.
        interfaces2 (set): A set of interfaces from the second path.

    Returns:
        float: The calculated disjointness value, between 0 and 1.
    """
    common_interfaces = interfaces1.intersection(interfaces2)
    total_interfaces = interfaces1.union(interfaces2)
    if not total_interfaces:
        return 1.0  # Fully disjoint if there are no interfaces
    return 1 - (len(common_interfaces) / len(total_interfaces))

def compute_max_disjointness_metrics(paths):
    """
    Computes disjointness metrics based on the maximum disjointness for each path.

    For each AS pair, it calculates the maximum disjointness of each path relative
    to all other paths in that pair.

    Args:
        paths (list): A list of all paths loaded from the data source.

    Returns:
        tuple: A tuple containing two lists:
               - A list of the average of maximum disjointness values for each AS pair.
               - A list of all individual maximum disjointness values.
    """
    avg_max_disjointness_per_as_pair = []
    all_max_disjointness_values = []
    
    paths_by_as_pair = {}
    for path in paths:
        src_as = path['source_as']
        dest_as = path['destination_as']
        if src_as != dest_as:
            as_pair = tuple(sorted((src_as, dest_as)))
            if as_pair not in paths_by_as_pair:
                paths_by_as_pair[as_pair] = []
            interfaces = extract_interfaces(path['hops'])
            paths_by_as_pair[as_pair].append({'path': path, 'interfaces': interfaces})

    for as_pair, path_list in paths_by_as_pair.items():
        if len(path_list) < 2:
            continue
        
        max_disjointness_for_each_path_in_pair = []
        for i in range(len(path_list)):
            current_path_interfaces = path_list[i]['interfaces']
            disjointness_values_for_current_path = []
            
            for j in range(len(path_list)):
                if i == j:
                    continue
                
                other_path_interfaces = path_list[j]['interfaces']
                disjointness = calculate_disjointness(current_path_interfaces, other_path_interfaces)
                disjointness_values_for_current_path.append(disjointness)
            
            # Add the maximum disjointness found for the current path
            if disjointness_values_for_current_path:
                max_disjointness_for_each_path_in_pair.append(np.max(disjointness_values_for_current_path))
            else:
                max_disjointness_for_each_path_in_pair.append(0) # Or 1.0, depending on desired behavior for single paths

        if max_disjointness_for_each_path_in_pair:
            # print(f"AS Pair: {as_pair}, Max Disjointness: {max_disjointness_for_each_path_in_pair}")
            avg_max_disjointness_per_as_pair.append(np.mean(max_disjointness_for_each_path_in_pair))
            all_max_disjointness_values.extend(max_disjointness_for_each_path_in_pair)
            
    return avg_max_disjointness_per_as_pair, all_max_disjointness_values

def plot_cdf(data, ylabel, output_filename):
    """
    Plots and saves a Cumulative Distribution Function (CDF) of the given data.

    Args:
        data (list): A list of numerical data points.
        ylabel (str): The label for the Y-axis.
        output_filename (str): The path to save the output plot file.
    """
    if not data:
        print(f"No data to plot for '{output_filename}'.")
        return

    sorted_data = np.sort(data)
    cdf = np.arange(1, len(sorted_data) + 1) / len(sorted_data)
    
    plt.rcParams.update({'font.size': 14})
    plt.figure(figsize=(6, 4))
    plt.plot(sorted_data, cdf, linestyle='-', color='blue')
    plt.xlabel('Path Disjointness', fontsize=14)
    plt.ylabel(ylabel, fontsize=14)
    plt.tick_params(axis='both', which='major', labelsize=12)
    plt.xlim(0.6, 1.02)
    # --- FIX ---
    # Explicitly set the Y-axis to the correct range for a CDF (0 to 1)
    plt.ylim(0, 1.01) 
    # -----------
    plt.grid(True)
    plt.tight_layout()
    
    plt.savefig(output_filename)
    print(f"CDF plot has been saved to '{output_filename}'")
    plt.close()

if __name__ == "__main__":
    csv_file = "path_disjointness.csv"
    paths = load_paths_from_csv(csv_file)
    
    if paths:
        avg_disjointness, all_max_disjointness = compute_max_disjointness_metrics(paths)
        
        plot_cdf(
            data=avg_disjointness,
            ylabel='CDF of AS Pairs',
            output_filename="avg_max_disjointness_cdf.pdf"
        )
        
        plot_cdf(
            data=all_max_disjointness,
            ylabel='CDF of Paths',
            output_filename="all_max_disjointness_cdf.pdf"
        )