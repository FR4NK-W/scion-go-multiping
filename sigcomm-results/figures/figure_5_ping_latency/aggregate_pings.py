import pandas as pd

overwrite = {
    "71-2:0:3e": "134.75.253.186",
    "71-2:0:3f": "134.75.254.171",
    "71-2:0:5c": "200.129.206.243",
    "71-2:0:48": "145.40.89.243",
}
def aggregate_histograms(valid_combinations_file, scion_histogram_file, ip_histogram_file, 
                         output_scion_file, output_ip_file):
    """
    Aggregates ping histograms based on a file of valid hour-source combinations.

    Args:
        valid_combinations_file (str): Path to the CSV file with valid combinations.
        scion_histogram_file (str): Path to the SCION histogram CSV file.
        ip_histogram_file (str): Path to the IP histogram CSV file.
        output_scion_file (str): Path to save the aggregated SCION histogram.
        output_ip_file (str): Path to save the aggregated IP histogram.
    """

    # Load the valid combinations file and create a set for quick lookups
    try:
        valid_combinations = pd.read_csv(valid_combinations_file)
        valid_set = { (row['hour'], row['src_addr']) for _, row in valid_combinations.iterrows() }
    except FileNotFoundError:
        print(f"Error: The file {valid_combinations_file} was not found.")
        return

    # --- Process the SCION histogram file ---
    try:
        scion_histogram = pd.read_csv(scion_histogram_file)
        aggregated_scion = {}

        for _, row in scion_histogram.iterrows():
            hour = row['hour']
            # Extract the IP address from the src_scion_addr
            src_ip = row['src_scion_addr'].split(',')[1].split(':')[0]
            # Overwrite the src address for those ASes that have overlapping source IPs
            if row['src_scion_addr'].split(',')[0] in overwrite.keys():
                src_ip = overwrite[row['src_scion_addr'].split(',')[0]]
            # Check if the combination of hour and source IP is valid
            if any(src_ip in valid_src_addr for valid_hour, valid_src_addr in valid_set if valid_hour == hour):
                rtt_bucket = row['rtt_bucket']
                if rtt_bucket not in aggregated_scion:
                    aggregated_scion[rtt_bucket] = {'lower_bound': row['lower_bound'], 'ping_count': 0}
                aggregated_scion[rtt_bucket]['ping_count'] += row['ping_count']

        # Convert the aggregated data to a DataFrame and save to CSV
        if aggregated_scion:
            output_scion_df = pd.DataFrame.from_dict(aggregated_scion, orient='index')
            output_scion_df = output_scion_df.reset_index().rename(columns={'index': 'rtt_bucket'})
            output_scion_df = output_scion_df.sort_values(by='rtt_bucket').reset_index(drop=True)
            output_scion_df.to_csv(output_scion_file, index=False)
            print(f"Successfully created aggregated SCION histogram at: {output_scion_file}")
        else:
            print("No valid combinations found for the SCION histogram.")

    except FileNotFoundError:
        print(f"Error: The file {scion_histogram_file} was not found.")
    except Exception as e:
        print(f"An error occurred while processing {scion_histogram_file}: {e}")

    # --- Process the IP histogram file ---
    try:
        ip_histogram = pd.read_csv(ip_histogram_file)
        aggregated_ip = {}

        for _, row in ip_histogram.iterrows():
            hour = row['hour']
            src_addr = row['src_addr']
            
            # Check if the combination of hour and source address is valid
            if (hour, src_addr) in valid_set:
                rtt_bucket = row['rtt_bucket']
                if rtt_bucket not in aggregated_ip:
                    aggregated_ip[rtt_bucket] = {'lower_bound': row['lower_bound'], 'ping_count': 0}
                aggregated_ip[rtt_bucket]['ping_count'] += row['ping_count']

        # Convert the aggregated data to a DataFrame and save to CSV
        if aggregated_ip:
            output_ip_df = pd.DataFrame.from_dict(aggregated_ip, orient='index')
            output_ip_df = output_ip_df.reset_index().rename(columns={'index': 'rtt_bucket'})
            output_ip_df = output_ip_df.sort_values(by='rtt_bucket').reset_index(drop=True)
            output_ip_df.to_csv(output_ip_file, index=False)
            print(f"Successfully created aggregated IP histogram at: {output_ip_file}")
        else:
            print("No valid combinations found for the IP histogram.")

    except FileNotFoundError:
        print(f"Error: The file {ip_histogram_file} was not found.")
    except Exception as e:
        print(f"An error occurred while processing {ip_histogram_file}: {e}")


if __name__ == '__main__':
    # Define your input and output file paths here
    VALID_COMBINATIONS_FILE = 'input/ip_pings_valid_hours.csv'
    SCION_HISTOGRAM_FILE = 'input/new_scion_pings_per_hour.csv'
    IP_HISTOGRAM_FILE = 'input/new_ip_pings_per_hour.csv'
    OUTPUT_SCION_FILE = 'gen/new_scion_pings_per_hour_final.csv'
    OUTPUT_IP_FILE = 'gen/new_ip_pings_per_hour_final.csv'

    # Run the aggregation process
    aggregate_histograms(VALID_COMBINATIONS_FILE, SCION_HISTOGRAM_FILE, IP_HISTOGRAM_FILE, 
                         OUTPUT_SCION_FILE, OUTPUT_IP_FILE)