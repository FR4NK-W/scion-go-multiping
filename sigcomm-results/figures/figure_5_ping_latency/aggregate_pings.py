import pandas as pd
import os

def create_aggregated_histograms(file1_path, file2_path, output1_path, output2_path, ping_threshold=1600, ping_threshold2=7500):
    """
    Aggregates per-minute ping histograms from two input CSV files into two
    separate output files.

    The aggregation for each file only includes data from minutes that are
    present in *both* input files and have a total ping count greater than
    or equal to the specified threshold in *each* file.

    Args:
        file1_path (str): The file path for the first input CSV.
        file2_path (str): The file path for the second input CSV.
        output1_path (str): The file path for the first aggregated output CSV.
        output2_path (str): The file path for the second aggregated output CSV.
        ping_threshold (int, optional): The minimum number of pings required
                                        per minute in each file. Defaults to 2000.
    """
    try:
        # Load the input CSV files into pandas DataFrames
        df1 = pd.read_csv(file1_path)
        df2 = pd.read_csv(file2_path)
    except FileNotFoundError as e:
        print(f"Error: {e}. Please ensure the input files exist.")
        return

    # --- Step 1: Find common minutes that meet the ping count criteria ---
    # Convert the 'minute' column to datetime objects for proper comparison
    df1['minute'] = pd.to_datetime(df1['minute'])
    df2['minute'] = pd.to_datetime(df2['minute'])

    # Get the unique minutes from each DataFrame
    minutes1 = set(df1['minute'].unique())
    minutes2 = set(df2['minute'].unique())

    # Find the initial intersection of minutes (present in both files)
    initial_common_minutes = minutes1.intersection(minutes2)

    if not initial_common_minutes:
        print("No common minutes found between the two files. No output files will be generated.")
        return

    # --- New Logic: Filter minutes based on ping count ---
    # Calculate the total ping count for each minute in both files
    pings_per_minute1 = df1.groupby('minute')['ping_count'].sum()
    pings_per_minute2 = df2.groupby('minute')['ping_count'].sum()

    # Identify minutes that meet the ping threshold in each file
    valid_minutes1 = {minute for minute, count in pings_per_minute1.items() if count >= ping_threshold }
    valid_minutes2 = {minute for minute, count in pings_per_minute2.items() if count >= ping_threshold }

    # Final list of common minutes must exist in both files AND meet the threshold in both
    common_minutes = list(initial_common_minutes.intersection(valid_minutes1).intersection(valid_minutes2))

    # --- Report on filtered minutes ---
    minutes_discarded = len(initial_common_minutes) - len(common_minutes)
    if minutes_discarded > 0:
        print(f"Discarded {minutes_discarded} minute(s) due to having fewer than {ping_threshold} pings in at least one file.")

    if not common_minutes:
        print("No common minutes remaining after applying the ping count threshold. No output files will be generated.")
        return

    # --- Step 2: Process the first file ---
    # Filter the first DataFrame to only include the final common minutes
    df1_filtered = df1[df1['minute'].isin(common_minutes)]

    # Group by the RTT bucket and lower bound, and sum the ping counts
    aggregated_df1 = df1_filtered.groupby(['rtt_bucket', 'lower_bound'])['ping_count'].sum().reset_index()

    # Ensure the columns are in the desired order
    aggregated_df1 = aggregated_df1[['rtt_bucket', 'lower_bound', 'ping_count']]

    # Save the first aggregated DataFrame to its output CSV file
    aggregated_df1.to_csv(output1_path, index=False)
    print(f"Successfully created aggregated file for {os.path.basename(file1_path)} at: {output1_path}")

    # --- Step 3: Process the second file ---
    # Filter the second DataFrame to only include the final common minutes
    df2_filtered = df2[df2['minute'].isin(common_minutes)]

    # Group by the RTT bucket and lower bound, and sum the ping counts
    aggregated_df2 = df2_filtered.groupby(['rtt_bucket', 'lower_bound'])['ping_count'].sum().reset_index()

    # Ensure the columns are in the desired order
    aggregated_df2 = aggregated_df2[['rtt_bucket', 'lower_bound', 'ping_count']]

    # Save the second aggregated DataFrame to its output CSV file
    aggregated_df2.to_csv(output2_path, index=False)
    print(f"Successfully created aggregated file for {os.path.basename(file2_path)} at: {output2_path}")


if __name__ == '__main__':

    # --- Define the file paths for the input and output files ---
    input_file1 = 'ip_hist_per_minute.csv'
    input_file2 = 'scion_hist_per_minute.csv'
    output_file1 = 'ip_hist_per_minute_out.csv'
    output_file2 = 'scion_hist_per_minute_out.csv'

    # --- Run the aggregation function ---
    # The new ping_threshold parameter is used here. You can change 2000 to any other value.
    create_aggregated_histograms(input_file1, input_file2, output_file1, output_file2, ping_threshold=1600, ping_threshold2=7500)

    # --- Display the content of the generated output files for verification ---
    print("\n--- Generated Files Content ---")
    if os.path.exists(output_file1):
        print(f"\nContent of {output_file1}:")
        with open(output_file1, 'r') as f:
            print(f.read())

    if os.path.exists(output_file2):
        print(f"\nContent of {output_file2}:")
        with open(output_file2, 'r') as f:
            print(f.read())