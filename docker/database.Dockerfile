FROM postgres:17-bookworm

# Set environment variables to avoid prompts
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
  && apt-get install -y --no-install-recommends curl ca-certificates  \
  && echo "deb https://packagecloud.io/timescale/timescaledb/debian/ bookworm main" | tee /etc/apt/sources.list.d/timescaledb.list \
  && curl -fsSL https://packagecloud.io/timescale/timescaledb/gpgkey | gpg --dearmor -o /etc/apt/trusted.gpg.d/timescaledb.gpg \
  && apt-get update \
  && apt-get install -y --no-install-recommends \
      postgresql-17-postgis-3 \
      postgresql-17-postgis-3-scripts \
      timescaledb-2-postgresql-17 \
      timescaledb-tools \
  && apt-get remove -y curl ca-certificates \
  && rm -rf /var/lib/apt/lists/*

RUN echo "#!/bin/bash \n\
timescaledb-tune --quiet --yes \n\
echo \"shared_preload_libraries = 'timescaledb'\" >> /var/lib/postgresql/data/postgresql.conf \n\
" > /docker-entrypoint-initdb.d/00-init-timescale.sh
