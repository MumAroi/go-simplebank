package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

const (
	dbDriver = "postgres"
	dbSource = "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable"
)

var testQueries *Queries

func TestMain(m *testing.M) {
	conn, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("can not connect to db:", err)
	}

	if err := conn.Ping(); err != nil {
		log.Fatal("can not ping db:", err)
	}

	testQueries = New(conn)

	code := m.Run()
	conn.Close()

	os.Exit(code)
}
