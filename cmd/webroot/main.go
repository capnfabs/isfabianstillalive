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
	"github.com/unrolled/secure"
)

// The password that the twilio inbound webhook uses. Must be set.
var TwilioInboundPassword = os.Getenv("TWILIO_INBOUND_PASSWORD")

// Postgres database URL. If not specified, app defaults to a local sqlite database.
var DbUrl = os.Getenv("DATABASE_URL")

// DEVELOPER MODE! A bunch of packages use this paradigm, we set them all with a single
// flag. Defaults to OFF.
var DevMode = os.Getenv("DEV_MODE") == "1"

// We render timestamps in 'long format' using this location. It would be heaps better to
// just dump these in ISO1234 (or whatever) format and then do them in javascript based on client
// timezone but...
var TimestampZone, _ = time.LoadLocation("Europe/Berlin")

// Message represents a text message received from our 'secret' Twilio phone number (yep, that's
// going to bite if I don't add some kind of authentication to ensure that messages are from me).
type Message struct {
	gorm.Model
	// WhenReceived probably isn't necessary because gorm.Model has WhenCreated, but it seems
	// cleaner to me from a separation of concerns perspective to include it, I guess. *shrug*.
	// We also index on it so that lookups are fast.
	WhenReceived time.Time `gorm:"index"`
	// StringContent is the actual, received message content.
	StringContent string
}

// FriendlyReceived returns a 'humanized' version of the timestamp. Again, would be better to do
// this in JS based on the timestamp (see comment on TimestampZone)
func (m *Message) FriendlyReceived() string {
	return humanize.Time(m.WhenReceived)
}

// TimestampReceived returns a precise version of the timestamp (but still for human consumption).
// Would be better to do this formatting in JS - again, see comment on TimestampZone.
func (m *Message) TimestampReceived() string {
	return m.WhenReceived.In(TimestampZone).Format(time.RFC1123)
}

// Returns an open database or panics. The next thing you should do is call `defer db.Close()`.
func mustCreateDatabase() *gorm.DB {
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
	return db
}

func makeSslMiddleware() gin.HandlerFunc {
	secureMiddleware := secure.New(secure.Options{
		SSLRedirect: true,
		FrameDeny:   true,
		// See https://jaketrent.com/post/https-redirect-node-heroku/ for why this is required
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   DevMode,
	})
	return func() gin.HandlerFunc {
		return func(c *gin.Context) {
			err := secureMiddleware.Process(c.Writer, c.Request)

			// If there was an error, do not continue.
			if err != nil {
				c.Abort()
				return
			}

			// If we're redirecting, we're done! Don't do anything else downstream.
			if status := c.Writer.Status(); status > 300 && status < 399 {
				c.Abort()
			}
		}
	}()
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	if TwilioInboundPassword == "" {
		log.Fatal("$TWILIO_INBOUND_PASSWORD must be set")
	}

	db := mustCreateDatabase()
	defer db.Close()

	// Migrate the schema.
	db.AutoMigrate(&Message{})

	if DevMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(makeSslMiddleware())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	// index.html
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

	// auth middleware for the inbound sms endpoint
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
			// We include this because my test requests were adding a stack of blank entries :-/
			c.AbortWithError(http.StatusBadRequest, errors.New("Expected a Body, didn't get one"))
		}

		db.Create(&Message{
			WhenReceived:  time.Now(),
			StringContent: messageBody,
		})
	})

	// Ok, los geht's!
	router.Run(":" + port)
}
