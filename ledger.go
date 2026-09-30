package main

import (
	"context"
	"fmt"
)

// recordCharge stores a charge and its ledger lines in a single transaction.
func (db *DB) recordCharge(ctx context.Context, id string, accountID int, total int64, items []int64) error {
	conn, err := db.acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO charges (id, account_id, amount) VALUES ($1, $2, $3)`,
		id, accountID, total); err != nil {
		return fmt.Errorf("insert charge: %w", err)
	}

	// All ledger lines go in one statement: one round trip, one index update pass.
	if _, err := tx.Exec(ctx,
		`INSERT INTO ledger_entries (charge_id, account_id, line, amount)
		 SELECT $1, $2, t.line, t.amount
		 FROM unnest($3::bigint[]) WITH ORDINALITY AS t(amount, line)`,
		id, accountID, items); err != nil {
		return fmt.Errorf("insert ledger entries: %w", err)
	}

	return tx.Commit(ctx)
}
