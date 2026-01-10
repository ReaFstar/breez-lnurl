-- Table to track events that have been forwarded to Breez-Notify
-- This ensures events are forwarded once and only once
CREATE TABLE public.nwc_forwarded_events (
  event_id varchar(64) PRIMARY KEY,
  user_pubkey bytea NOT NULL,
  app_pubkey bytea NOT NULL,
  forwarded_at timestamp NOT NULL DEFAULT NOW(),
  webhook_url varchar NOT NULL
);

-- Index for cleanup queries (events older than X days)
CREATE INDEX nwc_forwarded_events_forwarded_at_idx ON public.nwc_forwarded_events (forwarded_at);
