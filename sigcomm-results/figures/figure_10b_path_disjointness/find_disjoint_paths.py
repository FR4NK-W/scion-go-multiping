import csv
import re
import argparse
from typing import Set, List, Tuple

def parse_path_interfaces(path: str) -> Set[str]:
    """
    Parses a SCION path string to extract a set of unique interface IDs.

    An interface ID is defined as a combination of an ISD-AS and an interface number,
    formatted as 'ISD-AS#interface'. For example, in the segment '71-20965#7', 
    '71-20965#7' is the unique interface ID.

    Args:
        path (str): The SCION path string, which may contain one or more hops
                    separated by '->'.

    Returns:
        Set[str]: A set of unique interface IDs found in the path.
    """
    # Regex to find all occurrences of the ISD-AS#interface pattern
    # This pattern matches a sequence of digits and hyphens, followed by a '#', 
    # and then one or more digits.
    interface_pattern = r'\d+-\d+#\d+'
    
    # Find all matches of the pattern in the path string
    interfaces = re.findall(interface_pattern, path)
    
    return set(interfaces)

def find_disjoint_paths(input_file: str) -> None:
    """
    Analyzes a CSV file of SCION path pairs to identify and print those that are fully disjoint.

    Disjointness is determined by comparing the set of unique interface IDs from two paths.
    If the intersection of these sets is empty, the paths are considered disjoint.

    Args:
        input_file (str): The path to the input CSV file. The file should contain
                          'src_scion_addr', 'dst_scion_addr', 'path_1', and 'path_2' columns.
    """
    try:
        with open(input_file, mode='r', newline='', encoding='utf-8') as infile:
            reader = csv.DictReader(infile)
            
            # Verify that the necessary columns are present in the CSV file
            required_cols = ['src_scion_addr', 'dst_scion_addr', 'path_1', 'path_2']
            if not all(col in reader.fieldnames for col in required_cols):
                print(f"Error: The input CSV file must contain the following columns: {', '.join(required_cols)}")
                return

            print("Found fully disjoint paths for the following AS pairs:")
            
            # Keep track of the pairs that have already been printed to avoid duplicates
            printed_pairs: Set[Tuple[str, str]] = set()

            for row in reader:
                src_addr = row['src_scion_addr']
                dst_addr = row['dst_scion_addr']
                path1_str = row['path_1']
                path2_str = row['path_2']

                # Generate sets of interface IDs for both paths
                interfaces1 = parse_path_interfaces(path1_str)
                interfaces2 = parse_path_interfaces(path2_str)
                
                # If the intersection is empty, the paths are disjoint
                if not interfaces1.intersection(interfaces2):
                    print(f"path 1: {path1_str}, path 2: {path2_str}")
                    # Create a key for the source-destination pair to track printing
                    pair_key = (src_addr.split(',')[0], dst_addr.split(',')[0])
                    
                    if pair_key not in printed_pairs:
                        print(f"  - Source: {pair_key[0]}, Destination: {pair_key[1]}")
                        printed_pairs.add(pair_key)
                        
    except FileNotFoundError:
        print(f"Error: Input file not found at '{input_file}'")
    except Exception as e:
        print(f"An unexpected error occurred: {e}")

if __name__ == '__main__':
    # Configure the argument parser to handle the input file path
    parser = argparse.ArgumentParser(
        description="Find and print fully disjoint SCION paths from a given CSV file."
    )
    parser.add_argument(
        "input_file", 
        help="Path to the input CSV file containing SCION path pairs."
    )
    
    args = parser.parse_args()
    
    # Run the main function with the provided file path
    find_disjoint_paths(args.input_file)