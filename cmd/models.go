package cmd

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

var LAYOUT = "2006-01-02T15:04:05-0700"

// Conf is the main configuration file for arvo and holds the settings needed to run it
type Conf struct {
	DB            Database       `yaml:"db"`
	KeyTTLMinutes int            `yaml:"key_ttl_minutes"`
	DataDir       string         `yaml:"datadir"`
	Puppet        PuppetDBConfig `yaml:"puppet"`
	HieraFile     string         `yaml:"hiera_file"`
}

// Database holds the database settings to run arvo
type Database struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Database   string `yaml:"db"`
	Type       string `yaml:"type"`
	connection mongo.Client
}

// GetConf is a function that reads in data from a yaml file into a Conf object
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

type HierarchyYamlFile struct {
	Version   string                    `json:"version"yaml:"version"`
	Defaults  HierarchyYamlFileDefaults `json:"defaults"yaml:"defaults"`
	Hierarchy []HierarchyYamlFileEntry  `json:"hierarchy"yaml:"hierarchy"`
}

type HierarchyYamlFileDefaults struct {
	Datadir string `json:"datadir"yaml:"datadir"`
}

type HierarchyYamlFileEntry struct {
	Name      string                 `json:"name"yaml:"name"`
	LookupKey string                 `json:"name"lookup_key:"lookup_key"`
	Paths     *[]string              `json:"paths"lookup_key:"paths"`
	Path      *string                `json:"path"lookup_key:"path"`
	Uris      *[]string              `json:"uris"lookup_key:"uris"`
	Options   map[string]interface{} `json:"options"lookup_key:"options"`
}

func (c *HierarchyYamlFile) getConf(configFile string) {

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Println("Unmarshal: %v", err)
	}
}

//type Host struct {
//	Certname string          `json:"certname"`
//	Keys     []HieraKeyEntry `json:"keys"`
//}
//
//type HieraKeyEntry struct {
//	Key  string    `json:"key"`
//	Date time.Time `json:"date"`
//}

// HieraHostDBLogEntry is a single entry the hiera-log makes. So it is just a lookup for a key for a certname
type HieraHostDBLogEntry struct {
	Certname string  `json:"certname"`
	Key      string  `json:"key"`
	Date     *string `json:"date_string"`
}

// HieraHostDBEntry is just a collection ok key entries that have been looked up for a specific host.
type HieraHostDBEntry struct {
	ID      string                `bson:"_id"json:"id"`
	Entries []HieraHostDBLogEntry `json:"keys"`
}

// HieraKey Is an object that holds yaml data for a specific key
type HieraKey struct {
	Key      string          `json:"key"yaml:"key"`
	SubKeys  []string        `json:"sub_keys"yaml:"sub_keys"`
	InLookup bool            `json:"in_lookup"yaml:"in_lookup"`
	Values   []HieraKeyEntry `json:"in_lookup"yaml:"values"`
}

// HieraKeyENtry holds data for a single subkey of HieraKey and which data was in it and where it was found
type HieraKeyEntry struct {
	Path  string      `json:"path"yaml:"path"`
	Key   string      `json:"key"yaml:"key"`
	Type  string      `json:"type"yaml:"type"`
	Value interface{} `json:"value"yaml:"value"`
}

type HieraKeyMatch struct {
	Key       string               `json:"key"yaml:"key"`
	Locations []string             `json:"locations"yaml:"locations"`
	Matches   []HieraKeyMatchEntry `json:"matches"yaml:"matches"`
}

type HieraKeyMatchEntry struct {
	Path1 string `json:"path1"yaml:"path1"`
	Path2 string `json:"path2"yaml:"path2"`
	Key   string `json:"key"yaml:"key"`
}

// HierarchyResult is an object that is used to return data in json form trough the api. It holds the result for which hierarchy was found and which variables
type HierarchyResult struct {
	Paths     []string `json:"paths"yaml:"paths"`
	Variables []string `json:"vars"yaml:"vars"`
}

// YamlMapEntry contain the location of a file all the hiera data in a map and a flattened map of the same data
type YamlMapEntry struct {
	Path    string                 `json:"path"yaml:"path"`
	Content map[string]interface{} `json:"content"yaml:"content"`
	Flat    map[string]interface{} `json:"flat"yaml:"flat"`
}

// NewClient creates a database connection
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
