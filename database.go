package raptor

import (
	"reflect"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Migration func(*DB) error

type Migrations map[int]Migration

type DB struct {
	*gorm.DB
	Connector  Connector
	Migrations Migrations
}

type SchemaMigration struct {
	gorm.Model
	Version    string
	MigratedAt time.Time
}

func newDB(db Database) *DB {
	return &DB{
		Connector:  db.Connector,
		Migrations: db.Migrations,
	}
}

type Connector interface {
	Connect(config interface{}) (*gorm.DB, error)
}

type Database struct {
	Connector  Connector
	Migrations Migrations
}

func (db *DB) migrate() error {
	db.AutoMigrate(&SchemaMigration{})

	var currentVersion int64
	result := db.Find(&SchemaMigration{}).Count(&currentVersion)
	if result.Error != nil {
		return result.Error
	}

	for i := currentVersion + 1; i <= int64(len(db.Migrations)); i++ {
		funcName := strings.Split(runtime.FuncForPC(reflect.ValueOf(db.Migrations[int(i)]).Pointer()).Name(), "/")
		migrationName := funcName[len(funcName)-1]
		tx := db.Begin()
		err := db.Migrations[int(i)](db)
		if err == nil {
			result = db.Create(&SchemaMigration{Version: migrationName, MigratedAt: time.Now()})
			if result.Error != nil {
				tx.Rollback()
				return result.Error
			}
		} else {
			tx.Rollback()
			return err
		}
		tx.Commit()
	}

	return nil
}
