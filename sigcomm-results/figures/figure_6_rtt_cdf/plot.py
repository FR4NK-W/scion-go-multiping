import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from brokenaxes import brokenaxes

def plot_cdf():
    # Load merged data
    df = pd.read_csv("gen/merged_pings_filtered.csv")
    
    # Drop rows where avg_ip is missing
    df = df.dropna(subset=['avg_ip'])
    
    # Calculate RTT difference ratio where 1 represents equality
    df['rtt_diff_ratio'] = df['avg_scion'] / df['avg_ip']
    
    # Sort values for CDF
    sorted_ratios = np.sort(df['rtt_diff_ratio'])
    cdf = np.arange(1, len(sorted_ratios) + 1) / len(sorted_ratios)

    # Create DataFrame for easier manipulation
    cdf_df = pd.DataFrame({'rtt_diff_ratio': sorted_ratios, 'cdf': cdf})

    f, ax = plt.subplots(figsize=(6, 4.4))

    ax.set_xscale("log")
    ax.set_xlim(0.1, 12)
    ax.set_ylim(0, 1.01)

    # Filter out non-positive RTT ratios
    df = df[df['rtt_diff_ratio'] > 0]

    # Plot the full CDF
    sns.lineplot(data=cdf_df, x='rtt_diff_ratio', y='cdf', color='royalblue', ax=ax)
    marker_data = cdf_df.iloc[:-7]  # exclude last 7 rows

    ax.scatter(
        marker_data['rtt_diff_ratio'],
        marker_data['cdf'],
        marker="o",
        edgecolors='white',   # white outline
        linewidths=0.8,  
        color='royalblue',
        s=30,          # size of the marker
        zorder=3       # keep it above the line
    )
    
    highlight_xs = [1.829669, 1.837245, 3.320837, 3.830811, 3.863771, 5.283462, 10.330083]
    highlight_colors = ['red', 'purple', 'forestgreen','red','red','forestgreen','forestgreen'] 

    for x_val, color in zip(highlight_xs, highlight_colors):
        closest_idx = (np.abs(cdf_df['rtt_diff_ratio'] - x_val)).idxmin()
        x_point = cdf_df.loc[closest_idx, 'rtt_diff_ratio']
        y_point = cdf_df.loc[closest_idx, 'cdf']
        ax.scatter(x_point, y_point, color=color, s=30, zorder=5, marker="v",
        edgecolors='white',   # white outline
        linewidths=0.8,  )

    ax.annotate(
        '', xy=(cdf_df.iloc[-4]['rtt_diff_ratio'], cdf_df.iloc[-4]['cdf'] -0.01),
        xytext=(3, 0.74), arrowprops=dict(arrowstyle='-', linestyle=":",color='red')
    )

    ax.annotate(
        '', xy=(cdf_df.iloc[-7]['rtt_diff_ratio'], cdf_df.iloc[-7]['cdf'] -0.01),
        xytext=(2.7, 0.74), arrowprops=dict(arrowstyle='-',linestyle=":", color='red')
    )

    ax.annotate(
        '', xy=(cdf_df.iloc[-1]['rtt_diff_ratio'], cdf_df.iloc[-1]['cdf'] -0.01),
        xytext=(7, 0.91), arrowprops=dict(arrowstyle='-', linestyle=":",color='forestgreen')
    )
    ax.annotate(
        '', xy=(cdf_df.iloc[-5]['rtt_diff_ratio'], cdf_df.iloc[-5]['cdf']),
        xytext=(5.5, 0.91), arrowprops=dict(arrowstyle='-', linestyle=":",color='forestgreen')
    )


    ax.annotate(
    'Cut of direct\nlink between\nSingapore and\nDaejon',
    xy=(3.863771, 0.98), xycoords='data', fontsize=9,
    xytext=(-45, -100), textcoords='offset points', color='red',
    arrowprops=dict(arrowstyle="-",linestyle=":",color="red" )
    )
    

    ax.annotate(
        'UFMS to Equinix\nrouted through\nGÉANT',
        xy=(1.837245, 0.97), xycoords='data', fontsize=9,
        xytext=(-35, -150), textcoords='offset points', color='purple',
        arrowprops=dict(arrowstyle="-",linestyle=":",color="purple" )
        )
    
    ax.annotate(
        'Routing\ninstabilities\nover BRIDGES',
        xy=(5.283462, 0.99), xycoords='data', fontsize=9,
        xytext=(-13, -51), textcoords='offset points', color='forestgreen',
        arrowprops=dict(arrowstyle="-",linestyle=":",color="forestgreen" )
        )
    
    

    # Annotations
    ax.axvline(x=1, color='red', linestyle='--', label="SCION = IP",zorder=10)
    ax.legend(fontsize=13)

    ax.set_xlabel("RTT Ratio (SCION / IP)", fontsize=14)
    ax.set_ylabel("CDF", fontsize=14)

    ax.tick_params(axis='both', which='major', labelsize=11)
    ax.grid(True, which='both', linestyle='--', alpha=0.6)

    plt.subplots_adjust(left=0.15, right=0.95, bottom=0.15, top=0.95)

    plt.rcParams['pdf.fonttype'] = 42
    plt.rcParams['ps.fonttype'] = 42

    # plt.tight_layout(rect=[0, 0, 1, 1])  # Adjust layout to fit title
    plt.savefig("rtt_ratio_cdf.pdf")
    # plt.show()

if __name__ == "__main__":
    plot_cdf()
