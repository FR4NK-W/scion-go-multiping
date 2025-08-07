import matplotlib as mpl
mpl.use('agg')
import pandas as pd
import matplotlib.pyplot as plt
import numpy as np
from scipy.interpolate import interp1d

# --- CONFIGURATION SETTINGS ---
# Move all rcParams modifications to the top to ensure they apply to all plots.
# This tells matplotlib to use Type 42 (a.k.a. TrueType) fonts for PDF/PS output.
# This allows the fonts to be embedded, making the PDF self-contained.
mpl.rcParams['pdf.fonttype'] = 42
mpl.rcParams['ps.fonttype'] = 42
mpl.rcParams['axes.labelsize'] = 18
# Using usetex delegates text rendering to your LaTeX installation.
# plt.rcParams['text.usetex'] = True
# Note: font.family is often ignored when usetex=True, as LaTeX handles fonts.
# To use a specific font, you might need to configure the LaTeX preamble.
# For example:
# plt.rcParams['text.latex.preamble'] = r'\usepackage{amsmath}\usepackage{dejavu}'
plt.rcParams['font.family'] = 'DejaVu Sans'

def get_axis(axes_props, size=(8, 4)):
    fig, ax = plt.subplots(figsize=size)
    ax.set(**axes_props)
    return fig, ax


def normalize(df, colname):
    return (df[colname]/df[colname].sum()).mul(100)


def get_mean(df, bin_width):
    return (df['ping_count'] * (df['rtt'] + bin_width/2)).sum()/(df['ping_count'].sum())


def interpolate(target_y, df_scion, df_ip):
    y_scion = np.cumsum(df_scion["ping_count_norm"])
    x_scion = df_scion['rtt']

    y_ip = np.cumsum(df_ip["ping_count_norm"])
    x_ip = df_ip['rtt']

    interp_scion = interp1d(y_scion, x_scion, bounds_error=False, fill_value="extrapolate")
    interp_ip = interp1d(y_ip, x_ip, bounds_error=False, fill_value="extrapolate")

    x_at_y_scion = interp_scion(target_y)
    x_at_y_ip = interp_ip(target_y)

    print(f"SCION: x at y={target_y} is {x_at_y_scion}")
    print(f"IP:    x at y={target_y} is {x_at_y_ip}")
    return x_at_y_scion, x_at_y_ip

# Define bucket width (in ms)
bucket_width = 10

df_scion = pd.read_csv("gen/new_scion_pings_per_hour_final.csv").rename(columns={'lower_bound': 'rtt'})
df_ip = pd.read_csv("gen/new_ip_pings_per_hour_final.csv").rename(columns={'lower_bound': 'rtt'})

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
                    "ylabel": "Proportion of Pings (%)",
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

# fig.savefig("sciera_hist_norm_grouped.png", dpi=600, bbox_inches="tight", transparent=True)
fig.savefig("sciera_hist_norm_grouped.pdf", bbox_inches="tight")


ax_padding = 5
fig, ax = get_axis({"xlabel": "RTT (ms)",
                    "ylabel": "Proportion of Pings (%)",
               #     "title": "CDF of SCION vs IP Ping Latency",
                    "xlim": [-ax_padding, rtt_cutoff + ax_padding],
                    "ylim": [-ax_padding, 100+ax_padding]}) #, size=(12, 6))

ax.tick_params(axis='both', labelsize=14)

x50s, x50i = interpolate(50, df_scion, df_ip)
x90s, x90i = interpolate(90, df_scion, df_ip)

ax.plot(df_scion['rtt'], np.cumsum(df_scion["ping_count_norm"]), label='SCION')
ax.plot(df_ip['rtt'], np.cumsum(df_ip["ping_count_norm"]), label='IP', linestyle='--')
ax.grid(**grid_props)

#ax.annotate(text='SCION', xy=(156, 50), marker='x')
ax.plot([x50s], [50], marker='x', color='red')
ax.plot([x50i], [50], marker='x', color='red')

ax.hlines(y=50, xmin=-5, xmax=max(x50i,x50s), color='red', linestyle='dotted')
ax.vlines(x=x50s, ymin=-5, ymax=50, color='red', linestyle='dotted')
ax.vlines(x=x50i, ymin=-5, ymax=50, color='red', linestyle='dotted')



ax.plot([x90s], [90], marker='x', color='green')
ax.plot([x90i], [90], marker='x', color='green')
ax.hlines(y=90, xmin=-5, xmax=max(x90i, x90s), color='green', linestyle='dotted')
ax.vlines(x=x90s, ymin=-5, ymax=90, color='green', linestyle='dotted')
ax.vlines(x=x90i, ymin=-5, ymax=90, color='green', linestyle='dotted')

#ax.plot(df_scion.rtt, np.cumsum(df_scion["ping_count_norm"] - df_ip.ping_count_norm))

ax.legend(fontsize=17, loc='upper left', bbox_to_anchor=(0, 0.85))
plt.rcParams['pdf.fonttype'] = 42
plt.rcParams['ps.fonttype'] = 42
plt.rcParams['font.family'] = 'DejaVu Sans'
plt.tight_layout()

# fig.savefig("sciera_hist_norm_cdf.png", dpi=600, bbox_inches="tight", transparent=True)
fig.savefig("sciera_hist_norm_cdf.pdf", bbox_inches="tight" )#, transparent=True)

