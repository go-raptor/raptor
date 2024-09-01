package raptor

import (
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Migration func(*DB) error

type Migrations map[int]Migration

type DB struct {
	*sqlx.DB
	Connector  Connector
	Migrations Migrations
}

type SchemaMigration struct {
	ID         int       `db:"id"`
	Version    string    `db:"version"`
	MigratedAt time.Time `db:"migrated_at"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func newDB(db Database) *DB {
	return &DB{
		Connector:  db.Connector,
		Migrations: db.Migrations,
	}
}

type Connector interface {
	Connect(config interface{}) (*sqlx.DB, error)
}

type Database struct {
	Connector  Connector
	Migrations Migrations
}

func (db *DB) migrate() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			version VARCHAR(255) NOT NULL,
			migrated_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	var currentVersion int
	err = db.Get(&currentVersion, "SELECT COUNT(*) FROM schema_migrations")
	if err != nil {
		return err
	}

	for i := currentVersion + 1; i <= len(db.Migrations); i++ {
		funcName := strings.Split(runtime.FuncForPC(reflect.ValueOf(db.Migrations[i]).Pointer()).Name(), "/")
		migrationName := funcName[len(funcName)-1]

		tx, err := db.Beginx()
		if err != nil {
			return err
		}

		err = db.Migrations[i](db)
		if err == nil {
			_, err = tx.Exec("INSERT INTO schema_migrations (version, migrated_at, created_at, updated_at) VALUES ($1, $2, $2, $2)",
				migrationName, time.Now())
			if err != nil {
				tx.Rollback()
				return err
			}
		} else {
			tx.Rollback()
			return err
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}
