package backup

import (
	"database/sql"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func openMySQLForTest(t *testing.T, host, port, user, password, database string) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", mysqlDSNForTest(host, port, user, password, database))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func execMySQL(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatal(err)
	}
}

func mysqlDSNForTest(host, port, user, password, database string) string {
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = host + ":" + port
	cfg.DBName = database
	cfg.ParseTime = true
	cfg.Params = map[string]string{"charset": "utf8mb4", "multiStatements": "true"}
	return cfg.FormatDSN()
}
