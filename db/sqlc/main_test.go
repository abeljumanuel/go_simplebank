package sqlc

import (
	"context"
	"log"
	"os"
	"testing"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbDriver = "postgres"
	dbSource = "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable"
)

var testQueries *Queries
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {

	var err error

	connPoolTest, err := pgxpool.New(context.Background(), dbSource)
	if err != nil {
		log.Fatal("No se puede conectar a la BD usando pgxPool:", err)
	}

	defer connPoolTest.Close()
		
	testDB = connPoolTest

	testQueries = New(testDB)

	exitCode := m.Run()

	os.Exit(exitCode)
}