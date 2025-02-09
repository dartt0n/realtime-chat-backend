package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/dartt0n/realtime-chat-backend/controllers"
	"github.com/dartt0n/realtime-chat-backend/forms"
	"github.com/dartt0n/realtime-chat-backend/kv"
	"github.com/dartt0n/realtime-chat-backend/models"
	"github.com/dartt0n/realtime-chat-backend/service"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
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

func SlogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		rlog := logger.With(
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"client_ip", c.ClientIP(),
			"request_id", requestid.Get(c),
		)

		start := time.Now()
		rlog.Debug("request started")
		c.Next()
		duration := time.Since(start)
		rlog.Info("request completed", "status", c.Writer.Status(), "duration", duration)
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
			slog.Error("failed to load the env file")
			os.Exit(1)
		}
	}

	var logger *slog.Logger
	if os.Getenv("ENV") == "PRODUCTION" {
		gin.SetMode(gin.ReleaseMode)
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	slog.SetDefault(logger)

	//Start the default gin server
	r := gin.Default()

	//Custom form validator
	binding.Validator = new(forms.DefaultValidator)

	r.Use(CORSMiddleware())
	r.Use(requestid.New(requestid.WithCustomHeaderStrKey("X-Request-Id")))
	r.Use(SlogMiddleware(logger))
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	redisDb, err := strconv.ParseInt(os.Getenv("REDIS_DB"), 0, 0)
	if err != nil {
		slog.Error("failed to parse REDIS_DB env variable", "error", err)
		os.Exit(1)
	}
	redisKV, err := kv.NewRedisKV(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASS"), int(redisDb))
	if err != nil {
		slog.Error("failed to connect to key-value store", "error", err)
		os.Exit(1)
	}

	authService := service.NewAuthService(redisKV)
	tinodeService, err := service.NewTinodeService(
		os.Getenv("TINODE_ADDR"),
		models.Topic{ID: os.Getenv("TINODE_TOPIC_ID"), Name: "general"},
		os.Getenv("DB_URI"), os.Getenv("DB_NAME"),
		redisKV, authService)
	if err != nil {
		slog.Error("failed to connect to tinode", "error", err)
		os.Exit(1)
	}

	health := controllers.NewHealthController()
	r.GET("/health", health.Health)

	user := controllers.NewUserController(tinodeService, authService)
	r.POST("/signup", user.Register)
	r.POST("/login", user.Login)
	r.GET("/logout", user.Logout)

	auth := controllers.NewAuthController(authService)
	r.POST("/refresh", auth.Refresh)

	msg := controllers.NewMessageController(tinodeService, authService)
	r.GET("/messages", msg.FetchLast)
	r.POST("/message", msg.SendMsg)

	port := os.Getenv("PORT")

	slog.Info("server starting", "port", port, "env", os.Getenv("ENV"), "ssl", os.Getenv("SSL"))

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
