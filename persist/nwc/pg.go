package persist

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/breez/breez-lnurl/constant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgStore struct {
	pool *pgxpool.Pool
}

func NewPgStore(pool *pgxpool.Pool) *PgStore {
	return &PgStore{
		pool,
	}
}

func (s *PgStore) Set(ctx context.Context, webhook Webhook) error {
	userPubkey, err := hex.DecodeString(webhook.UserPubkey)
	if err != nil {
		return err
	}
	appPubkey, err := hex.DecodeString(webhook.AppPubkey)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var webhookId int64
	err = tx.QueryRow(
		ctx,
		`INSERT INTO public.nwc_webhooks (url, user_pubkey, app_pubkey, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_pubkey, app_pubkey) DO UPDATE SET url = $1, updated_at = NOW()
		 RETURNING id`,
		webhook.Url,
		userPubkey,
		appPubkey,
	).Scan(&webhookId)
	if err != nil {
		return fmt.Errorf("failed to insert/update webhook: %w", err)
	}

	relays, err := getRelaysByUrl(ctx, tx)
	if err != nil {
		return err
	}

	for _, relayUrl := range webhook.Relays {
		if _, exists := relays[relayUrl]; exists {
			continue
		}

		newRelayId := len(relays) % constant.NWC_MAX_RELAYS_LENGTH
		_, err = tx.Exec(
			ctx,
			`INSERT INTO public.nwc_relays (id, url) 
             VALUES ($1, $2) 
             ON CONFLICT (id) DO UPDATE SET url = EXCLUDED.url`,
			newRelayId, relayUrl,
		)
		if err != nil {
			return fmt.Errorf("failed to insert relay: %w", err)
		}
		_, err = tx.Exec(
			ctx,
			`INSERT INTO public.nwc_webhooks_relays (webhook_id, relay_id)
		 	 VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			webhookId,
			newRelayId,
		)
		if err != nil {
			return fmt.Errorf("failed to link webhook and relay: %w", err)
		}
		relays[relayUrl] = newRelayId
	}

	return tx.Commit(ctx)
}

func (s *PgStore) Get(ctx context.Context, userPubkey string, appPubkey string) (*Webhook, error) {
	userPubkeyBytes, err := hex.DecodeString(userPubkey)
	if err != nil {
		return nil, fmt.Errorf("invalid user pubkey: %w", err)
	}
	appPubkeyBytes, err := hex.DecodeString(appPubkey)
	if err != nil {
		return nil, fmt.Errorf("invalid app pubkey: %w", err)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var webhookId int64
	var url string
	err = tx.QueryRow(
		ctx,
		`SELECT id, url 
		 FROM public.nwc_webhooks 
		 WHERE user_pubkey = $1 AND app_pubkey = $2`,
		userPubkeyBytes,
		appPubkeyBytes,
	).Scan(&webhookId, &url)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying webhook: %w", err)
	}

	rows, err := tx.Query(
		ctx,
		`SELECT nr.url
		 FROM public.nwc_webhooks_relays nwr
		 INNER JOIN public.nwc_relays nr ON nwr.relay_id = nr.id
		 WHERE nwr.webhook_id = $1`,
		webhookId,
	)
	if err != nil {
		return nil, fmt.Errorf("querying relays: %w", err)
	}
	defer rows.Close()

	var relays []string
	for rows.Next() {
		var relayUrl string
		if err := rows.Scan(&relayUrl); err != nil {
			return nil, err
		}
		relays = append(relays, relayUrl)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &Webhook{
		Relays:     relays,
		AppPubkey:  appPubkey,
		UserPubkey: userPubkey,
		Url:        url,
	}, nil
}

func (s *PgStore) Delete(ctx context.Context, userPubkey string, appPubkey string) error {
	_, err := s.pool.Exec(
		ctx,
		`DELETE FROM public.nwc_webhooks WHERE user_pubkey = $1 AND app_pubkey = $2`,
		userPubkey,
		appPubkey,
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *PgStore) GetAppPubkeys(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT encode(app_pubkey, 'hex') app_pubkey FROM public.nwc_webhooks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rowsToArray(rows), nil
}

func (s *PgStore) GetRelays(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT url FROM public.nwc_relays`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rowsToArray(rows), nil
}

func (s *PgStore) DeleteExpired(ctx context.Context, before time.Time) error {
	beforeUnix := before.Unix()
	_, err := s.pool.Exec(
		ctx,
		`DELETE FROM public.nwc_webhooks
		 WHERE updated_at < to_timestamp($1)`,
		beforeUnix)
	return err
}

func getRelaysByUrl(ctx context.Context, con pgx.Tx) (map[string]int, error) {
	rows, err := con.Query(ctx, `SELECT id, url FROM public.nwc_relays`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id int
	var url string
	relays := make(map[string]int)
	for rows.Next() {
		err := rows.Scan(&id, &url)
		if err != nil {
			return nil, err
		}
		relays[url] = id
	}
	return relays, nil
}

func rowsToArray(rows pgx.Rows) []string {
	arr := []string{}
	for rows.Next() {
		var val string
		rows.Scan(&val)
		arr = append(arr, val)
	}
	return arr
}
