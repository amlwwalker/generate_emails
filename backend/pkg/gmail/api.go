// gmail package is some wrapper functions
// to simplify the needs of the gmail api
// documentation: https://godoc.org/google.golang.org/api/gmail/v1#UsersMessagesService.Insert
// code: https://github.com/google/google-api-go-client/blob/master/gmail/v1/gmail-gen.go
package user

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"

	"google.golang.org/api/gmail/v1"
	"log"
	"net/http"
	"net/mail"
	"strings"
)

// WatchResponse is what comes back when a watch is
// initialised on an account
// It contains the history ID at the time the watch began
type WatchResponse struct {
	HistoryID  uint64 `json:"historyId,string"`
	Expiration string `json:"expiration"`
}

// Response is the structure that is received
// when Gmail pushes a change to an inbox to this
// application
type Incoming struct {
	Message      Message `json:"message"`
	Subscription string  `json:"subscription"`
}

// Message contains all of the information regarding the user's
// account change. The Data element is a b64 encoded version
// of the relevant information. MessageID is the uniqueID to which
// email this information is relevant to
type Message struct {
	Data      string `json:"data"`
	MessageID string `json:"message_id"`
}

// The structure of the information contained within Message.Data
// above after it has been unmarshaled
type PushData struct {
	Email     string `json:"emailAddress"`
	HistoryID uint64 `json:"historyId"`
}
type RedisData struct {
	MessageId string `json:"messageId"`
	User string `json:"user"`
}

type Gmail struct {
	Srv *gmail.Service
}
type UnfilteredMessage struct {
	Recipient            *mail.Address
	Sender               *mail.Address
	Headers              []*gmail.MessagePartHeader
	BodyContentHeaders              []*gmail.MessagePartHeader
	Subject              string
	Snippet              string
	//BodyContentType *gmail.MessagePartHeader
	//BodyTransferEncoding *gmail.MessagePartHeader
	Body                 string
	BodySize             int64
	MessageId            string
	ThreadId             string
	Links                []string
	Raw string
}
// CreateGmail creates a new gmail object for the
// current client
func CreateGmail(client *http.Client) (Gmail, error) {
	srv, err := gmail.New(client)
	if err != nil {
		log.Printf("gmail error: %s\n", err)
		return Gmail{}, err
	}
	g := Gmail{srv}
	return g, nil
}

// Watch executes the gmail api Watch routine
// for a particular account
func (this *Gmail) Watch(user string) (WatchResponse, error) {
	var watchRequest gmail.WatchRequest
	watchRequest.LabelIds = append(watchRequest.LabelIds, "INBOX")
	watchRequest.TopicName = "projects/oauthproxy-1283/topics/inbox"//os.Getenv("GMAIL_TOPIC")
	req := this.Srv.Users.Watch(user, &watchRequest)

	r, err := req.Do()
	if err != nil {
		log.Printf("req error for %s: %s\n", user, r)
		return (WatchResponse{}), err
	}
	var watchResponse WatchResponse
	b, err := r.MarshalJSON()
	if err := json.Unmarshal(b, &watchResponse); err != nil {
		log.Printf("json error: %s\n", err)
		return (WatchResponse{}), err
	}
	return watchResponse, nil
}

// StopWatching disconnects the push service from an end user's account
func (this *Gmail) StopWatching(user string) error {
	req := this.Srv.Users.Stop(user)
	if err := req.Do(); err != nil {
		log.Printf("unable to make request when trying to stop(): %s\n", err)
		return err
	}
	return nil
}

func (this *Gmail) RetrieveUserProfile(user string) (*gmail.Profile, error) {
	log.Println(this, this.Srv)
	req := this.Srv.Users.GetProfile(user)
	r, err := req.Do()
	if err != nil {
		log.Printf("retrieving user profile failed: %s - hID: %s\n", err, user)
		return r, err
	}
	return r, nil
}

//RetrieveAliases
//// this could be many, but tends to be few
//func (this *Gmail) RetrieveAliases(user string, historyId uint64) ([]userAlias, error) {
//	r, err  := this.Srv.Users.Settings.SendAs.List(user).Do()
//	if err != nil {
//		log.Printf("retrieving history objects failed: %s - hID: %s\n", err, historyId)
//		return ([]*gmail.History{}), err
//	}
//
//	var userAliases []userAlias
//	for _, alias := range r.SendAs {
//		//we need to know if its a default, the sendAsEmail and the replyToAddress
//		var a userAlias
//		a.isDefault = alias.IsDefault
//		a.sendingAs = alias.SendAsEmail
//		a.replyingTo = alias.ReplyToAddress
//		userAliases = append(userAliases, a)
//	}
//	return r.History, nil
//}
// RetrieveHistory returns all history IDs since the history ID currently
// being processed. If this takes a while, or alot of events occur on the account
// this could be many, but tends to be few
func (this *Gmail) RetrieveHistory(user string, historyId uint64) ([]*gmail.History, error) {
	req := this.Srv.Users.History.List(user).StartHistoryId(historyId)
	r, err := req.Do()
	if err != nil {
		log.Printf("retrieving history objects failed: %s - hID: %s\n", err, historyId)
		return ([]*gmail.History{}), err
	}
	return r.History, nil
}
// processHistories is a helper function to loop through all the history
// elements and return the messages within
func (this *Gmail) ProcessSingleHistory(user string, history *gmail.History) []*gmail.Message {
	// var errors []error
	var messages []*gmail.Message
	//for each history we want to get jsut messages added to the inbox (not deleted)
	for _, m := range history.MessagesAdded {
		//we also don't want drafts
		draft := false
		for _, l := range m.Message.LabelIds {
			if l == "DRAFT" {
				draft = true
				break
			}
		}
		if !draft { //if we received it, its for processing
			messages = append(messages, m.Message)
		}
	}
	//for _, h := range history {
	//	messages = append(messages, h.Messages...)
	//}
	return messages
}
// processHistories is a helper function to loop through all the history
// elements and return the messages within
func (this *Gmail) ProcessHistories(user string, history []*gmail.History) []*gmail.Message {
	// var errors []error
	var messages []*gmail.Message
	for _, h := range history {
		//for each history we want to get jsut messages added to the inbox (not deleted)
		for _, m := range h.MessagesAdded {
			//we also don't want drafts
			ignore := false
			for _, l := range m.Message.LabelIds {
				if l == "DRAFT" {
					ignore = true
					break
				}
			}
			if !ignore { //if we received it, its for processing
				messages = append(messages, m.Message)
			}
		}
	}
	//for _, h := range history {
	//	messages = append(messages, h.Messages...)
	//}
	return messages
}
// RetrieveMessages processes each message per history that was returned from
// RetrieveHistory. This should be called in a loop over each history.Messages
func (this *Gmail) RetrieveUnfilteredMessage(user string, m RedisData) (UnfilteredMessage, error) {
	//this parses each and every message
	//TODO: collect each parsed message and process from somewhere
	//else. That way can process the reputation elsewhere
	//and then call upon updating the message thread
	var uMessage UnfilteredMessage
	//for m := range messages {
	log.Printf("retrieving message for: %s\n", user, m.MessageId)
	if msg, err := this.Srv.Users.Messages.Get(user, m.MessageId).Format("full").Do(); err != nil {
		log.Printf("retrieving message (%s) failed: %s\n", m.MessageId, err)
		return UnfilteredMessage{}, err
	} else {
		//in the mean time, get the body of the message
		var completed bool
		//var u UnfilteredMessage
		uMessage.Headers = msg.Payload.Headers
		//this is the first time we see the headers on the email
		//incomingEmail := false
		//for _, v := range msg.Payload.Headers {
		//	if strings.Contains(v.Name,"Received") {
		//		//this is an incoming email
		//		incomingEmail = true
		//		break
		//	}
		//}
		//if !incomingEmail {
		//	return UnfilteredMessage{}, errors.New("processed before. UnsubscribedAt.")
		//}
		uMessage.ThreadId = msg.ThreadId
		uMessage.MessageId = msg.Id
		if strings.Contains(msg.Payload.MimeType, "text/html") { //there could be a body on this
			log.Println("found body as part of payload")
			if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
				//for _, header := range msg.Payload.Headers {
				//	if strings.Contains(header.Name, "HtmlContent-Transfer-Encoding") {
				//		uMessage.BodyTransferEncoding = header
				//	}
				//	if strings.Contains(header.Name, "HtmlContent-Type") {
				//		uMessage.BodyContentType = header
				//	}
				//	//this is silly, checking headers that are irrelevant if above are found
				//}
				uMessage.BodyContentHeaders = msg.Payload.Headers
				uMessage.Body = msg.Payload.Body.Data
				uMessage.BodySize = msg.Payload.Body.Size
				completed = true
			}
		} else {
			//recursive function that looks for the html part
			uMessage.Body, uMessage.BodySize, uMessage.BodyContentHeaders, completed = HuntForRedOctober(msg.Payload.Parts)
			//uMessage.Body, uMessage.BodySize, uMessage.BodyTransferEncoding, uMessage.BodyContentType, completed = HuntForRedOctober(msg.Payload.Parts)
		}

		if !completed || len(uMessage.Body) == 0 || uMessage.BodySize == 0 {
			log.Println("no body was found, skipping message with ID ", msg.Id)
			//todo: if we are going to do this, we need to put it back in the inbox!
			return UnfilteredMessage{}, errors.New("no body was found, skipping " + msg.Id)
		}
		if err := this.MessageModify(user, msg.Id, "INBOX", false); err != nil { //false = remove
			log.Println("could not change label on email ", err, "for", msg.Id)
			return UnfilteredMessage{}, errors.New("could not change label on email, skipping " + msg.Id)
		}
		//process headers that we want exposed
		var err error
		for _, v := range uMessage.Headers {
			if strings.EqualFold(v.Name, "subject") {
				uMessage.Subject = v.Value
			} else if strings.EqualFold(v.Name, "To") {
				uMessage.Recipient, err = mail.ParseAddress(v.Value)
				if err != nil {
					log.Println("error retrieving to address", err)
				}
			} else if strings.EqualFold(v.Name, "From") {
				uMessage.Sender, err = mail.ParseAddress(v.Value)
				if err != nil {
					log.Println("error retrieving from address", err)
				}
			}
		}
		uMessage.Snippet = msg.Snippet
		//uMessage = append(uMessage, u)
		//if u, err := parseMessage(msg, m.UUID); err != nil {
		//	log.Printf("processing message (%s) failed: %s\n", m.UUID, err)
		//} else {
		//	log.Printf("processing message (%s) succeeded\n", m.UUID)
		//	u.MessageId = m.UUID
		//	u.ThreadId = m.ThreadId
		//	uMessage = append(uMessage, u)
		//}
	}
	//}
	return uMessage, nil
}

// RetrieveMessages processes each message per history that was returned from
// RetrieveHistory. This should be called in a loop over each history.Messages
func (this *Gmail) RetrieveUnfilteredMessages(user string, messages map[*gmail.Message]struct{}) ([]UnfilteredMessage, error) {
	//this parses each and every message
	//TODO: collect each parsed message and process from somewhere
	//else. That way can process the reputation elsewhere
	//and then call upon updating the message thread
	var uMessages []UnfilteredMessage
	for m := range messages {
		log.Printf("retrieving message for: %s\n", user, m.Id)
		if msg, err := this.Srv.Users.Messages.Get(user, m.Id).Format("full").Do(); err != nil {
			log.Printf("retrieving message (%s) failed: %s\n", m.Id, err)
			return ([]UnfilteredMessage{}), err
		} else {
			//in the mean time, get the body of the message
			var completed bool
			var u UnfilteredMessage
			u.Headers = msg.Payload.Headers
			//this is the first time we see the headers on the email
			incomingEmail := false
			for _, v := range msg.Payload.Headers {
				if strings.Contains(v.Name,"Received") {
					//this is an incoming email
					incomingEmail = true
					break
				}
			}
			if !incomingEmail {
				continue
			}
			u.ThreadId = m.ThreadId
			u.MessageId = m.Id
			if strings.Contains(msg.Payload.MimeType, "text/html") { //there could be a body on this
				log.Println("found body as part of payload")
				if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
					//for _, header := range msg.Payload.Headers {
					//	if strings.Contains(header.Name, "HtmlContent-Transfer-Encoding") {
					//		u.BodyTransferEncoding = header
					//	}
					//	if strings.Contains(header.Name, "HtmlContent-Type") {
					//		u.BodyContentType = header
					//	}
					//	//this is silly, checking headers that are irrelevant if above are found
					//}
					u.BodyContentHeaders = msg.Payload.Headers
					u.Body = msg.Payload.Body.Data
					u.BodySize = msg.Payload.Body.Size
					completed = true
				}
			} else {
				//recursive function that looks for the html part
				u.Body, u.BodySize, u.BodyContentHeaders, completed = HuntForRedOctober(msg.Payload.Parts)
				//u.Body, u.BodySize, u.BodyTransferEncoding, u.BodyContentType, completed = HuntForRedOctober(msg.Payload.Parts)
			}

			if !completed || len(u.Body) == 0 || u.BodySize == 0 {
				log.Println("no body was found, skipping message with ID ", m.Id)
				//todo: if we are going to do this, we need to put it back in the inbox!
				continue
			}
			if err := this.MessageModify(user, m.Id, "INBOX", false); err != nil { //false = remove
				log.Println("could not change label on email ", err, "for", m.Id)
				continue
			}
			//process headers that we want exposed
			var err error
			for _, v := range u.Headers {
				if strings.EqualFold(v.Name, "subject") {
					u.Subject = v.Value
				} else if strings.EqualFold(v.Name, "To") {
					u.Recipient, err = mail.ParseAddress(v.Value)
					if err != nil {
						log.Println("error retrieving to address", err)
					}
				} else if strings.EqualFold(v.Name, "From") {
					u.Sender, err = mail.ParseAddress(v.Value)
					if err != nil {
						log.Println("error retrieving from address", err)
					}
				}
			}
			u.Snippet = msg.Snippet
			uMessages = append(uMessages, u)
			//if u, err := parseMessage(msg, m.UUID); err != nil {
			//	log.Printf("processing message (%s) failed: %s\n", m.UUID, err)
			//} else {
			//	log.Printf("processing message (%s) succeeded\n", m.UUID)
			//	u.MessageId = m.UUID
			//	u.ThreadId = m.ThreadId
			//	uMessages = append(uMessages, u)
			//}
		}
	}
	return uMessages, nil
}

/*
if parts has a body, and the body header is text/html, then we can break out
we go down maximum 5 levels before leaving anyway
if not, if parts have parts, we call this function, passing the current count
if it doesn't move to the next part at this level
if we find it, return everything

*/
func HuntForRedOctober(parts []*gmail.MessagePart) (string, int64, []*gmail.MessagePartHeader, bool){

	for _, part := range parts {
		log.Printf("part %+v\r\n", part.MimeType)
		if strings.Contains(part.MimeType, "text/html") {
			if part.Body != nil && part.Body.Data != ""{
				//we've found a body that is html, for now we'll call that quits
				//var transferEncodingHeader *gmail.MessagePartHeader
				//var bodyContentType *gmail.MessagePartHeader
				//for _, header := range part.Headers {
				//	if strings.Contains(header.Name, "HtmlContent-Transfer-Encoding") {
				//		//we need the header later
				//		transferEncodingHeader = header
				//	}
				//	if strings.Contains(header.Name, "HtmlContent-Type") {
				//		bodyContentType = header
				//	}
				//}
				return part.Body.Data, part.Body.Size, part.Headers, true
			}
		}
		//this part isn't the one, so another check to do then
		if len(part.Parts) > 0 {
			//it has parts, so we call that too
			if data, size, bodyContentHeaders, completed := HuntForRedOctober(part.Parts); completed {
				//we found it, so we return everything
				return data, size, bodyContentHeaders, completed
			}
		}
	}
	var d string
	var s int64
	return d, s, []*gmail.MessagePartHeader{}, false
}

func (this *Gmail) TestSendMail(user, body string) error {
	// Message for our gmail service to send
	var message gmail.Message
	// Compose the message
	messageStr := []byte(
		"From: " + user + "\r\n" +
			"To: " + user + "\r\n" +
			"Subject: My first Gmail API message\r\n\r\n" +
			body)
	// Place messageStr into message.Raw in base64 encoded format
	message.Raw = b64.URLEncoding.EncodeToString(messageStr)
	_, err := this.SendMail(user, message) //don't need to know the id of the inserted message
	return err
}

// SendMail sends an email on behalf of an account holder
func (this *Gmail) SendMail(user string, message gmail.Message) (string, error) {
	// Send the message
	if res, err := this.Srv.Users.Messages.Send(user, &message).Do(); err != nil {
		log.Printf("Error sending mail: %v", err)
		return "", err
	} else {
		log.Printf("Message sent! threadID: %s, msgId: %s\n", res.ThreadId, res.Id)
		return res.Id, nil
	}
}

// Inserts mail into users mailbox
// Does not send a mail
// Requires the message that is going to be inserted
func (this *Gmail) TestInsertMail(user, body, thread string) error {
	// Message for our gmail service to send
	var message gmail.Message
	// Compose the message
	// body = html.EscapeString(body)
	messageStr := []byte(
		// "From: " + "Jeffrey Chang <jjeffreychang@gmail.com>" + "\r\n" +
		"From: " + user + "\r\n" +
			"To: amlwwalker@gmail.com\r\n" +
			"HtmlContent-Type: text/html; charset=\"utf-8\"\r\n" +
			"Subject: email scored 75%\r\n\r\n" +
			body)
	log.Println(string(messageStr))
	// Place messageStr into message.Raw in base64 encoded format
	message.Raw = b64.URLEncoding.EncodeToString(messageStr)
	message.ThreadId = thread
	_, err := this.InsertMail(user, message) //don't need to know the id of the inserted message
	return err
}

func (this *Gmail) InsertMail(user string, message gmail.Message) (string, error) {
	// Insert the message
	if res, err := this.Srv.Users.Messages.Insert(user, &message).Do(); err != nil {
		log.Printf("Error inserting mail: %v", err)
		return "", err
	} else {
		log.Printf("Message Inserted!\n")
		return res.Id, nil
	}
}

// TrashMail moves a message in a thread to the trash
// this is not a permanent delete
func (this *Gmail) TrashMail(user, mId string) error {
	// Trash the message
	log.Println("About to trash message: ", mId, " for: ", user)
	if msg, err := this.Srv.Users.Messages.Trash(user, mId).Do(); err != nil {
		log.Printf("Error trashing mail [%s]: %v", mId, err)
		return err
	} else {
		log.Printf("Message trashed!: [%s]\n", msg.Id)
		return nil
	}
}
// Delete deletes a mail immediately.
// Warning, dangerous
func (this *Gmail) DeleteMail(user, mId string) error {
	// Trash the message
	log.Println("About to delete message: ", mId, " for: ", user)
	if err := this.Srv.Users.Messages.Delete(user, mId).Do(); err != nil {
		log.Printf("Error deleting mail: %v", err)
		return err
	} else {
		log.Printf("Message Deleted!: [%s]\n", mId)
		return nil
	}
}

// ThreadModify puts a label on a thread
func (this *Gmail) ThreadModify(user, tId, labelMessage string, addLabel bool) error {

	var modifyThread gmail.ModifyThreadRequest
	if addLabel {
		log.Println("adding label: ", labelMessage)
		modifyThread.AddLabelIds = append(modifyThread.AddLabelIds, labelMessage)
	} else {
		log.Println("removing label: ", labelMessage)
		modifyThread.RemoveLabelIds = append(modifyThread.RemoveLabelIds, labelMessage)
	}

	// Trash the message
	if _, err := this.Srv.Users.Threads.Modify(user, tId, &modifyThread).Do(); err != nil {
		log.Printf("Error modifying message mail [%s]: %v", tId, err)
		return err
	} else {
		log.Printf("Message modified!\n")
		return nil
	}
}

// MessageModify puts a label on a message
func (this *Gmail) MessageModify(user, mId, labelMessage string, addLabel bool) error {

	var modifyMessage gmail.ModifyMessageRequest
	if addLabel {
		log.Println("adding label: ", labelMessage)
		modifyMessage.AddLabelIds = append(modifyMessage.AddLabelIds, labelMessage)
	} else {
		log.Println("removing label: ", labelMessage)
		modifyMessage.RemoveLabelIds = append(modifyMessage.RemoveLabelIds, labelMessage)
	}

	// Trash the message
	if _, err := this.Srv.Users.Messages.Modify(user, mId, &modifyMessage).Do(); err != nil {
		log.Printf("Error modifying message mail [%s]: %v", mId, err)
		return err
	} else {
		log.Printf("Message modified!\n")
		return nil
	}
}

func (this *Gmail) CreateLabel(user, labelName string) error {
	var label gmail.Label
	label.Name = labelName
	label.MessageListVisibility = "show"
	label.LabelListVisibility = "labelShow"
	if lbl, err := this.Srv.Users.Labels.Create(user, &label).Do(); err != nil {
		log.Printf("Error creating label [%s]: %v", labelName, err)
		return err
	} else {
		log.Printf("label created, id: [%s]!\n", lbl.Id)
		return nil
	}
}

