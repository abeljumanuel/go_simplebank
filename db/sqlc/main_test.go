package sqlc

import (
	"context"
	"log"
	"os"
	"testing"
	"github.com/jackc/pgx/v5"
)

const (
	dbDriver = "postgres"
	dbSource = "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable"
)

var testQueries *Queries
var testDB *pgx.Conn

func TestMain(m *testing.M) {
	conn, err := pgx.Connect(context.Background(), dbSource)
	if err != nil {
		log.Fatal("No se puede conectar a la BD usando pgx:", err)
	}

	testDB = conn

	testQueries = New(testDB)

	os.Exit(m.Run())
}