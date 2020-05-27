package main

import (
	cmd "arvo/api"
	"arvo/docs"
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log"
)

var (
	addr        = flag.String("listen-address", "0.0.0.0", "The address to listen on for ")
	swaggerHost = flag.String("swagger-host", "localhost", "The hostname that appears in swagger")
	port        = flag.Int("port", 8162, "The port to listen on")
	conf        = flag.String("conf", "arvo.yaml", "The path to the config file.")
)

// @version 0.0.1
// @description This is a small api to help you clean up hieradata
// @BasePath /v1

func main() {
	flag.Parse()
	var c cmd.Conf
	c.GetConf(*conf)
	// set defaults for database
	if c.DB.Host == "" {
		c.DB.Host = "localhost"
	}
	if c.DB.Port == 0 {
		c.DB.Port = 27017
	}
	if c.DB.Database == "" {
		c.DB.Database = "arvo"
	}
	if c.DB.Type == "" {
		c.DB.Type = "mongo"
	}
	if c.DB.AuthDatabase == "" {
		c.DB.AuthDatabase = "admin"
	}

	if c.DataDir == "" {
		c.DataDir = "/etc/puppetlabs/code/environment/production/data"

	}
	if c.KeyTTLMinutes == 0 {
		c.KeyTTLMinutes = 15
	}

	if c.HieraFile == "" {
		c.HieraFile = "/etc/puppetlabs/puppet/hiera.yaml"
	}

	if c.Puppet.Host == "" {
		c.Puppet.Host = "localhost"
	}

	if c.Puppet.Port <= 0 {
		c.Puppet.Port = 8080
	}

	//m := cmd.YamlFileToStringMap("C:\\Users\\tieyz_admin\\Desktop\\test\\common.yaml")
	//_, err := cmd.InsertHieraIdEntry("common",m, c.DB)
	//if err != nil {
	//	log.Println()
	//}
	//
	//res1 , err := cmd.GetOneHieraEntry(c.DB, "common")
	//if res1 != nil {
	//	for key, _ := range *res1 {
	//		log.Println(key)
	//	}
	//
	//}
	router := gin.Default()
	host := fmt.Sprintf("%s:%d", *addr, *port)
	hostSwag := fmt.Sprintf("%s:%d", *swaggerHost, *port)

	docs.SwaggerInfo.Title = "Arvo is puppet hiera helper api"
	docs.SwaggerInfo.Host = hostSwag

	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  false,
		AllowOriginFunc:  func(origin string) bool { return true },
		MaxAge:           86400,
	}))

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/v1")
	{

		v1.POST("/keys", cmd.PostKeyEndpoint(c))
		v1.GET("/keys", cmd.GetKeysForAllCertnamesEndpoint(c))
		v1.GET("/keys/:id", cmd.GetKeysForOneCertnamesEndpoint(c))

		// we must be able to set our own hierarchies as well to use with the api
		v1.GET("/hierarchy", cmd.GetHierarchyEndPoint(c))
		v1.GET("/hierarchy/:id", cmd.GetHierarchyForCertnameEndpoint(c))

		v1.GET("/clean-all/refresh", cmd.CleanAllRefreshEndpoint(c))
		v1.GET("/clean-all", cmd.CleanAllEndpoint(c))
		v1.GET("/clean/:id", cmd.GetKeyLocationsForCertnameEndpoint(c))

		v1.GET("/hiera/path", cmd.HieraIdsEndpoint(c))
		v1.GET("/hiera/path/:id", cmd.HieraIdEndpoint(c))
		v1.POST("/hiera/path/:id", cmd.HieraIdInsertEndpoint(c))
		v1.DELETE("/hiera/path/:id", cmd.DeleteHieraIdEndpoint(c))
		v1.PUT("/hiera/path/:id", cmd.HieraIdUpdateEndpoint(c))
		// we should also be able to get by key, so we can see occurence of key and data and were it is overridden

		// we should also be able to have a variables endpoint that we can use inside the hiera entries
		// global variables and path specific overrides, also allow facts?
	}

	err := router.Run(host)
	if err != nil {
		log.Fatal(err.Error())
	}

}
