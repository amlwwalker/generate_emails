package database
import (
	"github.com/jinzhu/gorm"
	"log"
	"time"
)
type MailError struct {
	gorm.Model
	OwnerId            uint     `json:"ownerId"`
	EmailId uint `json:"email_id"`
	CampaignId uint `json:"email_id"`
	Error string `json:"error"`
	time time.Time `jsom:"error_time"`
}
func CreateEmailError(e MailError) error {
	err := db.Create(&e).Error
	if err != nil {
		log.Println("error saving new error ", err)
	}
	return err
}
