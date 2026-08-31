package main

import (
	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/config"
)

func main() {
	//inisialiasai Gin
	router := gin.Default()

	//memuat environment variables
	config.LoadEnv()

	//membuat route dengan method GET
	router.GET("/", func(c *gin.Context) {

		//return response JSON
		c.JSON(200, gin.H{
			"message": "Hello World!",
		})
	})

	//mulai server dengan port dari environment variable
	router.Run(":" + config.GetEnv("APP_PORT", "3000"))
}
