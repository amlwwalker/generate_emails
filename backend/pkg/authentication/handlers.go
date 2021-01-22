package authentication

import (
	"amlwwalker/gmail-backend/backend/pkg/gmail"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
)

//no check done to see if is an administrator
func OAuthGoogle(c *gin.Context) {
	url := user.GoogleOauthConfig.AuthCodeURL(OauthStateString, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
	//this should go to google to oauth the administrator
	//return http.StatusTemporaryRedirect, nil
}
//the googleAuthCallback
func Callback(c *gin.Context) {

	gResponse := GResponse{}
	state := c.Query("state")
	code := c.Query("code")
	gResponse.State = state
	gResponse.Code = code
	log.Printf("gResponse: %+v\n", gResponse)

	email, token, _, err := ProcessNewUserAccount(gResponse)
	if err != nil {
		log.Println("err: " + err.Error())
		c.JSON(http.StatusInternalServerError, err.Error())
	} else {
		// http.SetCookie(c.Writer, &http.Cookie{
		// 	Name:    "token",
		// 	Value:   token,
		// 	Expires: expiration,
		// })

		//now redirect back to the app to store the token
		//c.JSON(http.StatusOK, gin.H{"success": true, "msg": "logged in succesfully", "user": email, "token": token})
		log.Println(gin.H{"success": true, "msg": "logged in succesfully", "user": email, "token": token})
		// c.JSON(http.StatusOK, gin.H{"success": true, "msg": "logged in succesfully", "user": email, "token": token})
		c.Redirect(http.StatusTemporaryRedirect, os.Getenv("G_AUTH_SUCCESS_URL") + "?access="+token)
	}
}
