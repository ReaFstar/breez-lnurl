-- Alter webhooks "updated_at" type from timestamp back to bigint (milliseconds since epoch)
ALTER TABLE public.nwc_webhooks ALTER COLUMN updated_at TYPE bigint USING (EXTRACT(EPOCH FROM updated_at) * 1000)::bigint;
