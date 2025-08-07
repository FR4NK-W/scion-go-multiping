SELECT time_bucket('1 hour', ping_time_new) AS hour, pr.src_scion_addr, pr.dst_scion_addr, avg(pr.rtt) FROM ping_results pr

WHERE pr.src_scion_addr NOT IN  ( '71-2:0:35,192.168.1.1:0') 
		AND pr.dst_scion_addr NOT IN ('71-225,127.0.0.1:30041') -- filter out SCION pings towards UVa as we did not send any IP pings either.
        AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new < '2025-02-08'
GROUP BY hour, pr.src_scion_addr, pr.dst_scion_addr



SELECT time_bucket('1 hour', ping_time_new) AS hour,  pr.src_addr, pr.dst_addr, avg(pr.rtt) FROM ip_ping_results pr

WHERE pr.src_addr NOT IN  ( '192.168.1.1') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new < '2025-02-08'
GROUP BY hour, pr.src_addr, pr.dst_addr