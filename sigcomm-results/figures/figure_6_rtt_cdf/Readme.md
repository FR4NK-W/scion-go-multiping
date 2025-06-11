# Figure 6 - RTT CDF

To reconstruct this figure, following query need to be executed against our dataset and the results need to be stored into `avg_scion_pings_pairs.csv`.

```sql
SELECT pr.src_scion_addr, pr.dst_scion_addr, avg(pr.rtt) FROM ping_results pr

WHERE pr.src_scion_addr NOT IN  ( '71-2:0:35,192.168.1.1:0') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new 
GROUP BY pr.src_scion_addr, pr.dst_scion_addr
```

Next, the following query need to be executed against our dataset and the results need to be stored into `avg_ip_pings_pairs.csv`.

```sql
SELECT pr.src_addr, pr.dst_addr, avg(pr.rtt) FROM ip_ping_results pr

WHERE pr.src_addr NOT IN  ( '192.168.1.1') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new
GROUP BY pr.src_addr, pr.dst_addr
```

Next, the `merge.py` command needs to be executed to generate the `merged_pings.csv` file. Finally, `plot.py` will generate the plot under `rtt_ratio_cdf.pdf`