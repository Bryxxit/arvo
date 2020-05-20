package main

import (
	cmd "arvo/cmd"
	"flag"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
)

var (
	addr = flag.String("listen-address", ":8162", "The address to listen on for ")
	conf = flag.String("conf", "arvo.yaml", "The path to the config file.")
)

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

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  false,
		AllowOriginFunc:  func(origin string) bool { return true },
		MaxAge:           86400,
	}))
	// Simple group: v1
	v1 := router.Group("/v1")
	{
		v1.POST("/keys", cmd.PostKeyEndpoint(c))
		v1.GET("/keys", cmd.GetKeysForAllCertnamesEndpoint(c))
		v1.GET("/keys/:id", cmd.GetKeysForOneCertnamesEndpoint(c))
		v1.GET("/hierarchy", cmd.GetHierarchyEndPoint(c))
		v1.GET("/hierarchy/:id", cmd.GetHierarchyForCertnameEndpoint(c))
		v1.GET("/clean", cmd.GetKeyLocationsForCertnameEndpoint(c))
		v1.GET("/clean/:id", cmd.GetKeyLocationsForCertnameEndpoint(c))

	}

	err := router.Run(*addr)
	if err != nil {
		log.Fatal(err.Error())
	}

}
