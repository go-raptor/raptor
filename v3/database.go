package raptor

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type Migration func(*DB) error

type Migrations map[int]Migration

type DB struct {
	*bun.DB
	Connector  Connector
	Migrations Migrations
}

type SchemaMigration struct {
	ID         int       `bun:"id,pk,autoincrement"`
	Version    string    `bun:"version,notnull"`
	MigratedAt time.Time `bun:"migrated_at,notnull"`
	CreatedAt  time.Time `bun:"created_at,notnull"`
	UpdatedAt  time.Time `bun:"updated_at,notnull"`
}

func newDB(db Database) *DB {
	return &DB{
		Connector:  db.Connector,
		Migrations: db.Migrations,
	}
}

type Connector interface {
	Connect(config interface{}) (*bun.DB, error)
}

type Database struct {
	Connector  Connector
	Migrations Migrations
}

func (db *DB) migrate() error {
	_, err := db.NewCreateTable().
		Model((*SchemaMigration)(nil)).
		IfNotExists().
		Exec(context.Background())
	if err != nil {
		return err
	}

	var currentVersion int
	err = db.NewSelect().
		Model((*SchemaMigration)(nil)).
		ColumnExpr("COUNT(*)").
		Scan(context.Background(), &currentVersion)
	if err != nil {
		return err
	}

	for i := currentVersion + 1; i <= len(db.Migrations); i++ {
		funcName := strings.Split(runtime.FuncForPC(reflect.ValueOf(db.Migrations[i]).Pointer()).Name(), "/")
		migrationName := funcName[len(funcName)-1]

		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}

		err = db.Migrations[i](db)
		if err == nil {
			_, err = tx.NewInsert().
				Model(&SchemaMigration{
					Version:    migrationName,
					MigratedAt: time.Now(),
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}).
				Exec(context.Background())
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
