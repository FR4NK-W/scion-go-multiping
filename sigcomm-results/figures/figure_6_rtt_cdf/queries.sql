SELECT pr.src_scion_addr, pr.dst_scion_addr, avg(pr.rtt) FROM ping_results pr

WHERE pr.src_scion_addr NOT IN  ( '71-2:0:35,192.168.1.1:0') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new
GROUP BY pr.src_scion_addr, pr.dst_scion_addr

SELECT pr.src_addr, pr.dst_addr, avg(pr.rtt) FROM ip_ping_results pr

WHERE pr.src_addr NOT IN  ( '192.168.1.1') AND pr.success = true and pr.RTT > 0 AND pr.ping_time_new 
GROUP BY pr.src_addr, pr.dst_addr