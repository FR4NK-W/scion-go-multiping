import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from brokenaxes import brokenaxes

def plot_cdf():
    # Load merged data
    df = pd.read_csv("merged_pings_filtered.csv")
    
    # Drop rows where avg_ip is missing
    df = df.dropna(subset=['avg_ip'])
    
    # Calculate RTT difference ratio where 1 represents equality
    df['rtt_diff_ratio'] = df['avg_scion'] / df['avg_ip']
    
    # Sort values for CDF
    sorted_ratios = np.sort(df['rtt_diff_ratio'])
    cdf = np.arange(1, len(sorted_ratios) + 1) / len(sorted_ratios)

    f, (ax1, ax2) = plt.subplots(ncols=2, nrows=1, sharey=True, width_ratios=[4, 1])
    ax = sns.lineplot(x=sorted_ratios, y=cdf, marker="o", linestyle="-", color="blue", ax=ax1)
    ax = sns.lineplot(x=sorted_ratios, y=cdf, marker="o", linestyle="-", color="blue", ax=ax2)

    ax1.set_xlim(0, 2)
    ax1.set_xticks([0, 0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 1.75,])
    ax2.set_xlim(3, 3.25)
    ax2.set_xticks([3.0, 3.25, 3.5])
    d = .5  # proportion of vertical to horizontal extent of the slanted line
    kwargs = dict(marker=[(-d, -1), (d, 1)], markersize=12,
                  linestyle="none", color='k', mec='k', mew=1, clip_on=False)
    ax1.plot([1, 1], [1, 0], transform=ax1.transAxes, **kwargs)
    ax2.plot([0, 0], [1, 0], transform=ax2.transAxes, **kwargs)

    ax1.spines.right.set_visible(False)
    ax2.spines.left.set_visible(False)
    ax2.yaxis.tick_right()
    plt.subplots_adjust(wspace=0.08, hspace=0,  left=0.1, right=0.95, bottom=0.15, top=0.95)

    # Centered title for both subplots
    # f.suptitle("CDF of RTT Ratio Between SCION and IP", fontsize=16, fontweight='bold')

    # Labels
    f.supxlabel("RTT Ratio (SCION / IP)", fontsize=14)
    f.supylabel("CDF", fontsize=14)

    ax1.axvline(x=1, color='red', linestyle='--', label="SCION = IP")
    ax1.legend()

    ax1.grid(linestyle="--", alpha=0.6)
    ax2.grid(linestyle="--", alpha=0.6)

    # plt.tight_layout(rect=[0, 0, 1, 1])  # Adjust layout to fit title
    plt.savefig("rtt_ratio_cdf.pdf")
    # plt.show()

if __name__ == "__main__":
    plot_cdf()
