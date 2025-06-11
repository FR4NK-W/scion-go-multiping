# Figure 9c - Link Failures

To generate the path link failures figure, the required data is a result of isolated `showpaths` measurements from different hosts in SCIERA. The JSON output can be enabled via `--json` and the outputs of different hosts were concatenated into `combined_paths_with_rtt.json` from the previous figures. Afterwards, out of all path outputs a graph is generated and a simulation is performed to cut edges out of this graph and calculate the number of connected ASes for each cut. The generated data out of 100 different simulations run is under `data_percentage`. We compare the simulation results for the cases having the SCION paths of SCIERA available or only a single path.

The link failure plot can be generated running `link_failures.py`.