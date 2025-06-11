# Figure 8 - Path Statistics Heatmap

To reconstruct this figure, following query need to be executed against our dataset and the results need to be stored into `path_stats_clean.csv`.

```sql
/* Path Statistics aggregated */
SELECT
    src_scion_addr,
    dst_scion_addr,
    AVG(available_paths) AS avg_paths,
    MIN(available_paths) AS min_paths,
    MAX(available_paths) AS max_paths,
    AVG(min_hops) AS avg_min_hops,
    AVG(max_hops) AS avg_max_hops
FROM path_statistics
WHERE available_paths > 0 and min_hops < 10000 and max_hops < 10000 AND lookup_time_new < '2025-02-08'
AND src_scion_addr NOT IN  ( '71-2:0:35,192.168.1.1:0') AND dst_scion_addr not in ('64-2:0:9,129.132.230.98:30041', '71-2:0:18,192.168.1.1:30041', '71-88,127.0.0.1:30041')
GROUP BY src_scion_addr, dst_scion_addr
ORDER BY src_scion_addr, dst_scion_addr;
```

Finally, `plot.sh` will generate the plot under `heatmap_manual.pdf`