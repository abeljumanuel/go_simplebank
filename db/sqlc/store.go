package sqlc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute db pgx queries and transactions
type Store struct {
	conn *pgxpool.Pool
	*Queries
}

// NewStore creates a new Store
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		conn: 	 pool,
		Queries: New(pool),
	}
}

// execTx executes a function within a database transaction
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.conn.BeginTx(ctx, pgx.TxOptions{
        IsoLevel: pgx.ReadCommitted,
    })
	if err != nil {
		return  err
	}

	q := New(tx)
	err = fn(q)

	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit(ctx)
}

// TransferTxParams contains the input parameters of the transfer transaction
type TransferTxParams struct {
	FromAccountID 	int64 		`json:"from_account_id"`
	ToAccountID		int64 		`json:"to_account_id"`
	Amount			int64 		`json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct {
	Transfer		Transfer	`json:"transfer"`
	FromAccount		Account		`json:"from_account"`
	ToAccount		Account		`json:"to_account"`
	FromEntry		Entry		`json:"from_entry"`
	ToEntry			Entry		`json:"to_entry"`
}

// TransferTX performs a money transfer from one account to the other
// It creates the transfer, add account entries, and update accounts' balance within a database transaction
func (store *Store) TransferTX(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID:	arg.FromAccountID,
			ToAccountID: 	arg.ToAccountID,
			Amount: 		arg.Amount,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: 		arg.FromAccountID,
			Amount: 		-arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: 		arg.ToAccountID,
			Amount: 		arg.Amount,
		})
		if err != nil {
			return err
		}

		// get account -> update its balance
		// account1
		result.FromAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:			arg.FromAccountID,
			Amount: 	-arg.Amount,
		})
		if err != nil {
			return err
		}
		// account2
		result.ToAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:			arg.ToAccountID,
			Amount: 	arg.Amount,
		})
		if err != nil {
			return err
		}

		return nil
	})

	return result, err

}

