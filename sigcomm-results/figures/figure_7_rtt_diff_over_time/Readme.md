# Figure 7 - RTT SCION in relation to IP over time

To reconstruct this figure, following query need to be executed against our dataset and the results need to be stored into `scion_pings_hour.csv`.

```sql
SELECT time_bucket('1 hour', ping_time_new) AS hour,
avg(pr.rtt) as avg_rtt,
COUNT(*) FILTER (WHERE pr.success = true AND pr.rtt > 0) * 1.0 / COUNT(*) AS success_ratio,
COUNT(*) AS prcount,
pr.src_scion_addr
FROM ping_results pr

WHERE pr.src_scion_addr NOT IN  ( '71-2:0:35,192.168.1.1:0') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new < '2025-02-08'
GROUP BY hour, pr.src_scion_addr
ORDER BY hour
```

Next, the following query need to be executed against our dataset and the results need to be stored into `ip_pings_hour.csv`.

```sql
SELECT time_bucket('1 hour', ping_time_new) AS hour,
avg(pr.rtt) as avg_rtt,
COUNT(*) AS prcount,
pr.src_addr

FROM ip_ping_results pr

WHERE pr.src_addr NOT IN  ( '192.168.1.1') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new < '2025-02-08'
GROUP BY hour, pr.src_addr
ORDER BY hour
```

Finally, `plot.py` will generate the plot under `rtt_ratio_time_scaled.pdf`