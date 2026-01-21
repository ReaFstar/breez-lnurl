-- Alter webhooks "updated_at" type from bigint (milliseconds since epoch) to timestamp
ALTER TABLE public.nwc_webhooks ALTER COLUMN updated_at TYPE timestamp without time zone USING to_timestamp(updated_at / 1000) AT TIME ZONE 'UTC';
