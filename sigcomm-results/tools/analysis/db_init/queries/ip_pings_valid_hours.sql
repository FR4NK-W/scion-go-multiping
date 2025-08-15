SELECT
    date_trunc('hour', ping_time_new) AS hour,
	avg(rtt) as avg_rtt,
	COUNT(*) FILTER (WHERE success = true AND rtt > 0) * 1.0 / COUNT(*) AS success_ratio,
	COUNT(*) AS prcount,
    src_addr
FROM
    public.ip_ping_results
WHERE 
    rtt > 0
    AND success = true
    AND ping_time_new < '2025-02-08'
GROUP BY
    hour,
    src_addr
HAVING
    COUNT(*) > 12000
ORDER BY
    hour,
    src_addr
