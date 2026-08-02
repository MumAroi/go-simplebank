package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/MumAroi/go-simplebank/util"
	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("can not load config:", err)
	}

	testDB, err = sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("can not connect to db:", err)
	}

	if err = testDB.Ping(); err != nil {
		log.Fatal("can not ping db:", err)
	}

	testQueries = New(testDB)

	code := m.Run()
	testDB.Close()

	os.Exit(code)
}
