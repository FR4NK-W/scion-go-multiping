import os
import re
import shutil
import sys
import numpy as np
import matplotlib.pyplot as plt


def generate_boxplot(data, pos):
    if len(data) > 0:
        dummy_required = pos in [3, 7]
        return r"""
\addplot+[
    fill,
    fill opacity=0.7,
    draw=linecolor_before,
    thick,
    boxplot prepared={
        median=%.2f,
        upper quartile=%.2f,
        lower quartile=%.2f,
        upper whisker=%.2f,
        lower whisker=%.2f,
    },
] coordinates {};""" % (
            np.median(data), np.percentile(data, 75), np.percentile(data, 25), max(data), min(data)
        ) + r"""

%dummy, for spacing
\addplot+[
    boxplot prepared={
        median=-10,
        upper quartile=-10,
        lower quartile=-10,
        upper whisker=-10,
        lower whisker=-10,
    },]coordinates {};
%dummy, for spacing
\addplot+[
    boxplot prepared={
        median=-10,
        upper quartile=-10,
        lower quartile=-10,
        upper whisker=-10,
        lower whisker=-10,
    },]coordinates {};
""" * dummy_required
    else:
        return r"\addplot coordinates {(%d,0)};" % pos  # Empty boxplot if no data


# Regex patterns to extract the required values
patterns = {
    'startup': r"Startup:\s*(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d+\s[^\s]+)",
    'starting': r"Starting\s.*\s(\d+)",
    'completed_hinter': r"Completed hinter\s.*\s(\d+)",
    'starting_fetch_config': r"Starting fetch config:\s*(\d+)",
    'completed_fetch_config': r"Completed fetch config:\s*(\d+)",
    'starting_trcs': r"Start fetching TRCs:\s*(\d+)",
    'completed_trcs': r"Completed fetching TRCs:\s*(\d+)",
    'starting_topo': r"Start fetching topo:\s*(\d+)",
    'completed_topo': r"Completed fetching topo:\s*(\d+)",
    'completed_bootstrapping': r"Completed bootstrapping:\s*(\d+)",
    'failed': r"failed"
}

files = ['Windows_output.txt', 'Linux_output.txt', 'macOS_output.txt']
all_configs_latencies = []
all_dns_latencies = []
all_total_latencies = []

# Print to file, redirect stdout
stdout = sys.stdout
f = open('performance.tex', 'w')
sys.stdout = f

for file_name in files:
    with open(file_name, 'r') as file:
        output = file.read()

    runs = output.split("Startup:")

    results = []

    for run in runs[1:]:  # skip the first empty part before the first "Startup:"

        if "failed" in run:
            continue  # Skip this run if it "failed"

        # Extract the values using regex
        run_data = {}
        run_data['starting_dns_hinter'] = int(re.search(patterns['starting'], run).group(1)) if re.search(patterns['starting'], run) else None
        run_data['completed_dns_hinter'] = int(re.search(patterns['completed_hinter'], run).group(1)) if re.search(patterns['completed_hinter'], run) else None
        run_data['starting_fetch_config'] = int(re.search(patterns['starting_fetch_config'], run).group(1)) if re.search(patterns['starting_fetch_config'], run) else None
        run_data['completed_fetch_config'] = int(re.search(patterns['completed_fetch_config'], run).group(1)) if re.search(patterns['completed_fetch_config'], run) else None
        run_data['starting_trcs'] = int(re.search(patterns['starting_trcs'], run).group(1)) if re.search(patterns['starting_trcs'], run) else None
        run_data['completed_trcs'] = int(re.search(patterns['completed_trcs'], run).group(1)) if re.search(patterns['completed_trcs'], run) else None
        run_data['starting_topo'] = int(re.search(patterns['starting_topo'], run).group(1)) if re.search(patterns['starting_topo'], run) else None
        run_data['completed_topo'] = int(re.search(patterns['completed_topo'], run).group(1)) if re.search(patterns['completed_topo'], run) else None
        run_data['completed_bootstrapping'] = int(re.search(patterns['completed_bootstrapping'], run).group(1)) if re.search(patterns['completed_bootstrapping'], run) else None

        # Calculate the latencies
        if (run_data['completed_fetch_config'] is not None and run_data['starting_fetch_config'] is not None):
            run_data['configs_latency'] = (run_data['completed_fetch_config'] - run_data['starting_fetch_config'])/1000
        if (run_data['completed_trcs'] is not None and run_data['starting_trcs'] is not None):
            run_data['trcs_latency'] = (run_data['completed_trcs'] - run_data['starting_trcs'])/1000
        if (run_data['completed_topo'] is not None and run_data['starting_topo'] is not None):
            run_data['topo_latency'] = (run_data['completed_topo'] - run_data['starting_topo'])/1000
        if (run_data['completed_dns_hinter'] is not None and run_data['starting_dns_hinter'] is not None):
            run_data['dns_latency'] = (run_data['completed_dns_hinter'] - run_data['starting_dns_hinter'])/1000
        if (run_data['completed_bootstrapping'] is not None and run_data['starting_dns_hinter'] is not None):
            run_data['total_latency'] = (run_data['completed_bootstrapping'] - run_data['starting_dns_hinter'])/1000

        # Append the valid run data
        results.append(run_data)

    # Now `results` contains the parsed values for each valid run
    # for i, result in enumerate(results, 1):
    #    print(f"Run {i}: {result}")


    configs_latencies = [run['configs_latency'] for run in results if 'configs_latency' in run]
    trcs_latencies = [run['trcs_latency'] for run in results if 'trcs_latency' in run]
    topo_latencies = [run['topo_latency'] for run in results if 'topo_latency' in run]
    dns_latencies = [run['dns_latency'] for run in results if 'dns_latency' in run]
    total_latencies = [run['total_latency'] for run in results if 'total_latency' in run]

    all_configs_latencies.append(configs_latencies)
    all_dns_latencies.append(dns_latencies)
    all_total_latencies.append(total_latencies)

    # Plot the boxplots
    plt.figure(figsize=(10, 6))
    plt.boxplot([configs_latencies, trcs_latencies, topo_latencies, dns_latencies, total_latencies],
                tick_labels=['Config Retrieval', 'TRC Latency', 'Topo Latency', 'DNS Latency', 'Total Latency'])

    plt.title('Latency Distribution Across Iterations')
    plt.ylabel('Latency (ms)')
    plt.xticks(rotation=45)
    plt.grid(True)


# print(len(all_configs_latencies))
# Show the plot
# plt.show()


# Generate LaTeX code
print(r"""
\begin{axis}
    [
    name=ax1,
    boxplot/draw direction=y,
    ylabel={Latency (ms)},
    boxplot/box extend=0.9,
    height=4cm,
    width=0.6\textwidth,
    cycle list={{linecolor_windows},{linecolor_linux},{linecolor_mac},{linecolor_before},{linecolor_before}},
    xtick={2,7,12},
    ymin=-1,
    xtick pos=bottom,
    xticklabels={
        {\shortstack{Hint retrieval}}, 
        {\shortstack{Config retrieval}}, 
        {\shortstack{Total}}, 
    },
    legend style={at={(0.5,1.01)}, anchor=south, /tikz/every even column/.append style={column sep=0.1cm}, draw=none},
    legend columns=-1,
    legend image code/.code={\draw[#1] (0cm,-0.05cm) rectangle (0.4cm,0.1cm);},
    ]
\addlegendentry{Windows\;};
\addlegendentry{Linux\;};
\addlegendentry{Mac\;};
""")

positions = [1, 2, 3, 5, 6, 7, 9, 10, 11]  # X-axis positions
all_latencies = [all_dns_latencies, all_configs_latencies, all_total_latencies]

for i, latency_group in enumerate(all_latencies):  # DNS, Config retrieval, and Total
    for j, data in enumerate(latency_group):  # Three different files
        print(generate_boxplot(data, positions[i * 3 + j]))

print(r"""
\end{axis}
""")
f.close()
sys.stdout = stdout

# Render TikZ to pdf
# Create basic template
with open('render.tex', 'w') as f:
    preamble = r"""
%\documentclass[tikz, border=1mm]{standalone}
\documentclass{minimal}
\usepackage{tikz}
\usepackage{pgfplots}\pgfplotsset{compat=1.18}
\usepackage{pgfplotstable}
\usepgfplotslibrary{groupplots,statistics}

\begin{document}
\begin{tikzpicture}
\definecolor{linecolor_before}{RGB}{00, 00, 50}
\definecolor{linecolor_after}{RGB}{80, 20, 80} 
\definecolor{linecolor_linux}{RGB}{00, 00, 200}
\definecolor{linecolor_windows}{RGB}{75, 146, 195}
\definecolor{linecolor_mac}{RGB}{255,152,62} 
"""
    f.write(preamble)
    with open("performance.tex") as tikz_figure:
        f.write(tikz_figure.read())
    epilogue = r"""
\end{tikzpicture}
\end{document}
"""
    f.write(epilogue)

pdflatex = shutil.which("pdflatex")
if pdflatex:
    os.system("pdflatex -jobname bootstrap_latency render.tex")
else:
    print("WARNING: A working pdflatex installation including texlive-pictures for pgf is required for rendering this figure. (Figure4)")
    print("WARNING: Make sure pdflatex is on your path and the texlive-pictures package providing Tikz is installed.") 
