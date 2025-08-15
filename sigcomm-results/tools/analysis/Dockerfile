# repo/sigcomm-results/Dockerfile
FROM timescale/timescaledb-ha:pg17
COPY db_init/ /docker-entrypoint-initdb.d/
RUN set -eux; \
    find /docker-entrypoint-initdb.d -type f -name "*.sh" -exec chmod +x {} \; ; \
    true