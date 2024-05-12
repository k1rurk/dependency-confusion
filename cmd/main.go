package main

import (
	"dependency-confusion/internal/api"
	log "github.com/sirupsen/logrus"
	"dependency-confusion/internal/database"
	"dependency-confusion/internal/dns_exfiltrate"
	"dependency-confusion/runconfig"
	"github.com/gin-gonic/gin"
)

// @title           Swagger Example API
// @version         1.0
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:9000
// @BasePath  /api/v1


// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	sql, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer sql.Close()
	cfg, err := runconfig.New().ReadInConfig()
	if err != nil {
		log.Warnf("error reading config: %v", err)
	}

	// start DNS
	dns_exfiltrate.RunDNS(sql, &cfg.DNSConfig)

	//gin.SetMode(gin.ReleaseMode)
	gin.SetMode(gin.DebugMode)

	r := api.InitRouter(sql, cfg)

	if err := r.Run(":9000"); err != nil {
		log.Fatal(err)
	}
}
