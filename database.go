package raptor

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Migration func(*DB) error

type Migrations map[int]Migration

type DB struct {
	*gorm.DB
	Migrations Migrations
}

type SchemaMigration struct {
	gorm.Model
	Version    string
	MigratedAt time.Time
}

func newDB(migrations Migrations) *DB {
	return &DB{
		Migrations: migrations,
	}
}

func (db *DB) connect(config *DatabaseConfig) error {
	var err error
	switch config.Type {
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", config.Host, config.Username, config.Password, config.Name, config.Port)
		db.DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case "sqlite":
		db.DB, err = gorm.Open(sqlite.Open(config.Name), &gorm.Config{})
	}

	return err
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
