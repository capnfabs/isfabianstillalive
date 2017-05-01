// go install github.com/capnfabs/isfabianstillalive.com/cmd/webroot && heroku local
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/mattn/go-sqlite3"
)

var TwilioInboundPassword = os.Getenv("TWILIO_INBOUND_PASSWORD")
var DbUrl = os.Getenv("DATABASE_URL")

type Message struct {
	gorm.Model
	WhenReceived  time.Time
	StringContent string
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	if TwilioInboundPassword == "" {
		log.Fatal("$TWILIO_INBOUND_PASSWORD must be set")
	}

	var err error
	var db *gorm.DB
	if DbUrl == "" {
		db, err = gorm.Open("sqlite3", "local.db")
	} else {
		db, err = gorm.Open("postgres", DbUrl)
	}
	if err != nil {
		panic("failed to connect to database " + DbUrl)
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&Message{})

	// Create

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		lastUpdate := "Hi Mum!"
		data := struct {
			LastUpdate string
		}{
			lastUpdate,
		}
		c.HTML(http.StatusOK, "index.tmpl.html", data)
	})

	accounts := gin.Accounts{
		"twilio": TwilioInboundPassword,
	}
	twilioAuth := gin.BasicAuth(accounts)

	router.Use(twilioAuth).POST("/inbound_sms", func(c *gin.Context) {
		err := c.Request.ParseForm()
		if err != nil {
			panic(err)
		}

		messageBody := c.Request.Form.Get("Body")

		db.Create(&Message{
			WhenReceived:  time.Now(),
			StringContent: messageBody,
		})
	})

	router.Run(":" + port)
}
