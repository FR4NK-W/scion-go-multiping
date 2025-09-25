# Figure 4- Bootstrapping latency by OS

This figure was directly generated from the data as TikZ gaphic for the paper.

We provide the datapoints as `.csv` and the generated Latex sources.

To reconstruct this figure independently without rendering the full paper, we provide the script `bootstrap_plot_by_OS.py` outputting both a pdf rendering of the plot and the TikZ code from the raw data.
A pdflatex installation including texlive-pictures for pgf is required.

The data was obtained by executing the bootstrapper 30 times per hinting mechanism on each OS in the SCION enabled network with the SCION bootstrapping server of the AS serving the local topology and the TRC files. The median latency and the variance for the hint retrieval, the configuration retrieval (local topology and TRC files) and for bootstrapping completion are reported for Windows 11, Linux (Ubuntu 22.04) and macOS 11.

Instructions for setting up a bootstrapping server are provided in the README.md of the bootstrapper at https://github.com/netsec-ethz/bootstrapper?tab=readme-ov-file#nginx-web-server. Information for setting up the hinting mechanisms with dnsmasq is given in https://github.com/netsec-ethz/bootstrapper?tab=readme-ov-file#discovery-mechanisms were special care is required, for the general DHCP or DNS record configuration with dnsmasq refer to the dnsmasq user manual at https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html.

The bootstrapper was instrumented to produce additional debug timing output, the relevant changes for the collection of timing data are available on the branch `https://github.com/FR4NK-W/bootstrapper/tree/timing_evaluation`, base version is https://github.com/netsec-ethz/bootstrapper/releases/tag/v0.0.8.

The measurements are collected in one output file per OS, and the Latex output for the final plot is generated using the script `bootstrap_plot_by_OS.py` and the provided measurement output files.
