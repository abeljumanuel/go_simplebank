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

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams(arg))
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

		// prevent deadlock by updating accounts in a fixed order
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		}
		if err != nil {
			return err
		}

		return nil
	})

	return result, err

}

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:		accountID1,
		Amount:	amount1,
	})
	if err != nil {
		return
	}
	
	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:		accountID2,
		Amount:	amount2,
	})
	if err != nil {
		return
	}
	return
}