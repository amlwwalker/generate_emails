package authentication

import (
	"amlwwalker/gmail-backend/backend/pkg/database"
	"amlwwalker/gmail-backend/backend/pkg/gmail"
	"github.com/dgrijalva/jwt-go"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

// API is a defined as struct bundle
// for api. Feel free to organize
// your app as you wish.
type Auth struct{}

// Create a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type Claims struct {
	Email string `json:"email"`
	Id uint `json:"id"`
	Admin bool `json:"admin"`
	jwt.StandardClaims
}

//https://medium.com/monstar-lab-bangladesh-engineering/jwt-auth-in-go-dde432440924
func RetrieveAuthClaims(c *gin.Context) (Claims, bool, error) {
	jwtKey := jwtSecret
	//obtain session token from the requests cookies
	ck, err := c.Request.Cookie("token")
	if err != nil {
		log.Print("couldn't retrieve cookie ", err)
		return Claims{}, false, err
	}

	// Get the JWT string from the cookie
	tokenString := ck.Value

	if tokenString == "null" || tokenString == "" { //js nil check
		fmt.Println("token string is ", tokenString)
		return Claims{}, false, errors.New("empty token")
	}
	// Initialize a new instance of `Claims`
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			log.Println("Invalid Token Signature ", err)
			//c.JSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "message": "Invalid Token Signature"})
			//c.Abort()
			return Claims{}, false, err
		}
		//c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Bad Request"})
		//c.Abort()
		log.Println("error validating token ", err)
		return Claims{}, false, err
	}

	if !token.Valid {
		log.Println("Invalid Token")
		//c.JSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "message": "Invalid Token"})
		//c.Abort()
		return Claims{}, false, errors.New("token is not vali")
	}
	log.Println("retrieved claims and valid token", *claims)
	return *claims, true, nil
	//if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
	//	log.Println("claimes are real apparently ", claims)
	//	return claims, true
	//}
	//return nil, false
}

func GetAccessTokenFromSession(userId uint) (string, error){
	_, token, _ := user.CreateClient(userId)
	return token.AccessToken, nil
}
// retrieveAuthClaims intercepts the requests, and check for a valid jwt token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		user, isAuthenticated, err := RetrieveAuthClaims(c)
		if !isAuthenticated {
			log.Println("authMiddleware - the user is not authenticated")
			c.Abort()
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "unauthorized", "error": err})
			return
		}
		log.Println("accepted auth middleware for ", user)
		c.Next()
	}
}

func Session(c *gin.Context) {
	claims, isAuthenticated, err := RetrieveAuthClaims(c)
	if !isAuthenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "unauthorized", "error": err})
		return
	}
	if user, err := database.RetrieveUserProfileById(claims.Id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "user": claims})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{"success": true, "user": user})
	}
}
//
//func (a *Auth) private(c *gin.Context) {
//	_, isAuthenticated := retrieveAuthClaims(c, jwtSecret)
//	if !isAuthenticated {
//		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "unauthorized"})
//		return
//	}
//	if user, exists := c.Get("user"); !exists {
//		c.JSON(http.StatusBadRequest, gin.H{"message": "error user field doesn't exist"})
//	} else {
//		log.Println("user ", user)
//		if u, ok := user.(*jwt.Token); !ok {
//			c.JSON(http.StatusBadRequest, gin.H{"message": "error user token doesn't exist"})
//		} else {
//			claims := u.Claims.(jwt.MapClaims)
//			name := claims["name"].(string)
//			c.JSON(http.StatusOK, gin.H{"message": "Welcome "+name+"!"})
//		}
//	}
//
//}
// func CreateClient(usrId uint) (*http.Client, *oauth2.Token, error) {
// 	//get the access token for the user from the database
// 	if accessToken, err := database.FindUserAccessToken(usrId); err != nil {
// 		log.Println("error finding access token: ", err, " for user: ", ID)
// 		return &(http.Client{}), &(oauth2.Token{}), err
// 	} else {

// 		//accessToken is the whole token, we now need to generate a new client and re-retrieve the token
// 		tokenSource := GoogleOauthConfig.TokenSource(oauth2.NoContext, accessToken)
// 		client := oauth2.NewClient(oauth2.NoContext, tokenSource)
// 		updatedToken, err := tokenSource.Token()
// 		if err != nil {
// 			fmt.Println("error renewing token", err)
// 			return &(http.Client{}), &(oauth2.Token{}), err
// 		}

// 		// client = googleOauthConfig.Client(oauth2.NoContext, accessToken)
// 		return client, updatedToken, nil
// 	}

// }
func CurrentUserId(c *gin.Context) (uint, error) {
	claims, isAuthenticated, err := RetrieveAuthClaims(c)
	if !isAuthenticated {
		return 0, errors.New("unauthorized, " + err.Error())
	}
	log.Println("claims ", claims.Id)
	c.Header("HtmlContent-Type", "application/json")

	return claims.Id, nil
}
