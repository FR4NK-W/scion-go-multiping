import json
import matplotlib.pyplot as plt
import numpy as np

# Function to load paths from the JSON file
def load_paths_from_json(file_path):
    with open(file_path, 'r') as f:
        data = json.load(f)
    return data['paths']

# Function to extract interfaces from the hops of a path
def extract_interfaces(hops):
    interfaces = set()
    for hop in hops:
        interface_id = hop['interface']
        interfaces.add((hop['isd_as'], interface_id))  # Store (AS, interface) tuple
    return interfaces

# Function to calculate the disjointness between two sets of interfaces
def calculate_disjointness(interfaces1, interfaces2):
    # Compute the number of common interfaces
    common_interfaces = interfaces1.intersection(interfaces2)
    total_interfaces = interfaces1.union(interfaces2)
    
    if len(total_interfaces) == 0:
        return 1  # Fully disjoint if there are no common interfaces
    
    # Disjointness is defined as 1 - (number of common interfaces / total interfaces)
    disjointness = 1 - (len(common_interfaces) / len(total_interfaces))
    return disjointness

# Function to compute path disjointness for each inter-AS pair
def compute_path_disjointness(paths):
    path_disjointness_list = []
    
    # Group paths by source-destination AS pairs (only inter-AS paths)
    paths_by_as_pair = {}
    for path in paths:
        src_as = path['source_as']
        dest_as = path['destination_as']
        
        if src_as != dest_as:  # Only include inter-AS paths
            src_dest = (src_as, dest_as)
            if src_dest not in paths_by_as_pair:
                paths_by_as_pair[src_dest] = []
            # Extract interfaces for each path
            interfaces = extract_interfaces(path['hops'])
            paths_by_as_pair[src_dest].append({'path': path, 'interfaces': interfaces})
    
    # Calculate disjointness for each AS pair
    for as_pair, path_list in paths_by_as_pair.items():
        if len(path_list) < 2:
            # If there's only one path between the ASes, we cannot calculate disjointness
            continue
        
        # Compare each pair of paths for disjointness
        #for i in range(len(path_list)):
        #    for j in range(i + 1, len(path_list)):
        #        interfaces1 = path_list[i]['interfaces']
        #        interfaces2 = path_list[j]['interfaces']
                
                # Calculate disjointness between the two paths
        #        disjointness = calculate_disjointness(interfaces1, interfaces2)
        #        if disjointness < 0.5:
        #            print(f"AS Pair: {as_pair}, Disjointness: {disjointness}")
        #        path_disjointness_list.append(disjointness)
        # Calculate disjointness for each AS pair
        
        disjointness_values = []
        
        # Compare each pair of paths for disjointness
        for i in range(len(path_list)):
            for j in range(i + 1, len(path_list)):
                interfaces1 = path_list[i]['interfaces']
                interfaces2 = path_list[j]['interfaces']
                
                # Calculate disjointness between the two paths
                disjointness = calculate_disjointness(interfaces1, interfaces2)
                disjointness_values.append(disjointness)
        
        # Compute the average disjointness for this AS pair
        if disjointness_values:
            avg_disjointness = np.mean(disjointness_values)
            path_disjointness_list.append(avg_disjointness)
    
    return path_disjointness_list

# Function to plot the CDF of path disjointness as a line chart
def plot_cdf(path_disjointness_list):
    # Sort the disjointness values
    sorted_disjointness = np.sort(path_disjointness_list)
    # print(sorted_disjointness)
    
    # Compute CDF values
    cdf = np.arange(1, len(sorted_disjointness) + 1) / len(sorted_disjointness)
    
    # Plot the CDF as a line chart
    plt.rcParams.update({'font.size': 14})
    plt.figure(figsize=(6, 4))
    plt.plot(sorted_disjointness, cdf, linestyle='-', color='blue')
    plt.xlabel('Path Disjointness', fontsize=14)
    plt.ylabel('CDF of AS Pairs', fontsize=14)
    plt.tick_params(axis='both', which='major', labelsize=12)
    plt.xlim(0, 1)
    plt.tight_layout()
    # plt.title('CDF of Path Disjointness between Inter-AS Combinations')
    plt.grid(True)
    plt.savefig("path_disjointness.pdf")
    # plt.show()

if __name__ == "__main__":
    # Step 1: Load the JSON file
    json_file = "combined_paths_with_rtt.json"  # Replace with your file path
    paths = load_paths_from_json(json_file)
    
    # Step 2: Compute path disjointness for all inter-AS paths
    path_disjointness_list = compute_path_disjointness(paths)
    
    # Step 3: Plot the CDF of path disjointness as a line chart
    plot_cdf(path_disjointness_list)
