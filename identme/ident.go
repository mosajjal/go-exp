package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		clientIP := c.ClientIP() + "\n"
		c.Data(http.StatusOK, "text/html", []byte(clientIP))
	})
	r.GET("/json", func(c *gin.Context) {
		clientIP := c.ClientIP()
		c.JSON(200, gin.H{
			"ip": clientIP,
		})
	})
	log.Fatalln(r.Run(":80"))
}
