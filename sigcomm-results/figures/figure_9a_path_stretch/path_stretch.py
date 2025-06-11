import json
import matplotlib.pyplot as plt
import numpy as np

# Function to load paths from the JSON file
def load_paths_from_json(file_path):
    with open(file_path, 'r') as f:
        data = json.load(f)
    return data['paths']

# Function to calculate the delay (sum of RTTs) of a path
def calculate_path_delay(hops):
    total_delay = 0
    for hop in hops:
        if hop['round_trip_times']:
            total_delay += sum(hop['round_trip_times']) / len(hop['round_trip_times'])  # Average RTT per hop
    return total_delay

# Function to compute path stretch for each AS pair
def compute_path_stretch(paths):
    path_stretch_list = []
    
    # Group paths by source-destination AS pairs
    paths_by_as_pair = {}
    paths_by_as_pair2 = {}
    for path in paths:
        src_dest = (path['source_as'], path['destination_as'])
        src_dest2 = path['source_as'] + path['destination_as']

        if src_dest not in paths_by_as_pair:
            paths_by_as_pair[src_dest] = []
            paths_by_as_pair2[src_dest2] = []

        # Calculate the total RTT (delay) for each path
        total_rtt = calculate_path_delay(path['hops'])
        paths_by_as_pair[src_dest].append({'path': path, 'rtt': total_rtt})
        paths_by_as_pair2[src_dest2].append({'path': path, 'rtt': total_rtt})
    
    with open("paths_full.json", 'w') as outfile:
        json.dump(paths_by_as_pair2, outfile, indent=4)

    # Calculate path stretch for each AS pair
    for as_pair, path_list in paths_by_as_pair.items():
        # Sort paths by RTT (total delay), not by the number of hops
        path_list.sort(key=lambda p: p['rtt'])
        
        if len(path_list) < 2:
            # If there's only one path between the ASes, we cannot calculate stretch
            continue
        
        # Calculate delay for the shortest path (lowest RTT)
        shortest_path_rtt = path_list[0]['rtt']
        
        # Calculate delay for the second shortest path
        second_shortest_path_rtt = path_list[1]['rtt']
        
        if shortest_path_rtt > 0:  # Avoid division by zero
            stretch = second_shortest_path_rtt / shortest_path_rtt
            print(f"AS Pair: {as_pair}, Stretch: {stretch}, dividing {second_shortest_path_rtt} by {shortest_path_rtt}")
            path_stretch_list.append(stretch)
    
    return path_stretch_list

# Function to plot the CDF of path stretch
def plot_cdf(path_stretch_list):
    # Sort the path stretch values
    sorted_stretch = np.sort(path_stretch_list)
    
    # Compute CDF values
    cdf = np.arange(1, len(sorted_stretch) + 1) / len(sorted_stretch)
    
    # Plot the CDF
    plt.figure(figsize=(6, 4))
    plt.rcParams.update({'font.size': 14})
    plt.plot(sorted_stretch, cdf, linestyle='-')
    plt.xlabel('Path Stretch',  fontsize=14)
    plt.ylabel('CDF of AS Pairs',  fontsize=14)
    plt.tick_params(axis='both', which='major', labelsize=12)
    plt.tight_layout()
    # plt.title('CDF of Path Stretch between AS Combinations')
    plt.xlim(0.5,3)
    plt.xticks([0.5, 1, 1.5, 2, 2.5, 3])
    plt.grid(True)
    plt.savefig("path_stretch.pdf")
    plt.show()

if __name__ == "__main__":
    # Step 1: Load the JSON file
    json_file = "combined_paths_with_rtt.json"  # Replace with your file path
    paths = load_paths_from_json(json_file)
    
    # Step 2: Compute path stretch for all inter-AS paths
    path_stretch_list = compute_path_stretch(paths)
    
    # Step 3: Plot the CDF of path stretch
    plot_cdf(path_stretch_list)
