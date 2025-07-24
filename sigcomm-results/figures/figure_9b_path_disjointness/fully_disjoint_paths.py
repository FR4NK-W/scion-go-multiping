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

def find_max_clique_size(adj_matrix):
    """
    Finds the size of the largest clique in a graph using a backtracking algorithm.

    Args:
        adj_matrix (list of lists): The adjacency matrix of the graph.

    Returns:
        int: The size of the maximum clique.
    """
    nodes = list(range(len(adj_matrix)))
    max_size = 0

    def backtrack(potential_clique, candidates):
        nonlocal max_size
        max_size = max(max_size, len(potential_clique))
        
        for i in range(len(candidates)):
            v = candidates[i]
            # Pruning: Only consider candidates that can extend the clique
            is_clique_member = True
            for node_in_clique in potential_clique:
                if not adj_matrix[v][node_in_clique]:
                    is_clique_member = False
                    break
            
            if is_clique_member:
                # New candidates are remaining nodes connected to v
                new_candidates = [c for c in candidates[i+1:] if adj_matrix[v][c]]
                backtrack(potential_clique + [v], new_candidates)

    backtrack([], nodes)
    return max_size

def analyze_mutually_disjoint_sets(paths):
    """
    Calculates the size of the largest set of mutually disjoint paths for each AS pair.

    Args:
        paths (list): A list of all paths.

    Returns:
        list: A list of sizes, where each size is the max number of mutually
              disjoint paths for an AS pair.
    """
    paths_by_as_pair = {}
    for path in paths:
        src_as = path['source_as']
        dest_as = path['destination_as']
        if src_as != dest_as:
            as_pair = tuple(sorted((src_as, dest_as)))
            if as_pair not in paths_by_as_pair:
                paths_by_as_pair[as_pair] = []
            interfaces = extract_interfaces(path['hops'])
            paths_by_as_pair[as_pair].append({'interfaces': interfaces})

    max_set_sizes = []
    for as_pair, path_list in paths_by_as_pair.items():
        num_paths = len(path_list)
        if num_paths < 2:
            max_set_sizes.append(num_paths)
            continue
        
        # Build the adjacency matrix for the disjointness graph
        adj_matrix = [[0] * num_paths for _ in range(num_paths)]
        for i in range(num_paths):
            for j in range(i + 1, num_paths):
                disjointness = calculate_disjointness(
                    path_list[i]['interfaces'], path_list[j]['interfaces']
                )
                if disjointness == 1.0:
                    adj_matrix[i][j] = adj_matrix[j][i] = 1
        
        # Find the size of the largest set of mutually disjoint paths (max clique)
        max_size = find_max_clique_size(adj_matrix)
        max_set_sizes.append(max_size)
        
    return max_set_sizes

def plot_cdf(data, xlabel, ylabel, output_filename):
    """
    Plots and saves a Cumulative Distribution Function (CDF) of the given data.

    Args:
        data (list): A list of numerical data points.
        xlabel (str): The label for the X-axis.
        ylabel (str): The label for the Y-axis.
        output_filename (str): The path to save the output plot file.
    """
    if not data:
        print(f"No data to plot for '{output_filename}'.")
        return

    sorted_data = np.sort(data)
    cdf = np.arange(1, len(sorted_data) + 1) / len(sorted_data)
    
    plt.rcParams.update({'font.size': 14})
    plt.figure(figsize=(8, 5))
    plt.plot(sorted_data, cdf, linestyle='-', marker='.', color='darkviolet')
    plt.xlabel(xlabel, fontsize=14)
    plt.ylabel(ylabel, fontsize=14)
    plt.tick_params(axis='both', which='major', labelsize=12)
    
    # Set x-axis ticks to be integers if the range is reasonable
    max_val = int(np.max(data))
    if max_val < 20:
        plt.xticks(np.arange(0, max_val + 2, 1))

    plt.ylim(0, 1.01)
    plt.xlim(left=0)
    plt.grid(True, which='both', linestyle='--', linewidth=0.5)
    plt.tight_layout()
    
    plt.savefig(output_filename)
    print(f"CDF plot has been saved to '{output_filename}'")
    plt.close()

if __name__ == "__main__":
    csv_file = "disjointness.csv"
    paths = load_paths_from_csv(csv_file)
    
    if paths:
        max_disjoint_set_sizes = analyze_mutually_disjoint_sets(paths)
        
        plot_cdf(
            data=max_disjoint_set_sizes,
            xlabel="Size of Largest Mutually Disjoint Path Set",
            ylabel="CDF of AS Pairs",
            output_filename="max_mutually_disjoint_paths_cdf.pdf"
        )