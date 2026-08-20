package db

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func RunInTransaction(fn func(pgx.Tx) error) error {
	tx, err := DB.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer tx.Rollback(context.Background())

	err = fn(tx)
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func RunInTransactionWithReturn(fn func(pgx.Tx) (any, error)) (any, error) {
	tx, err := DB.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	val, err := fn(tx)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return nil, err
	}
	return val, nil
}
