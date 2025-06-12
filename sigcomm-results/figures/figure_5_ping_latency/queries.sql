WITH params AS (
    SELECT 0 AS rtt_min, 400 AS rtt_max, 400/10 AS bucket_count
),
 bucketed AS (
     SELECT
         width_bucket(rtt, p.rtt_min, p.rtt_max, p.bucket_count) AS rtt_bucket
     FROM ip_ping_results s, params p
     where s.success = true and s.rtt > 0 
 )
 SELECT
     rtt_bucket,
     p.rtt_min + (rtt_bucket - 1) * ((p.rtt_max - p.rtt_min) * 1.0 / p.bucket_count) AS lower_bound,
     COUNT(*) AS ping_count
 FROM bucketed b, params p
 GROUP BY rtt_bucket, lower_bound
 ORDER BY rtt_bucket;



 WITH params AS (
    SELECT 0 AS rtt_min, 400 AS rtt_max, 400/10 AS bucket_count
),
 bucketed AS (
     SELECT
         width_bucket(rtt, p.rtt_min, p.rtt_max, p.bucket_count) AS rtt_bucket
     FROM ping_results s, params p
     where s.success = true and s.rtt > 0
 )
 SELECT
     rtt_bucket,
     p.rtt_min + (rtt_bucket - 1) * ((p.rtt_max - p.rtt_min) * 1.0 / p.bucket_count) AS lower_bound,
     COUNT(*) AS ping_count
 FROM bucketed b, params p
 GROUP BY rtt_bucket, lower_bound
 ORDER BY rtt_bucket;