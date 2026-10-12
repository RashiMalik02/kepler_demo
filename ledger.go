package main

import (
	"context"
	"fmt"
)

// recordCharge stores a charge and its ledger lines in a single transaction.
// Each ledger line carries the account's running balance so statements can be
// rendered without re-summing the ledger.
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

	for i, amount := range items {
		var balance int64
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(SUM(amount), 0) FROM ledger_entries WHERE account_id = $1`,
			accountID).Scan(&balance); err != nil {
			return fmt.Errorf("read running balance: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO ledger_entries (charge_id, account_id, line, amount, balance)
			 VALUES ($1, $2, $3, $4, $5)`,
			id, accountID, i+1, amount, balance+amount); err != nil {
			return fmt.Errorf("insert ledger entry %d: %w", i+1, err)
		}
	}

	return tx.Commit(ctx)
}
