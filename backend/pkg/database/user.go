package database

import (
	"errors"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"golang.org/x/oauth2"
	"log"
)

type User struct {
	gorm.Model
	Token AccessToken
	TokenID uint `gorm:"ForeignKey:ID"`
	SheetId string `json:"sheet_id"`
	Name string `json:"users_name"`
	Email      string `sql:"size:255;unique;index" "json":"email"`
	EmailQuota uint  `"json":"email_quota"`
	ImageUrl   string
	SuperAdmin bool      `"json":"superadmin"`
	Validity   bool      `"json":"validity"`
	Tour *bool `json:"tour"`
}
type AccessToken struct {
	gorm.Model
	AccessToken string
	RefreshToken string
	Expiry time.Time
	TokenType string
}

func RetrieveUserProfile(emailAddress string) (User, error) {
	var u User
	log.Println("looking for: " + emailAddress)
	count := 0
	db.Where("email = ?", emailAddress).First(&u).Count(&count)

	log.Println("result: ", u)
	if count == 0 {
		return User{}, errors.New("we didn't find a user")
	}
	return u, nil
}

func RetrieveUserProfileById(usrId uint) (User, error) {
	var u User
	if err := db.Where("id = ?", usrId).First(&u).Error; err != nil {
		return (User{}), err
	} else {
		return u, nil
	}
}

//for invalidating/enabling by email
func ToggleValidUser(userId uint, validity bool) error {
	var u User
	if err := db.Model(&u).Where("id = ?", userId).Update("validity", validity).Error; err != nil {
		log.Println("Unable to mark set validity for: ", userId, " to true: ", validity)
		return err
	}
	return nil
}
func FindOrCreate(u User) (User, bool, error) {
	if db.Where("email = ? ", u.Email).First(&u).RecordNotFound() {
		err := db.Create(&u).Error
		log.Println("created - so returning ", u)
		return u, true, err
	}
	log.Println("finding - so returning ", u)
	return u, false, nil
}

func UpdateCustomerSheet(u User, sheetId string) (error) {
	if err := db.Model(&u).Where("id = ?", u.ID).Update("sheet_id", sheetId).Error; err != nil {
		log.Println("Unable to set sheet for: ", u.ID, " to true: ", sheetId)
		return err
	}
	return nil
}
func RetrieveCustomerSheet(id uint) (string, error) {
	var tmp User
	if err := db.Model(&tmp).Where("id = ?", id).First(&tmp).Error; err != nil {
		log.Println("Unable to find sheet for: ", id)
		return "", err
	}
	return tmp.SheetId, nil
}
func FindUserAccessToken(id uint) (*oauth2.Token, error) {
	var u User
	if err := db.Where("id=?", id).Preload("Token").First(&u).Error; err != nil {
		log.Println("finding access token error ", err)
		return (&oauth2.Token{}), err
	} else {
		log.Println("no issues finding token and user ", u)
		tmpToken := new(oauth2.Token)
		tmpToken.AccessToken = u.Token.AccessToken
		tmpToken.RefreshToken = u.Token.RefreshToken
		tmpToken.Expiry = u.Token.Expiry
		tmpToken.TokenType = u.Token.TokenType
		return tmpToken, nil
	}
}
