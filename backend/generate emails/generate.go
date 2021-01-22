package generate_emails

import (
"fmt"
	"github.com/jasonlvhit/gocron"
	"amlwwalker/gmail-backend/backend/pkg/database"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syreclabs.com/go/faker"
	"time"
)

type CampaignEmail struct {
	gorm.Model
	OwnerId          uint      `json:"ownerId"`
	RecipientName    string    `json:"recipientName"`
	RecipientEmail   string    `json:"recipientEmail"`
	SenderName       string    `json:"senderName"`
	SenderEmail      string    `json:"senderEmail"`
	Subject          string    `json:"subject"`
	CampaignID       uint      `json:"campaignID"` //the campaign it is a member of
	EmailBody        string    `json:"emailBody"`
	GmailEmailId     string    `json:"gmailEmailId"` //todo: wire it up to listen to changes so that we know which email's successfully sent/failed etc
	CustomerID       uint      `json:"customerID"`
	CustomerViewed   *bool     `json:"customerViewed"`
	CustomerViewedAt time.Time `json:"customerViewedAt"`
	SentAt           time.Time `json:"sentAt"`
	Sent             *bool     `json:"sent"` //null is not sent, false is can't send due to no subscription, true is sent
	AbortedAt        time.Time `json:"abortedAt"`
	Aborted          *bool     `json:"aborted"` //null is not sent, false is can't send due to no subscription, true is sent
	BatchedAt        time.Time `json:"batchedAt"`
	Batched          *bool     `json:"batched"` //null is not batched, true is batched
	SendingError     string    `json:"sendingError"`
	ErrorOnSending   *bool     `json"errorOnSending`
}
var sentEmails = []CampaignEmail{}
func CreateCampaignEmail(e CampaignEmail) error {
	err := db.Create(&e).Error
	if err != nil {
		log.Println("error saving new campaign email ", err)
	}
	return err
}

func LimitPerSenderEmail(maxLimitReturn uint) ([]CampaignEmail, error) {
	limitedOrderedEmails := []CampaignEmail{}
	//lets simplify and count the number batched in the last 3 minutes
	//we could get just the IDs at this stage
	//todo: we could get the content much later. At this point just get the ideas a
	if err := db.Raw("select id from ( select ROW_NUMBER() OVER (PARTITION BY sender_email ORDER BY sender_email) AS r, t.* from campaign_emails t where (sent=false OR sent IS NULL) AND (aborted=false OR aborted IS NULL) AND (batched=false OR batched IS NULL) AND (error_on_sending=false OR error_on_sending IS NULL) AND cast(sending_time as time) BETWEEN CURRENT_TIME - (15 * interval '1 MINUTE') AND CURRENT_TIME) x where x.r <= ?;", maxLimitReturn).Scan(&limitedOrderedEmails).Error; err != nil {
		log.Println("struggled to get the data ", err)
		return limitedOrderedEmails, err
	}
	//fmt.Println("emailIds", limitedOrderedEmails)
	return limitedOrderedEmails, nil
}
func returnBatchedUpdateEmail(maxLimitReturn uint) ([]CampaignEmail, error) {
	limitedOrderedEmails := []CampaignEmail{}

	//users := []database.User{}
	//get each user
	//db.Debug().Model(database.User{}).Find(&users)
	//for each user...
	//get all emails to be sent in last 15 minutes, and were batched more than 3 minute ago (perhaps more minutes ago, Mark them as batched now
	//for _, v := range users {
	//	log.Println("processing:", v.Email)
	//todo: new condition to check sending time required.
	if err := db.Debug().Raw(`update campaign_emails set batched = true,batched_at = NOW() where id IN (select id from ( select ROW_NUMBER() OVER (PARTITION BY sender_email ORDER BY sender_email) AS r, t.* from campaign_emails t where (sent=false OR sent IS NULL) AND (aborted=false OR aborted IS NULL) AND (batched=false OR batched IS NULL OR (batched=true AND cast(batched_at as time) <= CURRENT_TIME - (3 * interval '1 MINUTE'))) AND (error_on_sending=false OR error_on_sending IS NULL) AND cast(sending_time as time) BETWEEN CURRENT_TIME - (15 * interval '1 MINUTE') AND CURRENT_TIME) x where x.r <= ?) RETURNING *`, maxLimitReturn).Scan(&limitedOrderedEmails).Error; err != nil {
		log.Println("error updating batching ", err)
		return nil, err
	}
	if len(limitedOrderedEmails) < 1{
		return limitedOrderedEmails, nil
	}
	email1 := limitedOrderedEmails[0]
	log.Println("about to process", len(limitedOrderedEmails), "emails for", email1.RecipientEmail, email1.Batched, email1.BatchedAt)
	//}
	//do the whole process for a user in its own thread

	//`update campaign_emails set batched = true,
	//															batched_at = NOW()
	//									where (owner_id = ?)
	//									AND (sent=false OR sent IS NULL)
	//									AND (aborted=false OR aborted IS NULL)
	//									AND (error_on_sending=false OR error_on_sending IS NULL)
	//									AND (batched=false OR batched IS NULL OR (batched=true AND cast(batched_at as time) <= CURRENT_TIME - (3 * interval '1 MINUTE')))
	//									RETURNING *`

	//update batching
	//where select all rows where:
	//sent = false or null
	//error = false or null
	//aborted = false or null
	//batched = false or null or (true and more than 1 minute ago)
	//sending time is within the last 15 minutes
	//return all those rows

	//what this is missing is a restriction on how many to return - currently its hard coded, but it should be based on how many were
	//batched in the last minute (i.e if there were 10 sent today and 5 batched in last 1 miunte then you can send 15 less than you thought

	//if err := db.Debug().Raw("update campaign_emails set batched = true, batched_at = NOW() where (select id from ( select ROW_NUMBER() OVER (PARTITION BY sender_email ORDER BY sender_email) AS r, t.id from campaign_emails t where (sent=false OR sent IS NULL) AND (aborted=false OR aborted IS NULL) AND (batched=false OR batched IS NULL OR (batched=true AND cast(batched_at as time) <= CURRENT_TIME - (1 * interval '1 MINUTE'))) AND (error_on_sending=false OR error_on_sending IS NULL) AND cast(sending_time as time) BETWEEN CURRENT_TIME - (15 * interval '1 MINUTE') AND CURRENT_TIME) x where x.r <= ?) RETURNING *", maxLimitReturn).Scan(&limitedOrderedEmails).Error; err != nil {
	//	log.Println("struggled to get the data ", err)
	//	return limitedOrderedEmails, err
	//}
	return limitedOrderedEmails, nil
}

var db *gorm.DB

func InitDB() {
	os.Setenv("PSQL_USERNAME", "alex")
	os.Setenv("PSQL_PASSWORD", "alex")
	os.Setenv("PSQL_DATABASE", "envoye")
	os.Setenv("PSQL_HOST", "localhost")

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
		log.Println("connection failed")
		panic("failed to connect database: " + err.Error())
	}
	log.Println("success with psql")
	//defer db.Close()
	db.AutoMigrate(&CampaignEmail{})
}

type sender struct {
	SenderName  string
	SenderEmail string
	Quota       int64
}

//this is a representation of the database object
var senders = []sender{{SenderName: "Alexz Walker", SenderEmail: "a.mlw.walker@gmail.com", Quota: 0}}
func (s *sender) retrieveQuota() bool{
	//number of emails sent
	//this should update the quota assuming that it represents one send event
	if s.Quota > 250 {
		return false
	}
	return true
}
func InitCron(wg *sync.WaitGroup) {
	gocron.Every(2).Second().Do(func() {sendBufferedEmailsForAccounts(wg)})
	<-gocron.Start()
}
func randate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}
func sendBufferedEmailsForAccounts(wg *sync.WaitGroup) {
	log.Println("called again at ", time.Now())
	maxSendLimit := 5   //limit per request to db
	dailyMaxQuota := 33 //limit for the day regardless of quota

	/*

		   ok so a user has a number of credits in a period of time
		   that limit is 250 credits per second
		   sending an email is 100 credits
		   that means as much as 2 emails a second per user is the limit


			//thought - each time the cron comes round, it will create a new thread for the same company
			//that can result in multiple threads for the same user, pushing the quota up
			//therefore the quota has to be shared and exponentially pushed off

	*/

	senderEmails := make(map[string][]CampaignEmail)
	if orderedEmails, err := returnBatchedUpdateEmail(uint(maxSendLimit)); err != nil {
		log.Println("error getting limited emails ", err)
	} else {
		//group into senders
		for _, v := range orderedEmails {
			fmt.Printf("email %d, %t, %s\r\n", v.ID, *v.Batched, v.BatchedAt)
			if _, ok := senderEmails[v.SenderEmail]; !ok {
				senderEmails[v.SenderEmail] = []CampaignEmail{}
			}
			senderEmails[v.SenderEmail] = append(senderEmails[v.SenderEmail], v) //its now marked as batched and so shouldn't be picked up next time
		}
	}
	//var mu = &sync.Mutex{}

	for key, emails := range senderEmails {
		log.Println("processing emails ", len(emails))

		wg.Add(1)
		go func(s string, e []CampaignEmail) {
			log.Printf("creating new routine for %s\r\n", key)
			defer func() {
				log.Printf("finished processing for %s\r\n", key)
				wg.Done()
			}()
			//we need to get the number batched in last 3 minutes
			if senderDailyCount, err := CountTodaysEmailsForUser(s); err != nil {
				log.Println("count not get a count for this user", err)

				//todo mark the quota very high and come back to this
				return
			} else {
				if dailyMaxQuota-senderDailyCount <= 0 {
					log.Println("no more quota for today for ", s)
					//todo mark sending time (quota time to tomorrow?)
					return
				}
				//limit to peeling off a max of 3 each time
				peelLimit := maxSendLimit
				if peelLimit > dailyMaxQuota-senderDailyCount {
					//then we limit to the maxquote
					peelLimit = dailyMaxQuota - senderDailyCount
				}
				if len(e) > peelLimit {
					e = e[:peelLimit] //now we have limited the resonse to make sure we cant send more than the daily limit
				}
				if len(e) == 0 {
					//todo mark sending time (quota time to tomorrow?)
					return
				}
			}
			//we need to find out how the sending quota for the user will change
			//get a gmailObject so we can send the email
			fmt.Printf("from %d emails, we are sending %d\r\n", len(emails), len(e))
			for _, email := range e {
				if _, err := sendEmail(email); err != nil {
					mailError := database.MailError{
						Model:      gorm.Model{},
						OwnerId:    email.OwnerId,
						EmailId:    email.ID,
						CampaignId: email.CampaignID,
						Error:      err.Error(),
					}
					database.CreateEmailError(mailError)
					//leave - this user has caused an error, we will pick up new emails shortly
					log.Println("we are exiting user ", s)
					return
				}
				//time.Sleep(1500 * time.Millisecond) //put a bit of a delay between emails sending
			}
		}(key, emails)
	}
	wg.Wait()
}
func initNewEmails() {
	for _, v := range senders {
		for i := 0; i < 2000; i++ {
			c := CampaignEmail{
				Model:            gorm.Model{},
				OwnerId:          1,
				RecipientName:    fmt.Sprintf("%d-amlwwalker", i),
				RecipientEmail:   fmt.Sprintf("amlwwalker+%d@gmail.com", i),
				SenderName:       v.SenderName,
				SenderEmail:      v.SenderEmail,
				Subject:          fmt.Sprintf("subject = %d", i),
				CampaignID:       0,
				EmailBody:        faker.Lorem().Paragraph(5),
				GmailEmailId:     faker.Hacker().Noun(),
				CustomerID:       0,
				CustomerViewed:   nil,
				CustomerViewedAt: time.Time{},
				SentAt:           time.Time{},
				Sent:             nil,
				AbortedAt:        time.Time{},
				Aborted:          nil,
				SendingError:     "",

				ErrorOnSending:   nil,
			}
			err := CreateCampaignEmail(c)
			if err != nil {
				log.Println("error storing campaign email ", err)
			}
		}
	}
}
func main() {
	initNewEmails()
	//c := make(chan os.Signal, 1)
	//signal.Notify(c, os.Interrupt)
	//go func(){
	//	for sig := range c {
	//		log.Printf("captured %v, stopping profiler and exiting..", sig)
	//		//ok now check for duplicates
	//		idList := []uint{}
	//		for _, v := range sentEmails {
	//			//print the
	//			for _, id := range idList {
	//				if v.ID == id {
	//					log.Println("WARNING DUPLICATE FOUND!", v.ID)
	//				} else {
	//					idList = append(idList, v.ID)
	//				}
	//			}
	//
	//		}
	//	}
	//}()
	//InitDB()
	//var wg sync.WaitGroup
	//initNewEmails()
	//time.Sleep(30 * time.Second)
	//go InitCron(&wg)
	//
	//
	//wg.Add(1)
	//wg.Wait()
}

func CountTodaysEmailsForUser(s string) (int, error) {
	return 3, nil
}
func sendEmail(email CampaignEmail) (CampaignEmail, error) {
	log.Println("sending mail %d to %s", email.ID, email.RecipientEmail)
	if err := db.Debug().Where("id = ?", email.ID).Update("sent = true AND sent_at = NOW").Error; err != nil {
		return email, err
	}
	sentEmails = append(sentEmails, email)
	return email, nil
}

