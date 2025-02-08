package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/dartt0n/realtime-chat-backend/controllers"
	"github.com/dartt0n/realtime-chat-backend/db"
	"github.com/dartt0n/realtime-chat-backend/forms"
	"github.com/dartt0n/realtime-chat-backend/kv"
	"github.com/gin-contrib/gzip"
	uuid "github.com/google/uuid"
	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// CORS (Cross-Origin Resource Sharing)
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "X-Requested-With, Content-Type, Origin, Authorization, Accept, Client-Security-Token, Accept-Encoding, x-access-token")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			fmt.Println("OPTIONS")
			c.AbortWithStatus(200)
		} else {
			c.Next()
		}
	}
}

// Generate a unique ID and attach it to each request for future reference or use
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := uuid.New()
		c.Writer.Header().Set("X-Request-Id", uuid.String())
		c.Next()
	}
}

var auth = new(controllers.AuthController)

// JWT Authentication middleware attached to each request that needs to be authenitcated to validate the access_token in the header
func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth.TokenValid(c)
		c.Next()
	}
}

func main() {
	var err error

	//Load the .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatal("error: failed to load the env file")
		}
	}

	if os.Getenv("ENV") == "PRODUCTION" {
		gin.SetMode(gin.ReleaseMode)
	}

	//Start the default gin server
	r := gin.Default()

	//Custom form validator
	binding.Validator = new(forms.DefaultValidator)

	r.Use(CORSMiddleware())
	r.Use(RequestIDMiddleware())
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	err = db.InitMongo(os.Getenv("DB_URI"), os.Getenv("DB_NAME"))
	if err != nil {
		log.Fatal("failed to connect to database: ", err)
	}

	redisDb, err := strconv.ParseInt(os.Getenv("REDIS_DB"), 0, 0)
	if err != nil {
		log.Fatal("failed to parse REDIS_DB env variable: ", err)
	}
	err = kv.InitRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASS"), int(redisDb))
	if err != nil {
		log.Fatal("failed to connect to key-value store: ", err)
	}

	v1 := r.Group("/v1")
	{
		health := new(controllers.HealthController)
		v1.GET("/health", health.Health)

		user := new(controllers.UserController)
		v1.POST("/user/login", user.Login)
		v1.POST("/user/signup", user.Register)
		v1.GET("/user/logout", user.Logout)

		auth := new(controllers.AuthController)
		v1.POST("/token/refresh", auth.Refresh)
	}

	port := os.Getenv("PORT")

	log.Printf("PORT: %s; ENV: %s; SSL: %s", port, os.Getenv("ENV"), os.Getenv("SSL"))

	if os.Getenv("SSL") == "TRUE" {

		//Generated using sh generate-certificate.sh
		SSLKeys := &struct {
			CERT string
			KEY  string
		}{
			CERT: "./cert/myCA.cer",
			KEY:  "./cert/myCA.key",
		}

		r.RunTLS(":"+port, SSLKeys.CERT, SSLKeys.KEY)
	} else {
		r.Run(":" + port)
	}

}
