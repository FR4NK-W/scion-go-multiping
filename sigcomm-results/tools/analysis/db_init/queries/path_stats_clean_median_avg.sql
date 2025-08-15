/* Path Statistics aggregated with Median and Average Difference */
WITH PathGroupMax AS (
    -- First, calculate the maximum available_paths for each group (src/dst pair)
    -- and make it available to each row within that group.
    SELECT
        src_scion_addr,
        dst_scion_addr,
        available_paths,
        min_hops,
        max_hops,
        MAX(available_paths) OVER (PARTITION BY src_scion_addr, dst_scion_addr) AS max_paths_for_group
    FROM path_statistics
    WHERE available_paths > 0
      AND min_hops < 10000
      AND max_hops < 10000
      AND lookup_time_new < '2025-02-08'
      AND src_scion_addr NOT IN ('71-2:0:35,192.168.1.1:0')
      AND dst_scion_addr NOT IN ('64-2:0:9,129.132.230.98:30041', '71-2:0:18,192.168.1.1:30041', '71-88,127.0.0.1:30041')
)
-- Now, perform the final aggregation
SELECT
    src_scion_addr,
    dst_scion_addr,
    AVG(available_paths) AS avg_paths,
    MIN(available_paths) AS min_paths,
    MAX(available_paths) AS max_paths,
    -- Calculate the average of the difference between each row's path count and the group's maximum
    AVG(max_paths_for_group - available_paths) AS avg_diff_to_max,
    -- Calculate the median of the difference
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY max_paths_for_group - available_paths) AS median_diff_to_max,
    AVG(min_hops) AS avg_min_hops,
    AVG(max_hops) AS avg_max_hops
FROM PathGroupMax
GROUP BY src_scion_addr, dst_scion_addr
ORDER BY src_scion_addr, dst_scion_addr