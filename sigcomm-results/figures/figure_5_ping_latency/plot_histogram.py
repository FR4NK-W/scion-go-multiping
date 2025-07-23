import matplotlib as mpl
mpl.use('agg')
import pandas as pd
import matplotlib.pyplot as plt
import numpy as np


mpl.rcParams['axes.labelsize'] = 18
plt.rcParams['text.usetex'] = True


# SQL to generate ping histogram CSVs (substitute table name as needed for IP/SCION measurements):
# (last timestamp: 2025-01-30 00:15:01.372327524+00:00)
#
# WITH params AS (
#     SELECT 0 AS rtt_min, 400 AS rtt_max, rtt_max/10 AS bucket_count
# ),
# bucketed AS (
#     SELECT
#         width_bucket(rtt, p.rtt_min, p.rtt_max, p.bucket_count) AS rtt_bucket
#     FROM ip_ping_results s, params p
#     where s.success = true and s.rtt > 0
# )
# SELECT
#     rtt_bucket,
#     p.rtt_min + (rtt_bucket - 1) * ((p.rtt_max - p.rtt_min) * 1.0 / p.bucket_count) AS lower_bound,
#     COUNT(*) AS ping_count
# FROM bucketed b, params p
# GROUP BY rtt_bucket, lower_bound
# ORDER BY rtt_bucket;

def get_axis(axes_props, size=(8, 4)):
    fig, ax = plt.subplots(figsize=size)
    ax.set(**axes_props)
    return fig, ax


def normalize(df, colname):
    return (df[colname]/df[colname].sum()).mul(100)


def get_mean(df, bin_width):
    return (df['ping_count'] * (df['rtt'] + bin_width/2)).sum()/(df['ping_count'].sum())


# Define bucket width (in ms)
bucket_width = 10

df_scion = pd.read_csv("new_scion_pings_per_hour_final.csv").rename(columns={'lower_bound': 'rtt'})
df_ip = pd.read_csv("new_ip_pings_per_hour_final.csv").rename(columns={'lower_bound': 'rtt'})

df_scion["ping_count_norm"] = normalize(df_scion, "ping_count")
df_ip["ping_count_norm"] = normalize(df_ip, "ping_count")

rtt_bins = df_ip['rtt']


bar_props = {
    'width': 0.25,
    'alpha': 0.8,
    'align': 'edge',
    'edgecolor': "black"
}

grid_props = {
    'visible': True,
    'linestyle': 'dashed',
    'color':'gray',
    'alpha': 0.5
}

scale = 0.75

hist_scion = (df_scion.groupby(df_scion.index//5)
                       .agg({'ping_count': 'sum',
                             'ping_count_norm': 'sum',
                             'rtt': 'min',
                             'rtt_bucket': lambda b: b.min()//5}))
hist_ip = (df_ip.groupby(df_ip.index//5)
                       .agg({'ping_count': 'sum',
                             'ping_count_norm': 'sum',
                             'rtt': 'min',
                             'rtt_bucket': lambda b: b.min()//5}))


fig, ax = get_axis({"xlabel": "RTT (ms)",
                    "ylabel": "Proportion of Pings (\\%)",
                    "xlim": [-0.1, hist_ip["rtt"].index.max()*scale+2*bar_props['width']+ 0.1]})

ax.bar(hist_scion["rtt"].index*scale, hist_scion["ping_count_norm"], label="SCION", **bar_props)
ax.bar(hist_ip["rtt"].index*scale + bar_props['width'], hist_ip["ping_count_norm"], label="IP", **bar_props)

# bin together long tail with RTT > 400ms
rtt_cutoff = 400

ax.set_xticks(hist_scion.index*scale + bar_props['width'],
              labels=hist_scion.rtt.apply(lambda x: f"[{int(x)}-{int(x)+bucket_width*5})" if x < rtt_cutoff else f"{rtt_cutoff}+" ),
              size=12)
ax.legend(fontsize=14)
ax.grid(**grid_props)
plt.tight_layout()

fig.savefig("sciera_hist_norm_grouped.png", dpi=600, bbox_inches="tight", transparent=True)
fig.savefig("sciera_hist_norm_grouped.pdf", bbox_inches="tight")


ax_padding = 5
fig, ax = get_axis({"xlabel": "RTT (ms)",
                    "ylabel": "Proportion of Pings (\\%)",
               #     "title": "CDF of SCION vs IP Ping Latency",
                    "xlim": [-ax_padding, rtt_cutoff + ax_padding],
                    "ylim": [-ax_padding, 100+ax_padding]}) #, size=(12, 6))

ax.tick_params(axis='both', labelsize=14)

ax.plot(df_scion['rtt'], np.cumsum(df_scion["ping_count_norm"]), label='SCION')
ax.plot(df_ip['rtt'], np.cumsum(df_ip["ping_count_norm"]), label='IP', linestyle='--')
ax.grid(**grid_props)

#ax.annotate(text='SCION', xy=(156, 50), marker='x')
ax.plot([167], [60], marker='x', color='red')
ax.plot([227], [60], marker='x', color='red')
ax.hlines(y=60, xmin=-5, xmax=227, color='red', linestyle='dotted')
ax.vlines(x=167, ymin=-5, ymax=60, color='red', linestyle='dotted')
ax.vlines(x=227, ymin=-5, ymax=60, color='red', linestyle='dotted')



ax.plot([285], [90], marker='x', color='green')
ax.plot([375], [90], marker='x', color='green')
ax.hlines(y=90, xmin=-5, xmax=375, color='green', linestyle='dotted')
ax.vlines(x=285, ymin=-5, ymax=90, color='green', linestyle='dotted')
ax.vlines(x=375, ymin=-5, ymax=90, color='green', linestyle='dotted')

#ax.plot(df_scion.rtt, np.cumsum(df_scion["ping_count_norm"] - df_ip.ping_count_norm))

ax.legend(fontsize=18)
plt.tight_layout()

fig.savefig("sciera_hist_norm_cdf.png", dpi=600, bbox_inches="tight", transparent=True)
fig.savefig("sciera_hist_norm_cdf.pdf", bbox_inches="tight" )#, transparent=True)

