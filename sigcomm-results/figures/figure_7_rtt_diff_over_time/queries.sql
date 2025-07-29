SELECT time_bucket('1 hour', ping_time_new) AS hour,
avg(pr.rtt) as avg_rtt,
COUNT(*) FILTER (WHERE pr.success = true AND pr.rtt > 0) * 1.0 / COUNT(*) AS success_ratio,
COUNT(*) AS prcount,
pr.src_scion_addr
FROM ping_results pr

WHERE pr.src_scion_addr NOT IN  ( '71-2:0:35,192.168.1.1:0')
		AND pr.dst_scion_addr NOT IN ('71-225,127.0.0.1:30041') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new < '2025-02-08'
GROUP BY hour, pr.src_scion_addr
ORDER BY hour





SELECT time_bucket('1 hour', ping_time_new) AS hour,
avg(pr.rtt) as avg_rtt,
COUNT(*) AS prcount,
pr.src_addr

FROM ip_ping_results pr

WHERE pr.src_addr NOT IN  ( '192.168.1.1') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new < '2025-02-08'
GROUP BY hour, pr.src_addr
ORDER BY hour

