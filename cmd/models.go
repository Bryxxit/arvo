package cmd

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"time"
)

var LAYOUT = "2006-01-02T15:04:05-0700"

type Conf struct {
	DB            Database       `yaml:"db"`
	KeyTTLMinutes int            `yaml:"key_ttl_minutes"`
	DataDir       string         `yaml:"datadir"`
	Puppet        PuppetDBConfig `yaml:"puppet"`
	HieraFile     string         `yaml:"hiera_file"`
}

func (c *Conf) GetConf(configFile string) *Conf {

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Println("Unmarshal: %v", err)
	}

	return c
}

type JSONID struct {
	ID string `uri:"id" binding:"required,uuid"`
}

type PuppetDBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	SSL      bool   `yaml:"ssl"`
	Key      string `yaml:"key"`
	Ca       string `yaml:"ca"`
	Insecure bool   `yaml:"insecure"`
	Cert     string `yaml:"cert"`
}

type LookupYaml struct {
	Version   string           `json:"version"yaml:"version"`
	Defaults  LookupDefaults   `json:"defaults"yaml:"defaults"`
	Hierarchy []LookupHierachy `json:"hierarchy"yaml:"hierarchy"`
}

type LookupDefaults struct {
	Datadir string `json:"datadir"yaml:"datadir"`
}

type LookupHierachy struct {
	Name      string                 `json:"name"yaml:"name"`
	LookupKey string                 `json:"name"lookup_key:"lookup_key"`
	Paths     *[]string              `json:"paths"lookup_key:"paths"`
	Path      *string                `json:"path"lookup_key:"path"`
	Uris      *[]string              `json:"uris"lookup_key:"uris"`
	Options   map[string]interface{} `json:"options"lookup_key:"options"`
}

type Host struct {
	Certname string          `json:"certname"`
	Keys     []HieraKeyEntry `json:"keys"`
}

type HieraKeyEntry struct {
	Key  string    `json:"key"`
	Date time.Time `json:"date"`
}

type HieraLogEntry struct {
	Certname string  `json:"certname"`
	Key      string  `json:"key"`
	Date     *string `json:"date_string"`
}

type HieraLogHost struct {
	ID      string          `bson:"_id"json:"id"`
	Entries []HieraLogEntry `json:"keys"`
}

func (c *LookupYaml) getConf(configFile string) {

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Println("Unmarshal: %v", err)
	}
}

type Database struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Database   string `yaml:"db"`
	Type       string `yaml:"type"`
	connection mongo.Client
}

type HieraKeyFullEntry struct {
	//Paths []string `json:"path"yaml:"path"`
	Key      string                   `json:"key"yaml:"key"`
	SubKeys  []string                 `json:"sub_keys"yaml:"sub_keys"`
	InLookup bool                     `json:"in_lookup"yaml:"in_lookup"`
	Values   []HieraKeyFullValueEntry `json:"in_lookup"yaml:"values"`
}

type HieraKeyFullValueEntry struct {
	Path  string      `json:"path"yaml:"path"`
	Key   string      `json:"key"yaml:"key"`
	Type  string      `json:"type"yaml:"type"`
	Value interface{} `json:"value"yaml:"value"`
}

type HieraMatch struct {
	Key       string            `json:"key"yaml:"key"`
	Locations []string          `json:"locations"yaml:"locations"`
	Matches   []HieraMatchEntry `json:"matches"yaml:"matches"`
}

type HieraMatchEntry struct {
	Path1 string `json:"path1"yaml:"path1"`
	Path2 string `json:"path2"yaml:"path2"`
	Key   string `json:"key"yaml:"key"`
}

type HierarchyResult struct {
	Paths     []string `json:"paths"yaml:"paths"`
	Variables []string `json:"vars"yaml:"vars"`
}

func NewClient(host string, port int, database string) (*mongo.Client, error) {
	// create the connection uri
	uri := fmt.Sprintf(`mongodb://%s:%d`,
		host,
		port,
	)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return client, err
}
