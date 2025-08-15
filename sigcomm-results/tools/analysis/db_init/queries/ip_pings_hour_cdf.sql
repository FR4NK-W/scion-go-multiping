SELECT time_bucket('1 hour', ping_time_new) AS hour,  pr.src_addr, pr.dst_addr, avg(pr.rtt) FROM ip_ping_results pr

WHERE pr.src_addr NOT IN  ( '192.168.1.1') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new < '2025-02-08'
GROUP BY hour, pr.src_addr, pr.dst_addr