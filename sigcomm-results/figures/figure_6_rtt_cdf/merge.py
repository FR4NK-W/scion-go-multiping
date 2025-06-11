import pandas as pd

def merge_pings():
    # Load CSV files
    scion_pings = pd.read_csv("avg_scion_pings.csv")
    ip_pings = pd.read_csv("avg_ip_pings.csv")
    address_mapping = pd.read_csv("addressmapping.csv")
    
    # Merge SCION source and destination addresses with corresponding IPs
    merged_scion = scion_pings.merge(
        address_mapping[['src_scion_addr', 'src_ip_addr']],
        on='src_scion_addr', how='left'
    ).merge(
        address_mapping[['dst_scion_addr', 'dst_ip_addr']],
        on='dst_scion_addr', how='left'
    )
    
    # Rename columns for clarity
    merged_scion.rename(columns={'src_ip_addr': 'mapped_src_ip', 'dst_ip_addr': 'mapped_dst_ip'}, inplace=True)
    
    # Merge with IP pings data
    final_merged = merged_scion.merge(
        ip_pings, left_on=['mapped_src_ip', 'mapped_dst_ip'], right_on=['src_addr', 'dst_addr'], how='left', suffixes=('_scion', '_ip')
    )
    
    # Drop unnecessary columns
    final_merged = final_merged[['src_scion_addr', 'dst_scion_addr', 'avg_scion', 'mapped_src_ip', 'mapped_dst_ip', 'avg_ip']]
    
    # Save to CSV
    final_merged.to_csv("merged_pings.csv", index=False)
    print("Merged data saved to merged_pings.csv")

if __name__ == "__main__":
    merge_pings()
