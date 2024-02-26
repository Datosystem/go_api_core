package datamanager

import (
	"log"
	"os"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

type Seeder struct {
	ID   string
	Seed func(*gorm.DB) error
}

type DataManager struct {
	Options *gormigrate.Options
	Models  []interface{}
	Before  []*gormigrate.Migration
	After   []*gormigrate.Migration
	Seeders []*Seeder
}

func (dm DataManager) BeforeMigrate(db *gorm.DB) error {
	if len(dm.Before) > 0 {
		start := time.Now()
		log.Printf("BeforeMigrate start\n")
		m := gormigrate.New(db, dm.Options, dm.Before)
		if err := m.Migrate(); err != nil {
			log.Printf("BeforeMigrate error %v\n", err)
			return err
		}
		log.Printf("BeforeMigrate done in %v\n", time.Since(start))
	}
	return nil
}

func (dm DataManager) Migrate(db *gorm.DB) error {
	start := time.Now()
	log.Printf("Migrate start\n")
	err := db.AutoMigrate(dm.Models...)
	if err != nil {
		log.Printf("Migrate error %v\n", err)
		return err
	}
	log.Printf("Migrate done in %v\n", time.Since(start))
	return nil
}

func (dm DataManager) AfterMigrate(db *gorm.DB) error {
	if len(dm.After) > 0 {
		start := time.Now()
		log.Printf("AfterMigrate start\n")
		m := gormigrate.New(db, dm.Options, dm.After)
		if err := m.Migrate(); err != nil {
			log.Printf("AfterMigrate error %v\n", err)
			return err
		}
		log.Printf("AfterMigrate done in %v\n", time.Since(start))
	}
	return nil
}

func (dm DataManager) SkipAfter(db *gorm.DB, ID string) {
	// TODO: Skip the 'before' when the DB has been created 100% by the APIs
	if len(ID) > 0 {
		for _, mod := range dm.After {
			if mod.ID == ID {
				log.Println("ID: " + mod.ID)
				db.Exec(`INSERT INTO `+dm.Options.TableName+` (id) VALUES(?)`, mod.ID)
			}
		}
	} else {
		for _, mod := range dm.After {
			log.Printf(mod.ID)
			db.Exec(`INSERT INTO `+dm.Options.TableName+` (id) VALUES(?)`, mod.ID)
		}
	}
}

func (dm DataManager) Seed(db *gorm.DB, ids []string) {
	for _, seeder := range dm.Seeders {
		for _, id := range ids {
			if seeder.ID == id {
				seeder.Seed(db)
			}
		}
	}
}

func (dm DataManager) Apply(db *gorm.DB) {
	if os.Getenv("ENV_TYPE") == "DEVELOPMENT" {
		log.Println("Skipping the migrations because the ENV_TYPE is set to DEVELOPMENT")
		return
	}
	err := dm.BeforeMigrate(db)
	if err != nil {
		return
	}
	err = dm.Migrate(db)
	if err != nil {
		return
	}
	err = dm.AfterMigrate(db)
	if err != nil {
		return
	}
}

func (dm DataManager) RollbackTo(db *gorm.DB, migrationID string) error {
	m := gormigrate.New(db, dm.Options, dm.Before)
	return m.RollbackTo(migrationID)
}

func (dm DataManager) RollbackLast(db *gorm.DB) error {
	m := gormigrate.New(db, dm.Options, dm.Before)
	return m.RollbackLast()
}
