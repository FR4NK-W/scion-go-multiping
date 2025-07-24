# Figure 10a - Path Latency Inflation

At first, store the output of this query into `path_stretch.csv`:

```sql
SELECT pr.fingerprint, pr.src_scion_addr, pr.dst_scion_addr, avg(pr.rtt) FROM ping_results pr

WHERE pr.src_scion_addr NOT IN  ( '71-2:0:35,192.168.1.1:0') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new < '2025-02-08'
GROUP BY pr.fingerprint, pr.src_scion_addr, pr.dst_scion_addr
```

With this file, the path stretch plot can be generated running `path_stretch.py`.