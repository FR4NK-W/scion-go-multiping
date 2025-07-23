import csv
import json
import random
import sys
import matplotlib.pyplot as plt
import numpy as np

# Function to load and parse paths from a CSV file
def load_paths_from_csv(file_path):
    """
    Loads data from a CSV file and extracts paths and links.

    The CSV file must have 'src_scion_addr', 'dst_scion_addr', and 'paths' columns.
    It parses these columns to create a data structure compatible with the simulation functions.

    Args:
        file_path (str): The path to the input CSV file.

    Returns:
        list: A list of dictionaries, where each dictionary represents a single path.
              Returns an empty list if the file is empty or headers are missing.
    """
    all_paths = []
    try:
        with open(file_path, 'r', newline='') as f:
            reader = csv.DictReader(f)
            # Check for required headers
            required_headers = ['src_scion_addr', 'dst_scion_addr', 'paths']
            if not all(header in reader.fieldnames for header in required_headers):
                print(f"Error: CSV file '{file_path}' is missing one or more required headers: {required_headers}", file=sys.stderr)
                sys.exit(1)

            for row in reader:
                # Extract source and destination AS, discarding IP and port
                source_as = row['src_scion_addr'].split(',')[0]
                destination_as = row['dst_scion_addr'].split(',')[0]
                
                # The 'paths' column contains multiple comma-separated path strings
                path_strings = row['paths'].split(',')
                
                for path_str in path_strings:
                    if not path_str:
                        continue
                    
                    path_dict = {
                        'source_as': source_as,
                        'destination_as': destination_as,
                        'hops': []
                    }
                    
                    # Each path is composed of hops separated by '->'
                    hop_strings = path_str.split('->')
                    
                    for hop_str in hop_strings:
                        # Each hop is an AS-interface pair separated by '#'
                        try:
                            isd_as, interface = hop_str.split('#')
                            hop_dict = {
                                'isd_as': isd_as,
                                'interface': interface
                            }
                            path_dict['hops'].append(hop_dict)
                        except ValueError:
                            print(f"Warning: Skipping malformed hop '{hop_str}' in row: {row}", file=sys.stderr)
                            continue
                            
                    if path_dict['hops']:
                        all_paths.append(path_dict)

    except FileNotFoundError:
        print(f"Error: The file '{file_path}' was not found.", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"An unexpected error occurred: {e}", file=sys.stderr)
        sys.exit(1)
        
    return all_paths

def keep_shortest_paths(paths):
    """
    Filters the list of paths to keep only the shortest path for each
    source-destination AS pair based on the number of hops.

    Args:
        paths (list): A list of path dictionaries.

    Returns:
        list: A new list containing only the shortest path for each unique
              (source_as, destination_as) pair.
    """
    shortest_paths_map = {}
    
    # Group paths by source and destination AS
    for path in paths:
        src_as = path['source_as']
        dest_as = path['destination_as']
        path_key = (src_as, dest_as)
        
        # Calculate the length of the current path by number of hops
        path_length = len(path['hops'])
        
        # If we haven't seen this AS pair before, or if the current path is shorter
        # than the one we've stored, update the map.
        if path_key not in shortest_paths_map or path_length < len(shortest_paths_map[path_key]['hops']):
            shortest_paths_map[path_key] = path
            
    # The values of the dictionary are the shortest paths
    return list(shortest_paths_map.values())


# Function to extract all unique links (AS, interface pairs) from all paths
def extract_links(paths):
    links = set()
    for path in paths:
        hops = path['hops']
        for i in range(len(hops) - 1):
            # Links are between consecutive hops
            source_hop = hops[i]
            dest_hop = hops[i + 1]

            src_as = source_hop['isd_as']
            dest_as = dest_hop['isd_as']
            if src_as == dest_as:
                continue  # Only consider inter-AS paths

            link = ((source_hop['isd_as'], source_hop['interface']),
                    (dest_hop['isd_as'], dest_hop['interface']))
            links.add(link)
    return list(links)  # Return a list to allow for random removal

# Function to simulate link failures and calculate AS path availability to core ASes
def simulate_link_failures_to_core_ases(paths, core_ases, num_trials=100):
    # Extract all unique links
    all_links = extract_links(paths)
    total_links = len(all_links)
    
    # Initialize simulation results
    simulation_results = []
    
    # Initialize a set of removed links
    removed_links = set()
    
    for trial in range(num_trials + 1):  # 0 to 100% of links removed
        # Calculate how many links to remove for this trial
        removed_links_count = trial #int(trial / num_trials * total_links)
        
        # Randomly add more links to the removed_links set
        if removed_links_count > len(removed_links):
            additional_links = set(random.sample([link for link in all_links if link not in removed_links], 
                                                removed_links_count - len(removed_links)))
            removed_links.update(additional_links)
        
        # Check how many ASes still have at least one available path to any core AS
        as_availability_to_core = {}
        for path in paths:
            src_as = path['source_as']
            dest_as = path['destination_as']
            
            # Only consider paths from non-core ASes to core ASes
            if dest_as in core_ases:
                # Check if any link in this path is removed
                path_broken = False
                for i in range(len(path['hops']) - 1):
                    source_hop = path['hops'][i]
                    dest_hop = path['hops'][i + 1]
                    link = ((source_hop['isd_as'], source_hop['interface']),
                            (dest_hop['isd_as'], dest_hop['interface']))
                    
                    if link in removed_links:
                        path_broken = True
                        break
                
                # If the path is not broken, mark the source AS as having at least one path to a core AS
                if not path_broken:
                    as_availability_to_core[src_as] = True
        
        # Calculate the percentage of ASes that still have a path to a core AS
        num_as_with_path_to_core = len(as_availability_to_core)
        total_non_core_as = len(set(path['source_as'] for path in paths if path['source_as'] not in core_ases))
        if total_non_core_as == 0:
            percentage_as_with_path_to_core = 0
        else:
            percentage_as_with_path_to_core = num_as_with_path_to_core / total_non_core_as * 100
        
        # Store the percentage of links removed and percentage of ASes with paths to core ASes
        simulation_results.append({
            'percentage_links_removed': (trial / total_links) * 100 if total_links > 0 else 0,
            'percentage_as_with_path_to_core': percentage_as_with_path_to_core
        })
    
    return simulation_results

# Function to simulate link failures and calculate AS path availability
def simulate_link_failures(paths, num_trials=80):
    # Extract all unique links
    all_links = extract_links(paths)
    total_links = len(all_links)
    
    # Initialize simulation results
    simulation_results = []
    removed_links = set()
    for trial in range(num_trials + 1):  # 0 to 100% of links removed
        removed_links_count = trial
        
        # Randomly add more links to the removed_links set
        if removed_links_count > len(removed_links):
            links_to_sample_from = [link for link in all_links if link not in removed_links]
            num_to_add = removed_links_count - len(removed_links)
            if num_to_add > len(links_to_sample_from):
                num_to_add = len(links_to_sample_from)
            
            additional_links = set(random.sample(links_to_sample_from, num_to_add))
            removed_links.update(additional_links)

        # Check how many AS pairs still have at least one available path
        as_availability = {}
        for path in paths:
            src_as = path['source_as']
            dest_as = path['destination_as']
            if src_as == dest_as:
                continue  # Only consider inter-AS paths
            
            # Check if any link in this path is removed
            path_broken = False
            for i in range(len(path['hops']) - 1):
                source_hop = path['hops'][i]
                dest_hop = path['hops'][i + 1]
                link = ((source_hop['isd_as'], source_hop['interface']),
                        (dest_hop['isd_as'], dest_hop['interface']))
                
                if link in removed_links:
                    path_broken = True
                    break
            
            # If the path is not broken, mark the AS pair as having at least one path
            if not path_broken:
                as_availability[(src_as, dest_as)] = True
        
        # Calculate the percentage of AS pairs that still have at least one path
        num_as_pairs_with_path = len(as_availability)
        total_as_pairs = len(set((path['source_as'], path['destination_as']) for path in paths if path['source_as'] != path['destination_as']))
        
        if total_as_pairs == 0:
            percentage_as_with_path = 0
        else:
            percentage_as_with_path = num_as_pairs_with_path / total_as_pairs * 100

        # Store the percentage of links removed and percentage of AS pairs with paths
        simulation_results.append({
            'percentage_links_removed': (trial / total_links) * 100 if total_links > 0 else 0,
            'percentage_as_with_path': percentage_as_with_path
        })
    
    return simulation_results

# Function to save the simulation results to a JSON file
def save_simulation_results(simulation_results, output_file):
    with open(output_file, 'w') as f:
        json.dump(simulation_results, f, indent=4)
    print(f"Simulation results saved to {output_file}")

# Function to plot the simulation results
def plot_simulation_results(simulation_results):
    # Extract data for plotting
    percentages_links_removed = [res['percentage_links_removed'] for res in simulation_results]
    percentages_as_with_path = [res['percentage_as_with_path'] for res in simulation_results]
    
    # Plot the results
    plt.figure(figsize=(8, 6))
    plt.plot(percentages_links_removed, percentages_as_with_path, linestyle='-', marker='o', color='blue')
    plt.xlabel('Percentage of Failed Links (%)')
    plt.ylabel('Percentage of AS Pairs with at Least One Path (%)')
    plt.title('Impact of Link Failures on Path Availability')
    plt.grid(True)
    plt.show()

if __name__ == "__main__":
    # Step 1: Specify the input CSV file
    # Replace "your_file.csv" with the actual path to your CSV file.
    csv_file = "path_disjointness.csv"
    
    # The core_ases list can be defined here if needed for the core ASes simulation
    core_ases = ["71-20965", "71-2:0:35", "71-2:0:3d"]
    
    # --- Simulation on ALL available paths ---
    print("--- Running simulations on ALL paths ---")
    for i in range(1, 101):
        # Load fresh data for each simulation run
        all_paths = load_paths_from_csv(csv_file)
        if not all_paths:
            print("No paths were loaded from the CSV file. Exiting.", file=sys.stderr)
            sys.exit(1)
            
        # Simulate link failures
        simulation_results = simulate_link_failures(all_paths, num_trials=80)
        
        # Save simulation results
        # Ensure the 'data' directory exists before running.
        output_file = f"data/simulation_results_{i}.json"
        save_simulation_results(simulation_results, output_file)
    print("Completed 100 simulation runs for ALL paths.")

    # --- Simulation on SHORTEST paths only ---
    print("\n--- Running simulations on SHORTEST paths only ---")
    # Load the full path set once
    all_paths = load_paths_from_csv(csv_file)
    if not all_paths:
        print("No paths were loaded from the CSV file. Exiting.", file=sys.stderr)
        sys.exit(1)

    # Filter to keep only the shortest path for each AS pair
    shortest_paths = keep_shortest_paths(all_paths)
    print(f"Filtered to {len(shortest_paths)} shortest paths from {len(all_paths)} total paths.")

    for i in range(1, 101):
        # The shortest_paths set is reused for each simulation run in this block
        # Step 2: Simulate link failures on the filtered (shortest) paths
        simulation_results = simulate_link_failures(shortest_paths, num_trials=80)
        
        # Step 3: Save simulation results to a different file
        output_file = f"data/simulation_results_{i}_single.json"
        save_simulation_results(simulation_results, output_file)

    print("Completed 100 simulation runs for SHORTEST paths.")