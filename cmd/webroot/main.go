// go install github.com/capnfabs/isfabianstillalive.com/cmd/webroot && heroku local
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"errors"

	humanize "github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/mattn/go-sqlite3"
)

var TwilioInboundPassword = os.Getenv("TWILIO_INBOUND_PASSWORD")
var DbUrl = os.Getenv("DATABASE_URL")
var DevMode = os.Getenv("DEV_MODE") == "1"

var TimestampZone, _ = time.LoadLocation("Europe/Berlin")

type Message struct {
	gorm.Model
	WhenReceived  time.Time `gorm:"index"`
	StringContent string
}

func (m *Message) FriendlyReceived() string {
	return humanize.Time(m.WhenReceived)
}

func (m *Message) TimestampReceived() string {
	return m.WhenReceived.In(TimestampZone).Format(time.RFC1123)
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

	if DevMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		var lastUpdates []Message
		db.Order("when_received desc").Limit(10).Find(&lastUpdates)
		data := struct {
			Newest       *Message
			OtherUpdates []Message
		}{}
		if len(lastUpdates) > 0 && time.Since(lastUpdates[0].WhenReceived) <= 7*24*time.Hour {
			data.Newest = &lastUpdates[0]
			data.OtherUpdates = lastUpdates[1:]
		} else {
			data.OtherUpdates = lastUpdates
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
		if messageBody == "" {
			c.AbortWithError(http.StatusBadRequest, errors.New("Expected a Body, didn't get one"))
		}

		db.Create(&Message{
			WhenReceived:  time.Now(),
			StringContent: messageBody,
		})
	})

	router.Run(":" + port)
}
