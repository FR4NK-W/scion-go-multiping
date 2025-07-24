import csv
import matplotlib.pyplot as plt
import numpy as np
import os

# TODO: ADapt to go over each path in an AS pair, calculate the disjoiintness to the most disjoint path (maximum of all others)
# Plot the CDF 1x with average over those maximums, second with putting all disjointness values into one list

# Function to load and parse paths from the specified CSV file
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

            # Extract the source and destination AS from the SCION addresses
            source_as = src_scion_addr.split(',')[0]
            destination_as = dst_scion_addr.split(',')[0]

            # Process each path string in the 'paths' field
            for path_str in paths_str.split(','):
                if not path_str:
                    continue

                hops = []
                # Split the path into individual hops
                for hop_str in path_str.split('->'):
                    # Split each hop to separate the AS-iSD from the interface number
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

# Function to extract interfaces from the hops of a path
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
        interface_id = hop['interface']
        interfaces.add((hop['isd_as'], interface_id))
    return interfaces

# Function to calculate the disjointness between two sets of interfaces
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
        return 1.0  # Fully disjoint if there are no interfaces to compare
    
    disjointness = 1 - (len(common_interfaces) / len(total_interfaces))
    return disjointness

# Function to compute path disjointness for each inter-AS pair
def compute_path_disjointness(paths):
    """
    Computes the average path disjointness for each pair of source and destination ASes.

    Args:
        paths (list): A list of all paths loaded from the data source.

    Returns:
        list: A list of average disjointness values for each inter-AS pair.
    """
    path_disjointness_list = []
    
    # Group paths by their source-destination AS pair
    paths_by_as_pair = {}
    for path in paths:
        src_as = path['source_as']
        dest_as = path['destination_as']
        
        # We only consider inter-AS paths
        if src_as != dest_as:
            as_pair = tuple(sorted((src_as, dest_as))) # Sort to treat (A,B) and (B,A) as the same pair
            if as_pair not in paths_by_as_pair:
                paths_by_as_pair[as_pair] = []
            
            interfaces = extract_interfaces(path['hops'])
            paths_by_as_pair[as_pair].append({'path': path, 'interfaces': interfaces})

    # Calculate disjointness for each AS pair
    for as_pair, path_list in paths_by_as_pair.items():
        if len(path_list) < 2:
            continue
            
        disjointness_values = []
        
        # Compare each pair of paths to calculate disjointness
        for i in range(len(path_list)):
            for j in range(i + 1, len(path_list)):
                interfaces1 = path_list[i]['interfaces']
                interfaces2 = path_list[j]['interfaces']
                
                disjointness = calculate_disjointness(interfaces1, interfaces2)

                #if disjointness == 1:  # Ensure disjointness is a valid value
                #    print(f"Disjoint paths found: {path_list[i]['path']} and {path_list[j]['path']}")

                disjointness_values.append(disjointness)
                path_disjointness_list.append(disjointness)
        
        #if disjointness_values:
        #    avg_disjointness = np.mean(disjointness_values)
        #    path_disjointness_list.append(avg_disjointness)
            
    return path_disjointness_list

# Function to plot the CDF of path disjointness as a line chart
def plot_cdf(path_disjointness_list):
    """
    Plots a Cumulative Distribution Function (CDF) of the path disjointness values.

    Args:
        path_disjointness_list (list): A list of path disjointness values.
    """
    if not path_disjointness_list:
        print("No disjointness data to plot.")
        return

    sorted_disjointness = np.sort(path_disjointness_list)
    cdf = np.arange(1, len(sorted_disjointness) + 1) / len(sorted_disjointness)
    
    plt.rcParams.update({'font.size': 14})
    plt.figure(figsize=(6, 4))
    plt.plot(sorted_disjointness, cdf, linestyle='-', color='blue')
    plt.xlabel('Path Disjointness', fontsize=14)
    plt.ylabel('CDF of AS Pairs', fontsize=14)
    plt.tick_params(axis='both', which='major', labelsize=12)
    plt.xlim(0, 1)
    plt.grid(True)
    plt.tight_layout()
    
    # Save the plot to a file
    output_filename = "path_disjointness.pdf"
    plt.savefig(output_filename)
    print(f"CDF plot has been saved to '{output_filename}'")
    # plt.show()

if __name__ == "__main__":
    # Step 1: Define the input CSV file and load the paths
    csv_file = "disjointness.csv"
    paths = load_paths_from_csv(csv_file)
    
    if paths:
        # Step 2: Compute path disjointness for all inter-AS paths
        path_disjointness_list = compute_path_disjointness(paths)
        
        # Step 3: Plot the CDF of the path disjointness values
        plot_cdf(path_disjointness_list)