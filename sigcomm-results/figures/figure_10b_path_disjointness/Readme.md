# Figure 10c - Simulated Link Failures

At first, store the output of this query into `disjointness.csv`:

```sql
WITH RankedPaths AS (
    -- First, rank each path within a src/dst group by the length of the 'paths' string in descending order
    SELECT
        src_scion_addr,
        dst_scion_addr,
        paths,
        ROW_NUMBER() OVER(PARTITION BY src_scion_addr, dst_scion_addr ORDER BY LENGTH(paths) DESC) as rn
    FROM
        path_statistics
    WHERE
        available_paths > 0
        AND min_hops < 10000
        AND max_hops < 10000
        AND lookup_time_new < '2025-02-08'
        AND src_scion_addr NOT IN ('71-2:0:35,192.168.1.1:0')
        AND dst_scion_addr NOT IN ('64-2:0:9,129.132.230.98:30041', '71-2:0:18,192.168.1.1:30041', '71-88,127.0.0.1:30041')
)
-- Now, select only the rows where the rank is 1 (the longest path for each group)
SELECT
    src_scion_addr,
    dst_scion_addr,
    paths
FROM
    RankedPaths
WHERE
    rn < 3
ORDER BY
    src_scion_addr, dst_scion_addr;
```


Then generate the plot via `new_path_disjointness.py`.