package database

import "github.com/jinzhu/gorm"

type Template struct {
	gorm.Model
	Name string
	HtmlContent string
}