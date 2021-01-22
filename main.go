package main

import (
	"amlwwalker/gmail-backend/backend/pkg/authentication"
	"amlwwalker/gmail-backend/backend/pkg/database"
	"amlwwalker/gmail-backend/backend/pkg/utilities"
	"log"
	"net/http"
	"os"
	"sync"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	if os.Getenv("JWT_SECRET") == "" {
		log.Println("JWT_SECRET cannot be blank")
		os.Exit(2)
	}
	if os.Getenv("SERVER_PORT") == "" {
		log.Println("SERVER_PORT cannot be blank")
		os.Exit(2)
	}

	database.InitDB()
	authentication.ConfigureAuthentication()

	// Set the router as the default one shipped with Gin
	router := gin.Default()
	config := cors.DefaultConfig()
	log.Println("PR_SERVER", os.Getenv("PR_SERVER"))
	config.AllowOrigins = []string{"https://envoye.app", os.Getenv("PR_SERVER")}
	config.AllowCredentials = true
	config.AddAllowHeaders("x-access-token")
	//config := cors.DefaultConfig()
	// config.AllowAllOrigins = true
	//cors.New(config)
	router.Use(cors.New(config))

	router.GET("/healthz", func(c *gin.Context) {c.Status(http.StatusOK); return })
	auth := router.Group("/auth")
	
	{
		auth.GET("/login", authentication.OAuthGoogle) //starts an authentication
		auth.GET("/callback", authentication.Callback) //authenticates a user
		auth.GET("/private", authentication.Session) //checks for the authd user
	}

	var wg sync.WaitGroup
	go utilities.Cron(&wg)

	// Start the app
	router.Run(":"+os.Getenv("SERVER_PORT"))
}

