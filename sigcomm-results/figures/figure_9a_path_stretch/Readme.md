# Figure 9a - Path Stretch

To generate the path stretch figure, the required data is a result of isolated `showpaths` measurements from different hosts in SCIERA. The JSON output can be enabled via `--json` and the outputs of different hosts were concatenated into `combined_paths_with_rtt.json`.

With this file, the path stretch plot can be generated running `path_stretch.py`.