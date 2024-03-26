package main

import (
	"example/models"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/shockerli/cvt"
	"github.com/vlkzh/gormup"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	log.Info("start")

	db, err := createDB()
	if err != nil {
		panic(err)
	}

	err = initDB(db)
	if err != nil {
		panic(err)
	}

	err = change(db)
	if err != nil {
		panic(err)
	}

	log.Info("done")
}

func initDB(db *gorm.DB) (err error) {

	err = db.Exec("CREATE SCHEMA IF NOT EXISTS docs;").Error
	if err != nil {
		return err
	}

	mg := db.Migrator()
	if mg.HasTable(&models.Document{}) {
		return nil
	}

	err = mg.AutoMigrate(
		&models.Document{},
		&models.DocDetails{},
		&models.Field{},
	)
	if err != nil {
		return err
	}

	doc := &models.Document{
		Name:   "test",
		Number: 1234,
		ContactInfo: models.ContactInfo{
			Address: "test",
			Point:   []int{1, 2},
			Phone:   "123456",
		},
		Meta: models.Meta{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
		DocType: models.DocType{
			Type: "contract",
			Code: "buy",
		},
		Details: &models.DocDetails{
			Printable: true,
			Published: true,
			Comment:   "not available comment",
		},
		Fields: []*models.Field{
			{
				Key:   "key1",
				Value: "value1",
			},
			{
				Key:   "key2",
				Value: "value2",
			},
		},
	}

	err = db.Session(&gorm.Session{
		//FullSaveAssociations: true,
	}).Save(doc).Error
	if err != nil {
		return err
	}

	return nil
}

func change(db *gorm.DB) (err error) {

	gormup.Register(db, gormup.Config{
		OtherPrimaryKeys: map[string][]string{
			"docs.documents": {"name"},
		},
	})

	//db = internal.WithoutQueryCache(db)
	//db = internal.WithoutReduceUpdate(db)

	var doc1 *models.Document
	err = db.Session(&gorm.Session{}).Find(&doc1, "id = ?", 1).Error
	if err != nil {
		return err
	}

	var doc2 *models.Document
	err = db.Session(&gorm.Session{}).Find(&doc2, "name = ?", doc1.Name).Error
	if err != nil {
		return err
	}

	spew.Dump(doc1, doc2)

	if doc1 != doc2 {
		panic(fmt.Sprintf("doc1[%p] != doc2[%p]\n", doc1, doc2))
	}

	doc1.Number = rand.Int63n(100000)

	err = db.
		Session(&gorm.Session{
			FullSaveAssociations: false,
		}).
		Save(doc1).
		Error
	if err != nil {
		return err
	}

	doc2.DocType.Type = "contract" + cvt.String(rand.Int())

	err = db.
		Session(&gorm.Session{
			FullSaveAssociations: false,
		}).
		Save(doc2).
		Error
	if err != nil {
		return err
	}

	return nil
}

func createDB() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=Europe/Moscow"
	return gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			Colorful:                  true,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      false,
			LogLevel:                  logger.Info,
		}),
	})
}
