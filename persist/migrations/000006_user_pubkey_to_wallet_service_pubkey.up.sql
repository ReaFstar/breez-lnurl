-- Rename user_pubkey to wallet_service_pubkey in nwc_webhooks
ALTER TABLE public.nwc_webhooks RENAME COLUMN user_pubkey TO wallet_service_pubkey;

-- Rename user_pubkey to wallet_service_pubkey in nwc_forwarded_events
ALTER TABLE public.nwc_forwarded_events RENAME COLUMN user_pubkey TO wallet_service_pubkey;
