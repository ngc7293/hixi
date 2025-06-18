CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE "public"."station"
(
    "id"                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "external_id"          TEXT NOT NULL,
    "name"                 TEXT,
    "location"             GEOMETRY(POINT, 4326),
    "last_status_reported" TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX "idx_station_external_id_uniq" ON "public"."station" ("external_id");

CREATE TABLE "public"."live_station_availability"
(
    "time"             TIMESTAMP WITH TIME ZONE NOT NULL,
    "station_id"       BIGINT                   NOT NULL,
    "bikes_available"  INTEGER,
    "bikes_disabled"   INTEGER,
    "ebikes_available" INTEGER,
    "ebikes_disabled"  INTEGER,
    "docks_available"  INTEGER,
    "docks_disabled"   INTEGER,

    FOREIGN KEY ("station_id") REFERENCES "public"."station" ("id") ON DELETE CASCADE
);
SELECT CREATE_HYPERTABLE('public.live_station_availability'::REGCLASS, 'time');

CREATE MATERIALIZED VIEW "public"."historical_station_availability"
    WITH (timescaledb.continuous) AS
SELECT TIME_BUCKET(INTERVAL '5 minutes', "time") AS time_bucket,
   "station_id",
   AVG("bikes_available")                    AS bikes_available,
   AVG("bikes_disabled")                     AS bikes_disabled,
   AVG("ebikes_available")                   AS ebikes_available,
   AVG("ebikes_disabled")                    AS ebikes_disabled,
   AVG("docks_available")                    AS docks_available,
   AVG("docks_disabled")                     AS docks_disabled
FROM "public"."live_station_availability"
GROUP BY "time_bucket", "station_id"
WITH NO DATA;

SELECT ADD_CONTINUOUS_AGGREGATE_POLICY(
   CONTINUOUS_AGGREGATE => 'public.historical_station_availability'::REGCLASS,
   START_OFFSET => '90 minutes'::INTERVAL,
   END_OFFSET => '30 minutes'::INTERVAL,
   SCHEDULE_INTERVAL => '15 minutes'::INTERVAL
);

SELECT ADD_RETENTION_POLICY(
  RELATION => 'public.live_station_availability'::REGCLASS,
  DROP_AFTER => '1 day'::INTERVAL,
  SCHEDULE_INTERVAL => '1 day'::INTERVAL
);

---- create above / drop below ----

DROP INDEX "idx_station_external_id";
DROP TABLE "public"."station";

SELECT REMOVE_RETENTION_POLICY('public.live_station_availability'::REGCLASS);
SELECT REMOVE_CONTINUOUS_AGGREGATE_POLICY('public.historical_station_availability'::REGCLASS);
DROP MATERIALIZED VIEW "public"."historical_station_availability";
DROP TABLE "public"."live_station_availability";

DROP EXTENSION postgis;
DROP EXTENSION timescaledb CASCADE;