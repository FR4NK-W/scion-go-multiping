import csv
import itertools
import argparse

def process_scion_paths(input_file, output_file):
    """
    Reads a CSV file with SCION source addresses, destination addresses, and their corresponding paths.
    It then generates all possible path combinations for each source-destination pair and writes
    the results to a specified output CSV file.

    The script expects the input CSV to have the following columns:
    - src_scion_addr
    - dst_scion_addr
    - paths (a comma-separated string of SCION paths)

    The output CSV file will have the following columns:
    - src_scion_addr
    - dst_scion_addr
    - path_1
    - path_2

    Args:
        input_file (str): The path to the input CSV file.
        output_file (str): The path where the output CSV file will be saved.
    """
    paths_by_pair = {}

    try:
        with open(input_file, mode='r', newline='', encoding='utf-8') as infile:
            reader = csv.DictReader(infile)
            # Check for the required columns in the header
            required_columns = ['src_scion_addr', 'dst_scion_addr', 'paths']
            if not all(col in reader.fieldnames for col in required_columns):
                print(f"Error: Input CSV must contain the columns: {', '.join(required_columns)}")
                return

            for row in reader:
                src_addr = row['src_scion_addr']
                dst_addr = row['dst_scion_addr']
                
                # The 'paths' cell might be empty
                if row['paths']:
                    paths = row['paths'].split(',')
                    pair_key = (src_addr, dst_addr)

                    if pair_key not in paths_by_pair:
                        paths_by_pair[pair_key] = []
                    
                    # Add the new paths to the list for this pair
                    paths_by_pair[pair_key].extend(paths)

    except FileNotFoundError:
        print(f"Error: The input file was not found at the specified path: {input_file}")
        return
    except Exception as e:
        print(f"An error occurred while reading the input file: {e}")
        return

    try:
        with open(output_file, mode='w', newline='', encoding='utf-8') as outfile:
            writer = csv.writer(outfile)
            # Write the header for the output file
            writer.writerow(['src_scion_addr', 'dst_scion_addr', 'path_1', 'path_2'])

            # Iterate through each source-destination pair and their collected paths
            for pair, path_list in paths_by_pair.items():
                src, dst = pair
                
                # Remove any potential empty strings that may result from splitting
                cleaned_path_list = [path for path in path_list if path]
                
                if not cleaned_path_list:
                    continue

                # Generate all possible combinations (Cartesian product) for the current pair's paths
                path_combinations = itertools.product(cleaned_path_list, repeat=2)

                # Write each path combination as a new row in the output file
                for combo in path_combinations:
                    writer.writerow([src, dst, combo[0], combo[1]])
        
        print(f"Successfully generated path combinations and saved them to {output_file}")

    except Exception as e:
        print(f"An error occurred while writing to the output file: {e}")
        return

if __name__ == '__main__':
    # Set up the command-line argument parser
    parser = argparse.ArgumentParser(
        description="Process a SCION path CSV to generate all path combinations for each source-destination pair."
    )
    parser.add_argument(
        "input_file", 
        help="The path to the input CSV file."
    )
    parser.add_argument(
        "output_file", 
        help="The path for the output CSV file."
    )
    
    args = parser.parse_args()
    
    # Execute the main processing function with the provided file paths
    process_scion_paths(args.input_file, args.output_file)