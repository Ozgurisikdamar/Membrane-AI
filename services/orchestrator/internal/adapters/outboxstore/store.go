// Package outboxstore implements the transactional-outbox pattern (D-013) for
// verdicts: Publish writes the verdict_audit row AND the outbox row in one
// ACID transaction, and Relay (relay.go) ships pending outbox rows to the bus.
// The store implements ports.VerdictPublisher, so the Saga is wired to it
// unchanged — durability first, delivery asynchronously.
package outboxstore

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/codec"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

const op = "orchestrator.adapters.outboxstore"

// aggregate tags outbox rows written by this store.
const aggregate = "verdict"

// Store persists verdicts and their outgoing events atomically.
type Store struct {
	pool  *pgxpool.Pool
	topic string
}

// New connects a pool. topic is the destination the relay will publish to.
func New(ctx context.Context, databaseURL, topic string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, errs.Unavailable(op, "connect postgres", err)
	}
	return &Store{pool: pool, topic: topic}, nil
}

// Publish implements ports.VerdictPublisher: one transaction writes the audit
// row and enqueues the wire payload in the outbox. The relay delivers it.
func (s *Store) Publish(ctx context.Context, v domain.Verdict) error {
	payload, err := codec.EncodeVerdict(v)
	if err != nil {
		return err
	}
	detail, err := json.Marshal(map[string]any{
		"ruleset_version": v.RulesetVersion,
		"findings":        len(v.Findings),
	})
	if err != nil {
		return errs.Internal(op, "marshal audit detail", err)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return errs.Unavailable(op, "begin tx", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`INSERT INTO verdict_audit (submission_id, org_id, verdict, source, detail)
		 VALUES ($1, $2, $3, $4, $5)`,
		v.SubmissionID, v.OrganizationID, string(v.Decision), string(v.Source), detail,
	); err != nil {
		return errs.Unavailable(op, "insert verdict_audit", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO outbox (aggregate, topic, partition_key, payload)
		 VALUES ($1, $2, $3, $4)`,
		aggregate, s.topic, v.OrganizationID, payload,
	); err != nil {
		return errs.Unavailable(op, "insert outbox", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return errs.Unavailable(op, "commit tx", err)
	}
	return nil
}

// PublishPending delivers up to batch pending outbox rows through publish and
// marks them published — all inside one transaction using FOR UPDATE SKIP
// LOCKED, so concurrent relays (or orchestrator replicas) never double-claim a
// row. Delivery is at-least-once: if the commit fails after a successful
// produce, the row is retried — consumers must dedupe (they key on
// submission_id). Returns the number of rows shipped.
func (s *Store) PublishPending(ctx context.Context, batch int, publish func(ctx context.Context, topic string, key, value []byte) error) (int, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, errs.Unavailable(op, "begin relay tx", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx,
		`SELECT id, topic, partition_key, payload
		 FROM outbox
		 WHERE published_at IS NULL
		 ORDER BY id
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`, batch)
	if err != nil {
		return 0, errs.Unavailable(op, "select pending", err)
	}

	type pending struct {
		id      int64
		topic   string
		key     string
		payload []byte
	}
	var claimed []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.topic, &p.key, &p.payload); err != nil {
			rows.Close()
			return 0, errs.Internal(op, "scan pending", err)
		}
		claimed = append(claimed, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, errs.Unavailable(op, "iterate pending", err)
	}
	if len(claimed) == 0 {
		return 0, tx.Commit(ctx)
	}

	ids := make([]int64, 0, len(claimed))
	for _, p := range claimed {
		if err := publish(ctx, p.topic, []byte(p.key), p.payload); err != nil {
			// Roll back: rows stay pending and are retried next tick.
			return 0, errs.Unavailable(op, "relay publish", err)
		}
		ids = append(ids, p.id)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE outbox SET published_at = now() WHERE id = ANY($1)`, ids); err != nil {
		return 0, errs.Unavailable(op, "mark published", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, errs.Unavailable(op, "commit relay tx", err)
	}
	return len(ids), nil
}

// Ping checks connectivity for readiness probes.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return errs.Unavailable(op, "ping postgres", err)
	}
	return nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }
