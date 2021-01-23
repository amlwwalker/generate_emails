package utilities

import (
	//"amlwwalker/envoye-backend/backend/pkg/authentication"
	"amlwwalker/gmail-backend/backend/pkg/database"
	g "amlwwalker/gmail-backend/backend/pkg/gmail"
	"bytes"
	b64 "encoding/base64"
	"github.com/jasonlvhit/gocron"
	gmail "google.golang.org/api/gmail/v1"
	"github.com/rs/xid"
	"github.com/jordan-wright/email"
	"github.com/jinzhu/gorm"
	"html/template"
	"sync"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func Cron(wg *sync.WaitGroup) {
	gocron.Every(2).Minutes().Do(func() {sendBufferedEmailsForAccounts(wg)})
	<-gocron.Start()
}
func sendBufferedEmailsForAccounts(wg *sync.WaitGroup) {
	t0 := time.Now()
	log.Println("buffering emails")
	maxSendLimit := 20 //limit per request to db
	dailyMaxQuota := 400 //limit for the day
	//go into the database
	//get a limit of 50 emails per account
	//prep and stagger sending for each
	//todo: this should be groups of 50, per account
	//need to know the user's ID for creating a client later
	//ok for each email
	//need to process this per user
	//so that the gmail client is correct for each user
	senderEmails := make(map[string][]database.CampaignEmail)
	if orderedEmails, err := database.RetrieveBufferedEmailsForUsers(uint(maxSendLimit)); err != nil {
		log.Println("error getting limited emails ", err)
	} else {
		//group into senders
		for _, v := range orderedEmails {
			log.Printf("email %d, %t, %s\r\n", v.ID, *v.Batched, v.BatchedAt)
			if _, ok := senderEmails[v.SenderEmail]; !ok {
				senderEmails[v.SenderEmail] = []database.CampaignEmail{}
			}
			senderEmails[v.SenderEmail] = append(senderEmails[v.SenderEmail], v) //its now marked as batched and so shouldn't be picked up next time
		}
	}
	//var mu = &sync.Mutex{}

	for key, emails := range senderEmails {
		log.Println("processing emails ", len(emails))

		wg.Add(1)
		go func(s string, e []database.CampaignEmail) {
			log.Printf("creating new routine for %s\r\n", key)
			defer func() {
				log.Printf("finished processing for %s\r\n", key)
				wg.Done()
			}()

			var err error
			var user database.User
			if user, err = database.RetrieveUserProfile(key); err != nil {
				//hmm we couldn't find this sender
				log.Println("hmm error finding ", key, err)
				return
			}

			//wg.Add(1)
			//send each accounts emails on a seperate thread
			//go func(senderID uint, emailsToSend []database.CampaignEmail) {
			var gmailService g.Gmail
			var report *Report
			if gmailService, report = createGmailServiceForSending(user.ID); report != nil {
				return
			}
			//we need to get the number batched in last 3 minutes
			if senderDailyCount, err := database.CountTodaysEmailsForUser(s); err != nil {
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
				log.Println("user sent ", senderDailyCount, " today, so there are ", peelLimit, "left. This batch has ", len(e), "to send")
			}
			//we need to find out how the sending quota for the user will change
			//get a gmailObject so we can send the email
			log.Printf("from %d emails, we are sending %d\r\n", len(emails), len(e))
			for _, email := range e {
				if _, err := sendEmail(email, gmailService, true); err != nil {
					mailError := database.MailError{
						Model:      gorm.Model{},
						OwnerId:    email.OwnerId,
						EmailId:    email.ID,
						CampaignId: email.CampaignID,
						Error:      err.Error(),
					}
					database.CreateEmailError(mailError)
					//leave - this user has caused an error, we will pick up new emails shortly
					log.Println("we are exiting user ", s, " due to ", err, " for ", email.ID, " owned by ", email.OwnerId)
					return
				}
				time.Sleep(1500 * time.Millisecond) //put a bit of a delay between emails sending
			}
		}(key, emails)
	}
	wg.Wait()
	log.Printf("time to process this batch: %s", time.Since(t0))
	return
}
func createGmailServiceForSending(senderID uint) (g.Gmail, *Report){
	var client *http.Client
	var gmailService g.Gmail
	var err error
	if client, _, err = g.CreateClient(senderID); err != nil {
		if err != nil {
			r := Report{
				Err: "Could not retrieve client for user: " + err.Error(),
			}
			return gmailService, &r
		}
	} else {
		{
			var err error
			gmailService, err = g.CreateGmail(client)
			if err != nil {
				r := Report{
					Err: "Could not create gmail service for user: " + err.Error(),
				}
				return gmailService, &r
			}
		}
	}
	return gmailService, nil
}

func injector(id string, htmlBody string, inject bool) string {
	if !inject {
		return htmlBody
	}
	if id == "" {
		return htmlBody
	}
	injectedContent := `
<table border="0" cellpadding="0" cellspacing="0" width="100%" style="background-color:#ffffff; border-collapse:collapse; padding:0; margin:0px;">
	<tr valign="top">
		<td align="center">
			<table id="u_content_text_2" class="u_content_text" style="font-family:arial,helvetica,sans-serif;" role="presentation" cellpadding="0" cellspacing="0" width="100%" border="0">
				<tbody>
					<tr>
						<td style="overflow-wrap:break-word;word-break:break-word;padding:10px;font-family:arial,helvetica,sans-serif;" align="left">
							<div class="v-text-align" style="color: #000000; line-height: 140%; text-align: left; word-wrap: break-word;">
								<p style="font-size: 12px; line-height: 140%; text-align: center;">
									<a href="https://www.envoye.app" target="_blank" rel="noopener" style="color:#656565; text-decoration:none;">Powered by Envoye - the simplest way to stay in touch with your customers</a>
								</p>
								<p style="font-size: 12px; line-height: 140%; text-align: center;">
								</p>
								<p style="font-size: 12px; line-height: 140%; text-align: center;">
									<a href="`+os.Getenv("TRACKING_URL")+"/api/utilities/unsubscribe/"+id+`" target="_blank" rel="noopener" style="color:#656565">unsubscribe to these emails</a>
								</p>
								<p style="font-size: 12px; line-height: 140%; text-align: center;">
									<img src="`+os.Getenv("TRACKING_URL")+`/api/utilities/track/`+id+`/pixel.png" />
								</p>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
		</td>
	</tr>
</table>
`

	//now search for the closing body tag
	i := strings.Index(htmlBody, "</body>")
	htmlBody = htmlBody[:i] + "\r\n\r\n\r\n" + injectedContent +"\r\n" + htmlBody[i:]
	return htmlBody
}
func injectTemplateMergeFields(firstName, lastName, htmlBody string) (string, error) {
	var tpl bytes.Buffer
	tmpl, err := template.New("email").Parse(htmlBody)
	if err != nil {
		log.Println("unable to parse template", err)
		return "", err
	}
	data := map[string]interface{}{"firstName": firstName, "lastName": lastName}
	if err = tmpl.Execute(&tpl, data); err != nil {
		log.Println("unable to execute template.")
		return "", err
	}
	return tpl.String(), nil

}
func injectDetailsIntoSubject(subject, recipientFirstName, recipientLastName string) (string, error) {
	var tpl bytes.Buffer
	data := map[string]interface{}{"firstName": recipientFirstName, "lastName": recipientLastName}
	//"Number one show \"{{ .Show}}\" has: \"{{ .Lead}}\""
	tmpl, err := template.New("subject").Parse(subject)
	if err != nil {
		return tpl.String(), err
	}
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		return tpl.String(), err
	}
	return tpl.String(), nil
}
func sendEmail(e database.CampaignEmail, gmailService g.Gmail, inject bool) (database.CampaignEmail, error) {

	//create a gmail message

	constructor := email.NewEmail()
	constructor.From = e.SenderName + "<" + e.SenderEmail + ">"
	constructor.To = []string{e.RecipientEmail}
	if subject, err := injectDetailsIntoSubject(e.Subject, e.RecipientFirstName, e.RecipientLastName); err != nil {
		constructor.Subject = e.Subject
	} else {
		constructor.Subject = subject
	}
	constructor.Text = []byte("The text body of this email is yet to be constructed properly")
	//before putting the html in the email, we need to add the unsubscribe and pixel tracker
	//before injecting our footers, lets add the customers's name in wherever its relevant
	mergedBody, err := injectTemplateMergeFields(e.RecipientFirstName, e.RecipientLastName, e.EmailBody)
	if err != nil {
		return e, errorPreparingEmail(e, err)
	}
	injectedEmailBody := injector(e.EmailIdentifier, mergedBody, inject)
	constructor.HTML = []byte(injectedEmailBody)

	raw, err := constructor.Bytes()
	if err != nil {
		return e, errorPreparingEmail(e, err)
	}
	var message gmail.Message
	//form the email
	message.Raw = b64.URLEncoding.EncodeToString(raw)
	//this may need to be the account holder and may be "me" perhasps?
	id, err := gmailService.SendMail(e.SenderEmail, message)
	e.GmailID = id
	//var err error
	if err != nil {
		return e, errorPreparingEmail(e, err)
	} else {
		t := true
		e.Sent = &t
		e.SentAt = time.Now()
	}
	if err := database.UpdateEmail(e); err != nil {
		return e, errorPreparingEmail(e, err)
	}
	log.Printf("SENT: ", e.ID, e.CampaignID, e.RecipientEmail, e.Sent, e.Batched, e.Aborted, e.ErrorOnSending, e.Sent, e.SentAt)
	return e, nil
}

func checkMarkEmailAsbatched(e database.CampaignEmail, batchToggle bool) (bool, error) {
	e.Batched = &batchToggle
	e.BatchedAt = time.Now()
	if rowsAffected, err := database.MarkBatchingForEmail(e); err != nil {
		return false, err
	} else if rowsAffected > 0 {
		//we did mark it
		return true, nil
	}
	return false, nil
}

func errorPreparingEmail(e database.CampaignEmail, err error) error {
	t := true //need a placeholder
	e.SendingError = err.Error()
	e.ErrorOnSending = &t
	if err := database.UpdateEmail(e); err != nil {
		return err
	}
	return nil
}
func SendTemplateTestEmail(u database.User, template database.Template) error {
	if gmailService, report := createGmailServiceForSending(u.ID); report != nil {
		return errors.New("Cant send user onboarding mail.")
	} else {
		tmp := database.CampaignEmail{}
		tmp.Subject = "Preview of Template - " + template.Name
		tmp.RecipientFirstName = u.Name
		tmp.RecipientLastName = ""
		tmp.RecipientEmail = u.Email
		tmp.SenderName = "Envoye.app" //the name of the sender?
		tmp.SenderEmail = u.Email
		tmp.EmailIdentifier = xid.New().String()
		//template/generate this content now
		tmp.EmailBody = template.HtmlContent
		if _, err := sendEmail(tmp, gmailService, true); err != nil {
			return err
		}
	}
	return nil
}
func SendOnboardingEmail(newUser database.User, template database.Template) error {
	if gmailService, report := createGmailServiceForSending(1); report != nil {
		return errors.New("Cant send user onboarding mail.")
	} else {
		tmp := database.CampaignEmail{}
		tmp.Subject = "Welcome to Envoye!"
		tmp.RecipientFirstName = newUser.Name
		tmp.RecipientLastName = ""
		tmp.RecipientEmail = newUser.Email
		tmp.SenderName = "Envoye Team" //the name of the sender?
		tmp.SenderEmail = "hello@envoye.app"
		tmp.EmailIdentifier = xid.New().String()
		//template/generate this content now
		tmp.EmailBody = template.HtmlContent
		if _, err := sendEmail(tmp, gmailService, false); err != nil {
			return err
		}
	}
	return nil
}