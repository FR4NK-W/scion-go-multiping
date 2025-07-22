WITH 
-- Step 1: Define the parameters for RTT bucketing.
params AS (
    SELECT 
        0 AS rtt_min, 
        400 AS rtt_max, 
        400/10 AS bucket_count
),

-- Step 2: For each successful ping, determine its one-minute time slot and its RTT bucket.
minute_bucketed_pings AS (
    SELECT
        time_bucket('1 minute', ping_time_new) AS minute,
        width_bucket(rtt, p.rtt_min, p.rtt_max, p.bucket_count) AS rtt_bucket
    FROM 
        ip_ping_results s, params p
    WHERE 
        s.success = true AND s.rtt > 0  and s.src_addr NOT IN   ( '192.168.1.1') AND s.ping_time_new < '2025-02-08'
),

-- Step 3: Group by minute and RTT bucket to get the count for each bucket.
-- Also, use a window function to calculate the total pings for the entire minute (prcount).
minute_counts AS (
    SELECT
        minute,
        rtt_bucket,
        COUNT(*) as ping_count,
        SUM(COUNT(*)) OVER (PARTITION BY minute) AS prcount
    FROM 
        minute_bucketed_pings
    GROUP BY 
        minute, rtt_bucket
)

-- Step 4: Final selection to calculate the lower bound of each RTT bucket and display all the required columns.
SELECT
    mc.minute,
    mc.prcount,
    mc.rtt_bucket,
    p.rtt_min + (mc.rtt_bucket - 1) * ((p.rtt_max - p.rtt_min) * 1.0 / p.bucket_count) AS lower_bound,
    mc.ping_count
FROM 
    minute_counts mc, params p
ORDER BY 
    mc.minute, mc.rtt_bucket;


-- -----------------------------------------------------------------

WITH 
-- Step 1: Define the parameters for RTT bucketing.
params AS (
    SELECT 
        0 AS rtt_min, 
        400 AS rtt_max, 
        400/10 AS bucket_count
),

-- Step 2: For each successful ping, determine its one-minute time slot and its RTT bucket.
minute_bucketed_pings AS (
    SELECT
        time_bucket('1 minute', ping_time_new) AS minute,
        width_bucket(rtt, p.rtt_min, p.rtt_max, p.bucket_count) AS rtt_bucket
    FROM 
        ping_results s, params p
    WHERE 
        s.success = true AND s.rtt > 0  and s.src_scion_addr NOT IN   ( '71-2:0:35,192.168.1.1:0') AND s.ping_time_new < '2025-02-08'
),

-- Step 3: Group by minute and RTT bucket to get the count for each bucket.
-- Also, use a window function to calculate the total pings for the entire minute (prcount).
minute_counts AS (
    SELECT
        minute,
        rtt_bucket,
        COUNT(*) as ping_count,
        SUM(COUNT(*)) OVER (PARTITION BY minute) AS prcount
    FROM 
        minute_bucketed_pings
    GROUP BY 
        minute, rtt_bucket
)

-- Step 4: Final selection to calculate the lower bound of each RTT bucket and display all the required columns.
SELECT
    mc.minute,
    mc.prcount,
    mc.rtt_bucket,
    p.rtt_min + (mc.rtt_bucket - 1) * ((p.rtt_max - p.rtt_min) * 1.0 / p.bucket_count) AS lower_bound,
    mc.ping_count
FROM 
    minute_counts mc, params p
ORDER BY 
    mc.minute, mc.rtt_bucket;



-- -----------------------------------------------------------------
-- Test to generate missing slots for IP/SCION
SELECT 
    all_minutes.minute,
    all_addrs.src_addr
FROM 
    -- Generate all minutes in the desired range
    (SELECT generate_series(
        (SELECT MIN(ping_time_new) FROM ip_ping_results WHERE ping_time_new < '2025-02-08'),
        (SELECT MAX(ping_time_new) FROM ip_ping_results WHERE ping_time_new < '2025-02-08'),
        '1 minute'::interval
     ) AS minute) AS all_minutes
CROSS JOIN
    -- Get all distinct source addresses
    (SELECT DISTINCT src_addr 
     FROM ip_ping_results 
     WHERE src_addr NOT IN ('192.168.1.1')) AS all_addrs
WHERE 
    -- Now, check for the non-existence of a successful ping for this combination
    NOT EXISTS (
        SELECT 1
        FROM ip_ping_results pr
        WHERE 
            pr.success = true 
            AND pr.rtt > 0
            -- Correlate the subquery with the outer query's combination
            AND pr.src_addr = all_addrs.src_addr
            AND time_bucket('1 minute', pr.ping_time_new) = all_minutes.minute
    )
ORDER BY 
    all_minutes.minute, all_addrs.src_addr;