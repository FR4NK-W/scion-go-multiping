import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt

def plot_heatmap(csv_file):
    df = pd.read_csv(csv_file)
    df_pivot = df.pivot(index='src_scion_addr', columns='dst_scion_addr', values='avg_paths')
    df_pivot = df_pivot.round(2)  # Round values to 2 decimal places
    
    plt.figure(figsize=(8, 6))
    sns.heatmap(df_pivot, cmap="coolwarm", annot=True, fmt=".0f")
    plt.title("Average Number of active Paths")
    plt.xlabel("Destination SCION AS")
    plt.ylabel("Source SCION AS")
    plt.xticks(rotation=45, ha="right")
    plt.yticks(rotation=0)
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    import sys
    if len(sys.argv) != 2:
        print("Usage: python plot_heatmap.py <input_csv>")
        sys.exit(1)
    
    input_csv = sys.argv[1]
    plot_heatmap(input_csv)