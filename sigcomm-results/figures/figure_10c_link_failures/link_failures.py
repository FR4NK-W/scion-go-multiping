import json
import os
import numpy as np
import matplotlib.pyplot as plt

# Function to load simulation results from a single JSON file
def load_simulation_results(file_path):
    with open(file_path, 'r') as f:
        return json.load(f)

# Function to load and average multiple simulation results
def average_simulation_results(file_paths):
    all_simulations = []

    # Load all simulation results
    for file_path in file_paths:
        results = load_simulation_results(file_path)
        all_simulations.append(results)

    # Extract the percentage of links removed from the first simulation (assuming it's the same for all)
    percentages_links_removed = [res['percentage_links_removed'] for res in all_simulations[0]]

    # Initialize an array to store cumulative percentage of AS pairs with paths
    cumulative_as_with_path = np.zeros(len(percentages_links_removed))

    # Accumulate results from all simulations
    for simulation in all_simulations:
        cumulative_as_with_path += np.array([res['percentage_as_with_path'] for res in simulation])

    # Average the values
    average_as_with_path = cumulative_as_with_path / len(all_simulations)
    # print(average_as_with_path)
    # Return the averaged results
    return percentages_links_removed, average_as_with_path

# Function to plot the averaged simulation results for both sets
def plot_averaged_simulation_results(percentages_links_removed, average_as_with_path_1, average_as_with_path_2):
    # Plot the averaged simulation results for the first set
    plt.figure(figsize=(8, 5.3))
    plt.plot(percentages_links_removed, average_as_with_path_1, linestyle='-', color='blue', label='Multipath')
    
    # Plot the averaged simulation results for the second set
    plt.plot(percentages_links_removed, average_as_with_path_2, linestyle='-', color='red', label='Singlepath')
    
    plt.rcParams.update({'font.size': 16})
    # plt.rcParams["font.family"] = "Times New Roman"
    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42
    plt.xlabel('Fraction of Links Removed (%)',  fontsize=16)
    plt.xlim(0, 100)
    plt.ylabel('AS Pairs with Connectivity (%)',  fontsize=16)
    #plt.title('Impact of Link Failures on AS Connectivity')
    plt.tick_params(axis='both', which='major', labelsize=14)
    plt.tight_layout()
    plt.legend()  # Add legend to distinguish between the two lines
    plt.grid(True)
    # plt.show()
    plt.savefig("link_failures_as_connectivity.pdf")

if __name__ == "__main__":
    # Step 1: List of simulation result files for the first set (Multi-path)
    simulation_files_multipath = []
    for i in range(1, 21):
        simulation_files_multipath.append(f"data/simulation_results_{i}.json")
    
    # List of simulation result files for the second set (Single-path)
    simulation_files_singlepath = []
    for i in range(1, 21):
        simulation_files_singlepath.append(f"data/simulation_results_{i}_single.json")
    
    # Step 2: Load and average simulation results for the first set (Multi-path)
    percentages_links_removed, average_as_with_path_multipath = average_simulation_results(simulation_files_multipath)
    
    # Step 3: Load and average simulation results for the second set (Single-path)
    _, average_as_with_path_singlepath = average_simulation_results(simulation_files_singlepath)
    
    # Step 4: Plot the averaged simulation results for both sets
    plot_averaged_simulation_results(percentages_links_removed, average_as_with_path_multipath, average_as_with_path_singlepath)
