package database

import (
	"github.com/jinzhu/gorm"
	"log"
	"os"
)

var db *gorm.DB

func InitDB() {
	username := os.Getenv("PSQL_USERNAME")
	password := os.Getenv("PSQL_PASSWORD")
	dbname := os.Getenv("PSQL_DATABASE")
	host := os.Getenv("PSQL_HOST")

	if username == "" {
		log.Println("ENV VARS missing, defaulting")
		os.Exit(1)
		//username = "alex"
	}

	log.Println("connecting with: ", "host="+host+" user="+username+" dbname="+dbname+" sslmode=disable password="+password)
	//configure the DB
	var err error
	db, err = gorm.Open("postgres", "host="+host+" user="+username+" dbname="+dbname+" sslmode=disable password="+password)
	if err != nil {
		log.Println("connection failed", err)
		panic("failed to connect database: " + err.Error())
	}
	log.Println("success with psql")
	// defer db.Close()

	//gob.Register(User{})
	//db.DropTableIfExists(&Customer{}, &AccessToken{}, &User{})
	db.AutoMigrate(&User{}, &AccessToken{}, &CampaignEmail{}, &MailError{}, &Campaign{}, &Template{})
	//db.CreateTable(&User{}, &AccessToken{}, &Customer{}, &Campaign{}, &Template{})
	db.Model(&User{}).AddForeignKey("token_id", "access_tokens(id)", "CASCADE", "CASCADE")
	//db.Model(&Campaign{}).AddForeignKey("template_id", "templates(id)", "RESTRICT", "RESTRICT")
	//db.Model(&Campaign{}).AddForeignKey("id", "customers(id)", "CASCADE", "CASCADE")
	//db.Model(&Campaign{}).AddForeignKey("id", "customers(id)", "CASCADE", "CASCADE") // Foreign key need to define manually
}
