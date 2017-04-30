package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var TwilioKey = os.Getenv("TWILIO_KEY")
var DbUrl = os.Getenv("DATABASE_URL")

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

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

	router.POST("/inbound_sms", func(c *gin.Context) {

	})

	router.Run(":" + port)
}
