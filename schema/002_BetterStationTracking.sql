UPDATE "public"."live_station_availability" SET
    bikes_available = bikes_available - COALESCE(ebikes_available, 0),
    bikes_disabled = CASE
        WHEN bikes_disabled IS NULL THEN NULL
        ELSE                             bikes_disabled - COALESCE(ebikes_disabled, 0)
    END;

ALTER TABLE "public"."station" ADD COLUMN "capacity" INTEGER NULL;

---- create above / drop below ----

ALTER TABLE "public"."station" DROP COLUMN "capacity";

UPDATE "public"."live_station_availability" SET
    bikes_available = bikes_available + COALESCE(ebikes_available, 0),
    bikes_disabled = CASE
         WHEN bikes_disabled IS NULL THEN NULL
         ELSE                             bikes_disabled + COALESCE(ebikes_disabled, 0)
    END;
