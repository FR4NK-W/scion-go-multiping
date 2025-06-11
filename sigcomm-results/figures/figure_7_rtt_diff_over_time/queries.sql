SELECT time_bucket('1 hour', ping_time_new) AS hour,
avg(pr.rtt) as avg_rtt,
COUNT(*) FILTER (WHERE pr.success = true AND pr.rtt > 0) * 1.0 / COUNT(*) AS success_ratio

FROM ping_results pr

WHERE pr.src_scion_addr NOT IN  ( '71-2:0:35,192.168.1.1:0') AND pr.success = true and pr.RTT > 0 
GROUP BY hour
ORDER BY hour


SELECT time_bucket('1 hour', ping_time_new) AS hour,
avg(pr.rtt) as avg_rtt

FROM ip_ping_results pr

WHERE pr.src_addr NOT IN  ( '192.168.1.1') AND pr.success = true and pr.RTT > 0
GROUP BY hour
ORDER BY hour


