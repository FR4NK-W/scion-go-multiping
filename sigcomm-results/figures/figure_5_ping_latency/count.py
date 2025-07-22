import pandas as pd
import os

def count_total_pings(file_paths):
    """
    Reads one or more aggregated ping histogram CSV files and calculates
    the total number of pings in each.

    Args:
        file_paths (list): A list of file paths for the CSV files to process.
    """
    print("--- Calculating Total Pings ---")
    
    total_pings_all_files = 0

    for file_path in file_paths:
        try:
            # Check if the file exists before trying to read it
            if not os.path.exists(file_path):
                print(f"Warning: File not found at '{file_path}'. Skipping.")
                continue

            # Load the CSV file into a pandas DataFrame
            df = pd.read_csv(file_path)

            # Check if the 'ping_count' column exists
            if 'ping_count' not in df.columns:
                print(f"Warning: '{file_path}' does not contain a 'ping_count' column. Skipping.")
                continue

            # Calculate the sum of the 'ping_count' column
            total_pings = df['ping_count'].sum()
            
            # Add to the grand total
            total_pings_all_files += total_pings

            # Print the result for the current file, formatted with commas
            print(f"Total pings in '{os.path.basename(file_path)}': {total_pings:,.0f}")

        except Exception as e:
            print(f"An error occurred while processing {file_path}: {e}")
    
    # Print the grand total from all processed files
    if len(file_paths) > 1:
        print("---------------------------------")
        print(f"Combined total for all files: {total_pings_all_files:,.0f}")


if __name__ == '__main__':

    # --- Define the list of files to be counted ---
    # These should be the paths to your output files from the previous step.
    files_to_count = ['new_scion_pings_per_hour_final.csv', 'new_ip_pings_per_hour_final.csv']

    # --- Run the counting function ---
    count_total_pings(files_to_count)