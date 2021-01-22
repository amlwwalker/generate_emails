package user

import (
	"amlwwalker/gmail-backend/backend/pkg/database"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"log"
	"net/http"
)

//type Account struct {
//	Customers []customers.Customer `gorm:"foreignkey:ID"`
//}

type GoogleProfile struct {
	ID string `"json":"id"`
	Email string `"json":"email"`
	Name string `"json":"name"`
	Picture string `"json":"picture"`
	Locale string `"json":"locale"`
}


func GetAliases(accessToken string) (GoogleProfile, error) {
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return (GoogleProfile{}), err
	}
	var profile GoogleProfile
	json.NewDecoder(response.Body).Decode(&profile)
	defer response.Body.Close()
	return profile, nil
}
func GetProfile(accessToken string) (GoogleProfile, error) {
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return (GoogleProfile{}), err
	}
	var profile GoogleProfile
	json.NewDecoder(response.Body).Decode(&profile)
	defer response.Body.Close()
	return profile, nil
}

func GetOrganisation(accessToken string) (error) {
	response, err := http.Get("https://www.googleapis.com/admin/directory/v1/users?access_token=" + accessToken)
	if err != nil {
		log.Printf("organisation error: %+v\n", err)
		return err
	}
	log.Printf("body: %+v\n", response.Body)
	return nil
}

//Processors do large group effors on data
//This one concentrates on processing an incoming pub/sub request
func CreateClient(ID uint) (*http.Client, *oauth2.Token, error) {
	//get the access token for the user from the database
	if accessToken, err := database.FindUserAccessToken(ID); err != nil {
		log.Println("error finding access token: ", err, " for user: ", ID)
		return &(http.Client{}), &(oauth2.Token{}), err
	} else {
		log.Println("access token found is ", accessToken)
		//accessToken is the whole token, we now need to generate a new client and re-retrieve the token
		tokenSource := GoogleOauthConfig.TokenSource(oauth2.NoContext, accessToken)
		client := oauth2.NewClient(oauth2.NoContext, tokenSource)
		updatedToken, err := tokenSource.Token()
		if err != nil {
			fmt.Println("error renewing token", err)
			return &(http.Client{}), &(oauth2.Token{}), err
		}
		// client = googleOauthConfig.Client(oauth2.NoContext, accessToken)
		return client, updatedToken, nil
	}

}

var GoogleOauthConfig *oauth2.Config
