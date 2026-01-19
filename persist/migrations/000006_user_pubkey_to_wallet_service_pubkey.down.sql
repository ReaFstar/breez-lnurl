-- Rename wallet_service_pubkey back to user_pubkey in nwc_webhooks
ALTER TABLE public.nwc_webhooks RENAME COLUMN wallet_service_pubkey TO user_pubkey;

-- Rename wallet_service_pubkey back to user_pubkey in nwc_forwarded_events
ALTER TABLE public.nwc_forwarded_events RENAME COLUMN wallet_service_pubkey TO user_pubkey;
