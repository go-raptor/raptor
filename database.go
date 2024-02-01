package raptor

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDatabase(db *Database) (*gorm.DB, error) {
	switch db.Type {
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", db.Host, db.Username, db.Password, db.Name, db.Port)
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case "sqlite":
		return gorm.Open(sqlite.Open(db.Name), &gorm.Config{})
	}

	return nil, nil
}
