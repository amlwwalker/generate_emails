package database

import (
	"log"
	"time"
	"github.com/jinzhu/gorm"
)

type Campaign struct {}

type CampaignEmail struct {
	gorm.Model
	OwnerId            uint     `json:"ownerId"`
	RecipientFirstName string   `json:"recipientFirstName"`
	RecipientLastName string   `json:"recipientLastame"`
	RecipientEmail     string   `json:"recipientEmail"`
	SenderName         string   `json:"senderName"`
	SenderEmail        string   `json:"senderEmail"`
	Subject            string   `json:"subject"`
	Campaign           Campaign `json:"campaign"`
	CampaignID         uint     `json:"campaignID"` //the campaign it is a member of
	EmailBody          string   `json:"emailBody"`
	EmailIdentifier    string   `json:"emailIdentifier"`
	GmailID            string   `json:"gmailEmailId"` //todo: wire it up to listen to changes so that we know which email's successfully sent/failed etc
	CustomerID         uint     `json:"customerID"`
	CustomerViewed   *bool     `json:"customerViewed"`
	CustomerViewedAt time.Time `json:"customerViewedAt"`
	SentAt           time.Time `json:"sentAt"`
	Sent             *bool     `json:"sent"` //null is not sent, false is can't send due to no subscription, true is sent
	AbortedAt        time.Time `json:"abortedAt"`
	Aborted          *bool     `json:"aborted"` //null is not sent, false is can't send due to no subscription, true is sent
	BatchedAt        time.Time `json:"batchedAt"`
	Batched	        *bool     `json:"batched"` //null is not batched, true is batched
	SendingError     string    `json:"sendingError"`
	ErrorOnSending   *bool     `json"errorOnSending`
	SendingTime time.Time `json:"sendingTime" sql:"DEFAULT:current_timestamp"` //emails will go out shortly after this time each day
}
func CreateCampaignEmail(e CampaignEmail) error {
	err := db.Create(&e).Error
	if err != nil {
		log.Println("error saving new campaign email ", err)
	}
	return err
}
func RetrieveBufferedEmailsForUsers(maxLimitReturn uint) ([]CampaignEmail, error) {
	limitedOrderedEmails := []CampaignEmail{}
	if err := db.Debug().Raw(`update campaign_emails set batched = true,batched_at = NOW() where id IN 
									(select id from ( select ROW_NUMBER() OVER (PARTITION BY sender_email ORDER BY sender_email) AS r, t.* from campaign_emails t where 
											(sent=false OR sent IS NULL) AND 
											(aborted=false OR aborted IS NULL) AND 
											(batched=false OR batched IS NULL OR (batched=true AND cast(batched_at as time) <= CURRENT_TIME - (5 * interval '1 MINUTE'))) AND 
											(error_on_sending=false OR error_on_sending IS NULL) AND 
											cast(sending_time as time) BETWEEN CURRENT_TIME - (30 * interval '1 MINUTE') AND CURRENT_TIME) 
											x where x.r <= ?) RETURNING *`, maxLimitReturn).Scan(&limitedOrderedEmails).Error; err != nil {
		log.Println("error updating batching ", err)
		return nil, err
	}
	if len(limitedOrderedEmails) < 1{
		return limitedOrderedEmails, nil
	}
	email1 := limitedOrderedEmails[0]
	log.Println("about to process", len(limitedOrderedEmails), "emails for", email1.RecipientEmail, email1.Batched, email1.BatchedAt)

	return limitedOrderedEmails, nil
}

func CountTodaysEmailsForUser(sender string) (int, error) {
	emailCount := 0
	err := db.Table("campaign_emails").Where("sender_email = ? AND sent = true AND sent_at = CURRENT_DATE", sender).Count(&emailCount).Error
	return emailCount, err
}
func UpdateEmail(e CampaignEmail) error {
	err := db.Model(&e).Where("id = ?", e.ID).Update(&e).Error
	return err
}
func AbortCampaignEmails(u User, campaignId uint) error {
	t := true
	result := db.Debug().Model(&CampaignEmail{}).Where("owner_id = ? AND campaign_id = ? ", u.ID, campaignId).Update(&CampaignEmail{Aborted: &t, AbortedAt: time.Now()})
	return result.Error
}
func MarkBatchingForEmail(e CampaignEmail) (int64, error) {
	//dont mark it if the batch state is the same as the current state
	result := db.Debug().Model(&CampaignEmail{}).Where("id = ? AND (batched IS NULL OR batched = ?)", e.ID, !*e.Batched).Update(&CampaignEmail{Batched: e.Batched, BatchedAt: e.BatchedAt})
	return result.RowsAffected, result.Error
}
