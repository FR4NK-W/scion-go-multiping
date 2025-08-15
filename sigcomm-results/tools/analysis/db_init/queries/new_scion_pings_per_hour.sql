
WITH 
-- Step 1: Define the parameters for RTT bucketing. This part remains unchanged.
params AS (
    SELECT 
        0 AS rtt_min, 
        400 AS rtt_max, 
        400/10 AS bucket_count
),

-- Step 2: For each successful ping, determine its one-HOUR time slot, its RTT bucket, and its source address.
hourly_bucketed_pings AS (
    SELECT
        time_bucket('1 hour', ping_time_new) AS hour, -- CHANGED: from '1 minute' to '1 hour' and renamed to 'hour'
        s.src_scion_addr,                                   -- ADDED: include the source address
        width_bucket(rtt, p.rtt_min, p.rtt_max, p.bucket_count) AS rtt_bucket
    FROM 
        ping_results s, params p
    WHERE 
        s.success = true 
        AND s.rtt > 0  
        AND s.src_scion_addr NOT IN ('71-2:0:35,192.168.1.1:0') 
		AND s.dst_scion_addr NOT IN ('71-225,127.0.0.1:30041') -- remove pings towards UVa as we do not send IP pings to UVa either.
        AND s.ping_time_new < '2025-02-08'
),

-- Step 3: Group by hour, source address, and RTT bucket to get the count for each bucket.
-- The window function now calculates prcount for each specific hour-address combination.
hourly_counts AS (
    SELECT
        hour,
        src_scion_addr,                                     -- ADDED: include src_addr in selection
        rtt_bucket,
        COUNT(*) as ping_count,
        -- CHANGED: The window now partitions by both hour and src_addr
        SUM(COUNT(*)) OVER (PARTITION BY hour, src_scion_addr) AS prcount 
    FROM 
        hourly_bucketed_pings
    GROUP BY 
        hour, src_scion_addr, rtt_bucket                    -- ADDED: src_addr to the GROUP BY clause
)

-- Step 4: Final selection to calculate the lower bound of each RTT bucket and display all the required columns.
SELECT
    hc.hour,
    hc.src_scion_addr,                                      -- ADDED: show src_addr in the final output
    hc.prcount,
    hc.rtt_bucket,
    p.rtt_min + (hc.rtt_bucket - 1) * ((p.rtt_max - p.rtt_min) * 1.0 / p.bucket_count) AS lower_bound,
    hc.ping_count
FROM 
    hourly_counts hc, params p
ORDER BY 
    hc.hour, hc.src_scion_addr, hc.rtt_bucket;              -- CHANGED: updated the ordering for clarity