import pandas as pd
overwrite = {
    "71-2:0:3e": "134.75.253.186",
    "71-2:0:3f": "134.75.254.171",
    "71-2:0:5c": "200.129.206.243",
    "71-2:0:48": "145.40.89.243",
}
def merge_and_aggregate_pings():
    """
    Loads and filters SCION and IP ping data by valid hours, merges them,
    aggregates the hourly results into a final average, and saves the output.
    Intermediate dataframes are saved for debugging.
    """
    try:
        # --- 1. Load all necessary CSV files ---
        # The 'hour' columns are parsed as datetime objects for accurate matching.
        scion_pings = pd.read_csv("input/scion_pings_cdf_hour.csv", parse_dates=["hour"])
        ip_pings = pd.read_csv("input/ip_pings_hour_cdf.csv", parse_dates=["hour"])
        address_mapping = pd.read_csv("input/addressmapping.csv")
        valid_hours = pd.read_csv("input/ip_pings_valid_hours.csv", parse_dates=["hour"])
        
    except FileNotFoundError as e:
        print(f"Error: Could not find a required input file. {e}")
        return

    print("Successfully loaded all input CSV files.")

    # --- 2. Create a set of valid (hour, src_addr) combinations for efficient filtering ---
    valid_combinations = set(zip(valid_hours['hour'], valid_hours['src_addr']))
    print(f"Created a set of {len(valid_combinations)} valid hour-source IP combinations.")

    # --- 3. Filter IP pings DataFrame ---
    # Keep only the rows where the (hour, src_addr) tuple exists in our valid set.
    ip_pings_filtered = ip_pings[ip_pings.apply(lambda row: (row['hour'], row['src_addr']) in valid_combinations, axis=1)].copy()
    
    # Save the intermediate filtered IP pings DataFrame for debugging.
    ip_pings_filtered.to_csv("gen/ip_pings_filtered_debug.csv", index=False)
    print(f"Filtered IP pings. Kept {len(ip_pings_filtered)} of {len(ip_pings)} rows.")
    print("Saved intermediate data to gen/ip_pings_filtered_debug.csv")

    # --- 4. Filter SCION pings DataFrame ---
    # First, map each 'src_scion_addr' to its corresponding 'src_ip_addr'.
    scion_pings_with_src_ip = scion_pings.merge(
        address_mapping[['src_scion_addr', 'src_ip_addr']],
        on='src_scion_addr',
        how='left'
    )
    # Save this initial merge for debugging.
    scion_pings_with_src_ip.to_csv("gen/scion_pings_with_src_ip_debug.csv", index=False)
    print("Saved intermediate SCION pings with mapped source IP to gen/scion_pings_with_src_ip_debug.csv")

    # Now, filter this newly created DataFrame using the valid combinations.
    scion_pings_filtered = scion_pings_with_src_ip[scion_pings_with_src_ip.apply(lambda row: (row['hour'], row['src_ip_addr']) in valid_combinations, axis=1)].copy()


    # --- 5. Perform the hourly merge using the filtered data ---
    merged_scion_filtered = scion_pings_filtered.merge(
        address_mapping[['dst_scion_addr', 'dst_ip_addr']],
        on='dst_scion_addr',
        how='left'
    )
        # Save the filtered SCION pings DataFrame for debugging.
    merged_scion_filtered.to_csv("gen/scion_pings_filtered_debug.csv", index=False)
    print(f"Filtered SCION pings. Kept {len(merged_scion_filtered)} of {len(scion_pings)} rows.")
    print("Saved intermediate data to gen/scion_pings_filtered_debug.csv")

    # Rename columns for clarity before the final merge.
    merged_scion_filtered.rename(columns={'src_ip_addr': 'mapped_src_ip', 'dst_ip_addr': 'mapped_dst_ip', 'avg': 'avg_scion'}, inplace=True)
    ip_pings_filtered.rename(columns={'avg': 'avg_ip'}, inplace=True)

    # Merge the fully mapped SCION data with the filtered IP ping data on hour and addresses.
    final_merged_hourly = merged_scion_filtered.merge(
        ip_pings_filtered,
        left_on=['hour', 'mapped_src_ip', 'mapped_dst_ip'],
        right_on=['hour', 'src_addr', 'dst_addr'],
        how='left'
    )

    # --- 6. Aggregate results and save the final data ---
    
    # Define the columns that identify a unique path.
    grouping_cols = ['src_scion_addr', 'dst_scion_addr', 'mapped_src_ip', 'mapped_dst_ip']

    # Group by the unique path and calculate the mean of the RTTs from all valid hours.
    # This aggregates the data and removes the 'hour' column.
    aggregated_data = final_merged_hourly.groupby(grouping_cols, as_index=False).agg(
        avg_scion=('avg_scion', 'mean'),
        avg_ip=('avg_ip', 'mean')
    )

    # Save the final aggregated data to the output CSV file.
    aggregated_data.to_csv("gen/merged_pings_filtered.csv", index=False)
    print("\nAggregation and merge complete.")
    print(f"Final aggregated data with {len(aggregated_data)} rows saved to gen/merged_pings_filtered.csv")

if __name__ == "__main__":
    merge_and_aggregate_pings()