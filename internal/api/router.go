package api

import (
	"database/sql"
	"dependency-confusion/docs" // which is the generated folder after swag init
	"dependency-confusion/internal/api/handlers"
	"dependency-confusion/internal/middleware"
	"dependency-confusion/runconfig"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter(db *sql.DB, config *runconfig.Config) *gin.Engine {
	r := gin.Default()

	r.Use(gin.Logger())

	h := handlers.New(db, config)
	// if gin.Mode() == gin.ReleaseMode {
	// }
	r.Use(middleware.Cors())
	// r.Use(middleware.RateLimiter(rate.Every(1*time.Minute), 60)) // 60 requests per minute
	r.Static("/site", "../public")
	r.GET("/", handlers.RootDir)
	docs.SwaggerInfo.BasePath = "/api/v1"
	v1 := r.Group("/api/v1")
	{
		v1.POST("/domain", h.FindDomainPackages)
		v1.POST("/github/org", h.GetOrgRepos)
		v1.POST("/github/repo", h.GetRepo)
		v1.POST("/parser/file", h.ParseFile)
		v1.POST("/parser/directory", h.ParseDir)
		v1.GET("/dns", h.GetDNSData)
		v1.POST("/registry", h.SendPackageRegistry)
		v1.GET("/report", h.GenerateReport)
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	return r
}
