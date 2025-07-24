import pandas as pd

def transform_csv(input_file: str, output_file: str):
    """
    Loads path statistics from a CSV, aggregates them by source and destination,
    and saves the transformed data to a new CSV file.

    The script performs the following steps:
    1. Reads the input CSV into a pandas DataFrame.
    2. Cleans the 'src_scion_addr' and 'dst_scion_addr' by keeping only the
       part before the first comma.
    3. Converts all relevant measurement columns to a numeric type.
    4. Groups the data by the cleaned source and destination addresses.
    5. Calculates the mean for all measured values within each group.
    6. Saves the aggregated results into the specified output CSV file.
    """
    try:
        # Step 1: Load the CSV file into a pandas DataFrame
        df = pd.read_csv(input_file)
        print(f"Successfully loaded '{input_file}' with {len(df)} rows.")

        # Step 2: Clean the address columns to remove IP/port information
        # The lambda function splits the string by the comma and takes the first element.
        df['src_scion_addr'] = df['src_scion_addr'].apply(lambda x: x.split(',')[0])
        df['dst_scion_addr'] = df['dst_scion_addr'].apply(lambda x: x.split(',')[0])
        print("Cleaned 'src_scion_addr' and 'dst_scion_addr' columns.")

        # Step 3: Ensure all metric columns are numeric, coercing errors to NaN
        # This is good practice in case some data is malformed.
        numeric_cols = [
            'avg_paths', 'min_paths', 'max_paths',
            'avg_diff_to_max', 'median_diff_to_max',
            'avg_min_hops', 'avg_max_hops'
        ]
        for col in numeric_cols:
            df[col] = pd.to_numeric(df[col], errors='coerce')
        
        # Drop rows where conversion to numeric failed in key columns
        df.dropna(subset=numeric_cols, inplace=True)

        # Step 4: Group by the cleaned addresses and aggregate the data
        # We calculate the mean for each group as shown in the example output.
        print("Grouping data by source/destination and calculating the mean...")
        aggregated_df = df.groupby(['src_scion_addr', 'dst_scion_addr']).mean()

        # The groupby operation moves the grouped columns to the index.
        # .reset_index() turns them back into columns.
        aggregated_df = aggregated_df.reset_index()
        
        # Step 5: Ensure the output columns are in the desired order
        # This also ensures we keep the additional columns as requested.
        output_columns = [
            'src_scion_addr', 'dst_scion_addr', 'avg_paths', 'min_paths',
            'max_paths', 'avg_diff_to_max', 'median_diff_to_max',
            'avg_min_hops', 'avg_max_hops'
        ]
        aggregated_df = aggregated_df[output_columns]

        # Step 6: Save the transformed DataFrame to a new CSV file
        # index=False prevents pandas from writing the DataFrame index as a column.
        aggregated_df.to_csv(output_file, index=False)
        print(f"Transformation complete. Output saved to '{output_file}' with {len(aggregated_df)} rows.")

    except FileNotFoundError:
        print(f"Error: The file '{input_file}' was not found.")
    except Exception as e:
        print(f"An unexpected error occurred: {e}")

# --- Main execution ---
if __name__ == "__main__":
    # Define your input and output file names here
    input_csv_file = 'path_stats_clean_median_avg.csv'  # Your source file
    output_csv_file = 'path_stats_clean.csv' # The destination file

    
    # Run the transformation
    transform_csv(input_csv_file, output_csv_file)