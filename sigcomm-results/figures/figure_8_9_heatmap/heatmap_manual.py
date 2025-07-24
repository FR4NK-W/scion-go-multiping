import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
import argparse

def plot_heatmap(csv_file, x_order=None, y_order=None):
    df = pd.read_csv(csv_file)
    df_pivot = df.pivot(index='src_scion_addr', columns='dst_scion_addr', values='max_paths')
    df_pivot = df_pivot.round(2)  # Round values to 2 decimal places

    if x_order:
        x_order = x_order.split(',')
        df_pivot = df_pivot.reindex(columns=x_order)
    
    if y_order:
        y_order = y_order.split(',')
        df_pivot = df_pivot.reindex(index=y_order)
    
    plt.figure(figsize=(7, 5))
    
    # Plot heatmap with increased font sizes
    sns.heatmap(df_pivot, cmap="Greens", annot=True, fmt=".0f", cbar=False, annot_kws={"size": 12})
    
    # plt.title("Max. Number of Active Paths", fontsize=16, fontweight='bold')
    plt.xlabel("Destination SCION AS", fontsize=14)
    plt.ylabel("Source SCION AS", fontsize=14)
    plt.xticks(rotation=45, ha="right", fontsize=12)
    plt.yticks(rotation=0, fontsize=12)
    plt.tight_layout()
    plt.savefig("heatmap_manual.pdf")
    # plt.show()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Plot SCION AS heatmap with custom axis order.")
    parser.add_argument("csv_file", help="Input CSV file containing SCION path data")
    parser.add_argument("--x_order", help="Comma-separated list of ASes for x-axis order", default=None)
    parser.add_argument("--y_order", help="Comma-separated list of ASes for y-axis order", default=None)
    
    args = parser.parse_args()
    plot_heatmap(args.csv_file, args.x_order, args.y_order)
