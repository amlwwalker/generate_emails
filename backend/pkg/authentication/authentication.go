package authentication


import (
	"amlwwalker/gmail-backend/backend/pkg/database"
	"amlwwalker/gmail-backend/backend/pkg/gmail"
	"github.com/dgrijalva/jwt-go"
	"errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var OauthStateString string
//https://skarlso.github.io/2016/06/12/google-signin-with-go/
func ConfigureAuthentication() {
	user.GoogleOauthConfig = &oauth2.Config{
		RedirectURL:  os.Getenv("SERVER_URL") + os.Getenv("G_AUTH_REDIRECT_URL"),
		ClientID:     os.Getenv("G_AUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("G_AUTH_CLIENT_SECRET"),
		Scopes: []string{"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/gmail.send",
			//"https://www.googleapis.com/auth/contacts.readonly",
			"https://www.googleapis.com/auth/drive.file",
			//"https://www.googleapis.com/auth/spreadsheets",
		},
		Endpoint: google.Endpoint,
	}
	// Some random string, random for each request
	OauthStateString = os.Getenv("STATE_STRING")
	log.Println("config ", user.GoogleOauthConfig.ClientID)
}

const oauthGoogleUrlAPI = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="

//func ProcessNewUserAccountWithCode(code string) (string, string, time.Time, error) {
//	expiration := time.Now()
//	//state has hopefully been managed by the front end
//	token, err := GoogleOauthConfig.Exchange(oauth2.NoContext, code)
//	if err != nil {
//		log.Printf("Code exchange failed with '%s'\n", err)
//		// this.Redirect("/close", http.StatusTemporaryRedirect)
//		return "", "", expiration, err
//	}
//	log.Printf("token recovered: %s\n", token)
//	//now we have a token, we should process the account
//	if usr, err := processAccountForStorage(token); err != nil {
//		return "", "", expiration, err
//	} else {
//		//if err := database.UpdateUserAccessToken(usr.ID, token); err != nil {
//		//	log.Printf("could not store the user access token, %+v, err, %+v\r\n", token, err)
//		//	return usr.Email, "", expiration, errors.New("couldnt store access token " + err.Error())
//		//}
//		expiration = expiration.Add(120 * time.Minute)
//		//token := jwt.New(jwt.SigningMethodHS256)
//		// Set claims
//		// This is the information which frontend can use
//		// The backend can also decode the token and get admin etc.
//
//		claims := jwt.MapClaims{}
//		claims["email"] = usr.Email
//		claims["admin"] = false
//		claims["exp"] = expiration
//		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
//		// Generate encoded token and send it as response.
//		// The signing string should be secret (a generated UUID          works too)
//		t, err := token.SignedString(jwtSecret)
//		return usr.Email, t, expiration, err
//	}
//}
func ProcessNewUserAccount(gResponse GResponse) (string, string, time.Time, error) {
	expiration := time.Now()
	//todo: state should be set on the session so its different for everyone
	if gResponse.State != OauthStateString {
		log.Printf("invalid oauth state, expected '%s', got '%s'\n", OauthStateString, gResponse.State)
		// this.Redirect("/close", http.StatusTemporaryRedirect)
		return "", "", expiration, errors.New("State does not equal state string")
	}

	token, err := user.GoogleOauthConfig.Exchange(oauth2.NoContext, gResponse.Code)
	if err != nil {
		log.Printf("Code exchange failed with '%s'\n", err)
		// this.Redirect("/close", http.StatusTemporaryRedirect)
		return "", "", expiration, err
	}
	log.Printf("token recovered: %s\n", token)
	//now we have a token, we should process the account
	if usr, err := processAccountForStorage(token); err != nil {
		return "", "", expiration, err
	} else {
		//if err := database.UpdateUserAccessToken(usr.ID, token); err != nil {
		//	log.Printf("could not store the user access token, %+v, err, %+v\r\n", token, err)
		//	return usr.Email, "", expiration, errors.New("couldnt store access token " + err.Error())
		//}
		//expiration = expiration.Add(30 * time.Minute)
		//token := jwt.New(jwt.SigningMethodHS256)
		// Set claims
		// This is the information which frontend can use
		// The backend can also decode the token and get admin etc.
		expirationTime := time.Now().Add(72 * time.Hour)
		// Create the JWT claims, which includes the username and expiry time
		claims := &Claims{
			Email: usr.Email,
			Id: usr.ID,
			Admin: usr.SuperAdmin,
			StandardClaims: jwt.StandardClaims{
				// In JWT, the expiry time is expressed as unix milliseconds
				ExpiresAt: expirationTime.Unix(),
			},
		}
		//claims := jwt.MapClaims{}
		//claims["email"] = usr.Email
		//claims["id"] = usr.ID
		//claims["admin"] = false
		//claims["exp"] = expiration
		log.Println("setting claims to ", claims)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		// Generate encoded token and send it as response.
		// The signing string should be secret (a generated UUID          works too)
		log.Println("using jwtSecret ", jwtSecret)
		t, err := token.SignedString(jwtSecret)
		return usr.Email, t, expiration, err
	}
}
func getUserDataFromGoogle(accessToken string) ([]byte, error) {
	// Use code to get token and get user info from Google.

	response, err := http.Get(oauthGoogleUrlAPI + accessToken)
	if err != nil {
		return nil, errors.New("failed getting user info: " + err.Error())
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("failed read response: "+ err.Error())
	}
	return contents, nil
}
func processAccountForStorage(token *oauth2.Token) (database.User, error) {

	var profile user.GoogleProfile
	var err error
	if profile, err = user.GetProfile(token.AccessToken); err != nil {
		return (database.User{}), err
	}

	log.Printf("profile: %+v\n", profile)

	var t database.AccessToken
	//t.UserID = id
	t.AccessToken = token.AccessToken
	t.RefreshToken = token.RefreshToken
	t.Expiry = token.Expiry
	t.TokenType = token.TokenType

	var u database.User
	u.Email = profile.Email
	u.ImageUrl = profile.Picture
	u.Name = profile.Name
	u.Token = t
	u.EmailQuota = 50
	var res database.User
	// // var found bool
	if res, _, err = database.FindOrCreate(u); err != nil {
		log.Println("create user error: ", err )
		//if there is an error, nothing else matters
		return (database.User{}), err
	}

	//if err := database.ToggleValidUser(u.ID, true); err != nil {
	//	fmt.Println("setting validity to true failed - the client failed with error: ", err)
	//	return (database.User{}), err
	//}
	//

	return res, nil

}
