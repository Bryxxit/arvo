package api

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

type APIMessage struct {
	Success bool
	Message string
}

type APIArrayMessage struct {
	Success bool
	Message []string
}

type HieraDataExample struct {
	Key  string                 `json:"key"yaml:"key"`
	Key2 map[string]interface{} `json:"key2"yaml:"key2"`
	Key3 bool                   `json:"key3"yaml:"key3"`
	Key4 int                    `json:"key4"yaml:"key4"`
}

// Conf is the main configuration file for arvo and holds the settings needed to run it
type Conf struct {
	DB             Database       `yaml:"db"`
	KeyTTLMinutes  int            `yaml:"key_ttl_minutes"`
	DataDir        string         `yaml:"datadir"`
	Puppet         PuppetDBConfig `yaml:"puppet"`
	HieraFile      string         `yaml:"hiera_file"`
	Hierarchy      []string       `yaml:"hierarchy"`
	Dummy          bool           `yaml:"dummy"`
	UseInflux      bool           `yaml:"use_influx"`
	Url            string         `yaml:"url"`
	Bucket         string         `yaml:"bucket"`
	InfluxInterval int            `yaml:"influx_interval"`
}

// Database holds the database settings to run arvo
type Database struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Database     string `yaml:"db"`
	AuthDatabase string `yaml:"auth_db"`
	Type         string `yaml:"type"`
	connection   mongo.Client
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

type HIERAKEYID struct {
	ID       string `uri:"id" binding:"required"`
	Certname string `uri:"certname" binding:"required"`
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
func NewClient(db Database) (*mongo.Client, error) {
	// create the connection uri
	uri := fmt.Sprintf(`mongodb://%s:%d`,
		db.Host,
		db.Port,
	)
	clientOptions := options.Client()
	clientOptions.ApplyURI(uri)
	if db.Username != "" && db.Password != "" {
		clientOptions.SetAuth(options.Credential{
			AuthSource: db.AuthDatabase, Username: db.Username, Password: db.Password,
		})
	}
	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
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

type YamlCleanResult struct {
	InLogNotInHiera []string             `json:"in_log_not_in_hiera"yaml:"in_log_not_in_hiera"`
	InLogAndHiera   []InLogAndHieraEntry `json:"in_log_and_hiera"yaml:"in_log_and_hiera"`
	InHieraNotInLog []InLogAndHieraEntry `json:"in_hiera_not_in_log"yaml:"in_hiera_not_in_log"`
	DuplicateData   []InLogAndHieraEntry `json:"duplicates"yaml:"duplicates"`
}

type CleanAllResult struct {
	ID             string        `bson:"_id"json:"id"`
	PathsNeverUsed []string      `json:"paths_never_used"yaml:"paths_never_used"`
	KeysNeverUsed  []YamlKeyPath `json:"keys_never_used"yaml:"keys_never_used"`
}

type YamlKeyPath struct {
	Paths []string `json:"paths"yaml:"paths"`
	Key   string   `json:"key"yaml:"key"`
}
