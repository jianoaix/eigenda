package ejector

import (
	"net/http"
	"os"
	"time"

	"github.com/Layr-Labs/eigenda/disperser/dataapi/docs"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files" // swagger embed files
)

type server struct {
	ejector Ejector
}

func (s *server) Start() error {
	router := gin.New()
	basePath := "/api/v1"
	docs.SwaggerInfo.BasePath = basePath
	docs.SwaggerInfo.Host = os.Getenv("SWAGGER_HOST")
	v1 := router.Group(basePath)
	{
		ejection := v1.Group("/ejection")
		ejection.GET("/", s.handleEjection)

		swagger := v1.Group("/swagger")
		{
			swagger.GET("/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
		}
	}

	router.GET("/", func(g *gin.Context) {
		g.JSON(http.StatusAccepted, gin.H{"status": "OK"})
	})

	router.Use(logger.SetLogger(
		logger.WithSkipPath([]string{"/"}),
	))

	config := cors.DefaultConfig()
	config.AllowOrigins = s.allowOrigins
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "HEAD", "OPTIONS"}

	if s.serverMode != gin.ReleaseMode {
		config.AllowOrigins = []string{"*"}
	}
	router.Use(cors.New(config))

	srv := &http.Server{
		Addr:              s.socketAddr,
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start the ejector
	s.ejector.Start()

	return nil
}
