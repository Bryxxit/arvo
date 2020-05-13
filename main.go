package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/akira/go-puppetdb"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var (
	addr = flag.String("listen-address", ":8162", "The address to listen on for ")
	conf = flag.String("conf", "config/hiera-clean.yaml", "The path to the config file.")
)

var LAYOUT = "2006-01-02T15:04:05-0700"

//https://gist.github.com/cuixin/f10cea0f8639454acdfbc0c9cdced764

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

func main() {
	flag.Parse()
	var c Conf
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

	if c.DataDir == "" {
		c.DataDir = "/root/Desktop/HieraData"
	}
	if c.KeyTTLMinutes == 0 {
		c.KeyTTLMinutes = 1
	}

	if c.HieraFile == "" {
		c.HieraFile = "/etc/puppetlabs/puppet/hiera.yaml"
		c.HieraFile = "C:\\Users\\tieyz_admin\\Desktop\\Go\\arvo\\hiera.yaml"
	}

	if c.Puppet.Host == "" {
		c.Puppet.Host = "localhost"
	}

	if c.Puppet.Port <= 0 {
		c.Puppet.Port = 8080
	}

	//router := gin.Default()
	//
	//router.Use(cors.New(cors.Config{
	//	AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
	//	AllowHeaders:     []string{"*"},
	//	ExposeHeaders:    []string{"Content-Length"},
	//	AllowCredentials: true,
	//	AllowAllOrigins:  false,
	//	AllowOriginFunc:  func(origin string) bool { return true },
	//	MaxAge:           86400,
	//}))
	//// Simple group: v1
	//v1 := router.Group("/v1")
	//{
	//	v1.POST("/keys", PostKeyEndpoint(c))
	//	v1.GET("/keys", GetKeysForAllCertnamesEndpoint(c))
	//	v1.GET("/keys/:id", GetKeysForOneCertnamesEndpoint(c))
	//	v1.GET("/hierarchy", GetHierarchyEndPoint(c))
	//
	//}
	//
	//err := router.Run(*addr)
	//if err != nil {
	//	log.Fatal(err.Error())
	//}
	GetHierarchyForCertnameEndpoint(c, "lnx-a-rp01-17")

}

func GetHierarchyEndPoint(conf Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		defer c.Done()
		var hier LookupYaml
		hier.getConf(conf.HieraFile)
		paths_to_read := []string{}

		for _, h := range hier.Hierarchy {
			if h.Paths != nil {
				for _, p := range *h.Paths {
					//found, _ := regexp.MatchString("(.*)?([%][{].*[}]).*", p)
					//if found {
					//	facts := make(map[string]string)
					//	facts["mdi_region"] = "emea_be2"
					//	facts["mdi_platform"] = "puppet"
					//	facts["mdi_tier"] = "production"
					//	facts["clientcert"] = "by1acn.eu.seeds.basf.net"
					//	p = ReplaceVariableInPath(p, facts)
					//}
					paths_to_read = append(paths_to_read, conf.DataDir+"/"+p)
					log.Println(p)
				}
			}
			if h.Path != nil {
				paths_to_read = append(paths_to_read, conf.DataDir+"/"+*h.Path)
				log.Println(*h.Path)

			}
		}
		c.JSON(http.StatusOK, gin.H{})

	}
	return gin.HandlerFunc(fn)
}

func GetHierarchyForCertnameEndpoint(conf Conf, certname string) {
	var cl *puppetdb.Client
	if !conf.Puppet.SSL {
		cl = puppetdb.NewClient(conf.Puppet.Host, conf.Puppet.Port, false)

	} else {
		if conf.Puppet.Insecure {
			cl = puppetdb.NewClientSSLInsecure(conf.Puppet.Host, conf.Puppet.Port, false)

		} else {
			cl = puppetdb.NewClientSSL(conf.Puppet.Host, conf.Puppet.Port, conf.Puppet.Key, conf.Puppet.Cert, conf.Puppet.Ca, false)

		}
	}
	facts, _ := cl.NodeFacts(certname)
	//flattened := make(map[string]interface{})
	for _, fact := range facts {
		//fn := fact.Name
		//flat, err := flatten.Flatten(fact.Value, "", flatten.RailsStyle)
		switch (fact.Value.Data()).(type) {
		case map[string]interface{}:
			log.Println("hello")
		case interface{}:
			log.Println("hello there")
		default:
			log.Println("Unknown data type was parsed in facts of this host " + certname + " fact " + fact.Name)
		}

	}
}

func parseMap(aMap bson.M, factTotal *bson.M) {
	for key, val := range aMap {
		fn := key
		if strings.Contains(key, ".") {
			fn = strings.Replace(fn, ".", "_", -1)
		}
		switch concreteVal := val.(type) {
		case map[string]interface{}:
			var factClean bson.M = bson.M{}
			parseMap(val.(map[string]interface{}), &factClean)
			(*factTotal)[fn] = factClean
		case []interface{}:
			var factClean bson.M = bson.M{}
			// test if array of strings
			test := isStringArray(val.([]interface{}))
			if test {
				(*factTotal)[fn] = concreteVal
			} else {
				parseArray(val.([]interface{}), &factClean, fn)
				(*factTotal)[fn] = factClean

			}
		default:
			//fmt.Println(key, ":", concreteVal)
			(*factTotal)[fn] = concreteVal
		}
	}
}

func isStringArray(anArray []interface{}) bool {
	check := true

	for _, val := range anArray {
		t := reflect.TypeOf(val).String()
		if t != "string" && t != "int" && t != "float64" {
			check = false
		}
	}

	return check
}

func parseArray(anArray []interface{}, factTotal *bson.M, factName string) {
	var toSave []interface{}
	for _, val := range anArray {
		switch concreteVal := val.(type) {
		case map[string]interface{}:
			parseMap(val.(map[string]interface{}), factTotal)
		case []interface{}:
			parseArray(val.([]interface{}), factTotal, factName)
		default:
			//fmt.Println("Index", i, ":", concreteVal)
			toSave = append(toSave, concreteVal)

		}
	}
	(*factTotal)[factName] = toSave

}

func GetKeysForOneCertnamesEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		defer c.Done()

		s, err := GetOneCertnameLogEntry(d.DB, u1.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, *s)
		}
	}
	return gin.HandlerFunc(fn)
}

func GetKeysForAllCertnamesEndpoint(conf Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		hosts, err := GetAllCertnameLogEntry(conf.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, hosts)
		}

	}
	return gin.HandlerFunc(fn)
}

func PostKeyEndpoint(conf Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u HieraLogEntry
		err := c.BindJSON(&u)
		defer c.Done()

		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"inserted": false, "message": err.Error()})
		} else {
			currentTime := time.Now()
			str := currentTime.Format(LAYOUT)
			e := HieraLogHost{
				ID: u.Certname,
				Entries: []HieraLogEntry{
					{
						Certname: u.Certname,
						Key:      u.Key,
						Date:     &str,
					},
				},
			}
			res, err := InsertLogEntryWrapper(e, conf)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			} else {
				c.JSON(http.StatusCreated, gin.H{"success": true, "message": *res})
			}

		}
	}
	return gin.HandlerFunc(fn)
}

func GetAllCertnameLogEntry(d Database) ([]HieraLogHost, error) {
	dbConn, err := NewClient(d.Host, d.Port, d.Database)
	arr := []HieraLogHost{}
	if err != nil {
		return arr, err
	}
	collection := dbConn.Database(d.Database).Collection("logging")
	findOptions := options.Find()
	findOptions.SetLimit(1)
	filter := bson.D{}
	// Finding multiple documents returns a cursor
	cur, err := collection.Find(context.TODO(), filter, findOptions)
	for cur.Next(context.TODO()) {
		var elem HieraLogHost

		err := cur.Decode(&elem)
		if err != nil {
			log.Println(err.Error())
		} else {
			arr = append(arr, elem)
		}
	}
	defer dbConn.Disconnect(context.TODO())
	return arr, nil
}

func GetOneCertnameLogEntry(d Database, certname string) (*HieraLogHost, error) {
	dbConn, err := NewClient(d.Host, d.Port, d.Database)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection("logging")
	findOptions := options.Find()
	findOptions.SetLimit(1)
	filter := bson.D{{"_id", certname}}
	var result *HieraLogHost
	// Finding multiple documents returns a cursor
	cur, err := collection.Find(context.TODO(), filter, findOptions)
	for cur.Next(context.TODO()) {
		var elem HieraLogHost
		err := cur.Decode(&elem)
		if err != nil {
			log.Println(err.Error())
		} else {
			result = &elem
		}
	}
	//err = curr.Decode(result)
	if err != nil {
		return nil, err
	}
	defer dbConn.Disconnect(context.TODO())
	if result != nil {
		return result, nil
	}
	return nil, errors.New("Entry not found")
}

func InsertLogEntry(e HieraLogHost, d Database) (*string, error) {

	dbConn, err := NewClient(d.Host, d.Port, d.Database)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection("logging")
	insertResult, err := collection.InsertOne(context.TODO(), e)
	if err != nil {
		return nil, err
	}
	defer dbConn.Disconnect(context.TODO())
	str := fmt.Sprintf("Inserted one entry id: %s", insertResult.InsertedID)
	return &str, nil
}

func InsertLogEntryWrapper(e HieraLogHost, d Conf) (*string, error) {
	// first see if entry exists
	e2, err := GetOneCertnameLogEntry(d.DB, e.ID)
	// if noet we can Insert
	if e2 == nil || err != nil {
		res, err := InsertLogEntry(e, d.DB)
		return res, err
	} else {
		if e2 != nil {
			cleaned := RemoveOldLogEntries(*e2, d, e.Entries[0])
			res, err := UpdateLogEntryDB(cleaned, d.DB)
			if err != nil {
				log.Println(err.Error())
			}
			return res, err
		}

	}
	return nil, errors.New("Something went wrong with code logic")
}

func UpdateLogEntryDB(e HieraLogHost, d Database) (*string, error) {
	ks, _ := GetOneCertnameLogEntry(d, e.ID)
	if ks == nil {
		return nil, errors.New("Entry not found")
	}

	dbConn, err := NewClient(d.Host, d.Port, d.Database)
	if err != nil {
		return nil, errors.New("Database connection failed")
	}

	collection := dbConn.Database(d.Database).Collection("logging")
	var result *mongo.UpdateResult
	selector := bson.M{"_id": e.ID}
	//updator :=  r.ToBSON()
	result, err = collection.ReplaceOne(context.TODO(), selector, e)

	if err != nil {
		return nil, err
	}
	defer dbConn.Disconnect(context.TODO())

	str := fmt.Sprintf("Updated number of entries: %d", result.MatchedCount)
	return &str, nil
}

func RemoveOldLogEntries(e HieraLogHost, conf Conf, entry HieraLogEntry) HieraLogHost {
	entries := []HieraLogEntry{}
	for _, k := range e.Entries {

		//str := currentTime.Format(LAYOUT)
		if k.Date != nil && k.Key != entry.Key {

			diff := GetDiffBetweenDates(k.Date)
			if diff < conf.KeyTTLMinutes {
				entries = append(entries, k)
			}

		}
	}
	entries = append(entries, entry)
	e.Entries = entries
	return e
}

func GetDiffBetweenDates(date2 *string) int {
	currentTime := time.Now()
	currentTime.Format(LAYOUT)

	if date2 != nil {

		t, err := time.Parse(LAYOUT, *date2)
		if err == nil {
			//t := FixLocationIssue(t)
			diff := currentTime.Sub(t)
			return int(math.Floor(diff.Minutes()))
		}

	}
	return 0

}

func temp() {
	datadir := "/root/Desktop/hieradata"
	var c LookupYaml
	c.getConf("/root/Desktop/go-scripts/hiera-clean/example-lookup.yaml")
	paths_to_read := []string{}

	for _, h := range c.Hierarchy {
		if h.Paths != nil {
			for _, p := range *h.Paths {
				found, _ := regexp.MatchString("(.*)?([%][{].*[}]).*", p)
				if found {
					facts := make(map[string]string)
					facts["mdi_region"] = "emea_be2"
					facts["mdi_platform"] = "puppet"
					facts["mdi_tier"] = "production"
					facts["clientcert"] = "by1acn.eu.seeds.basf.net"
					p = ReplaceVariableInPath(p, facts)
				}
				paths_to_read = append(paths_to_read, datadir+"/"+p)
			}
		}
	}
	// we have all paths
	values := getAllValuesInYaml(paths_to_read)
	SetInLookup(&values)
	PrintKeysNotLookedUp(values)
	fmt.Println()
	printDuplicateEntries(values)
}

func printDuplicateEntries(values []HieraKeyFullEntry) {
	matches := LookForDuplicateData(values)

	for _, m := range matches {
		fmt.Println(m.Key)
		fmt.Println("locations:")
		for _, l := range m.Locations {
			println("  - " + l)
		}
		fmt.Println("matches:")
		for _, l := range m.Matches {
			str := fmt.Sprintf("%s: %s - %s", l.Key, l.Path1, l.Path2)
			println("  - " + str)
		}
	}
}

func LookForDuplicateData(values []HieraKeyFullEntry) []HieraMatch {
	matchArr := []HieraMatch{}
	arr := []string{}
	for index1, val1 := range values {
		matches := []HieraKeyFullEntry{val1}
		for index2, val2 := range values {
			if val1.Key == val2.Key && index1 != index2 {
				matches = append(matches, val2)
			}
		}

		if len(matches) > 1 {
			if !stringInSlice(val1.Key, arr) {
				a := HieraMatch{
					Key:       val1.Key,
					Locations: []string{},
					Matches:   []HieraMatchEntry{},
				}
				for index3, v := range matches {
					for _, r := range v.Values {
						if !stringInSlice(r.Path, a.Locations) {
							a.Locations = append(a.Locations, r.Path)
						}
					}
					if index3 < len(matches)-1 {
						entries := CompareTwoHieraEntries(matches[index3], matches[index3+1])
						a.Matches = append(a.Matches, entries...)
					}
				}
				matchArr = append(matchArr, a)
				arr = append(arr, val1.Key)
			}

		}
	}
	return matchArr
}

func CompareTwoHieraEntries(var1 HieraKeyFullEntry, var2 HieraKeyFullEntry) []HieraMatchEntry {
	arr := []HieraMatchEntry{}
	for _, value1 := range var1.Values {
		for _, value2 := range var2.Values {
			if value1.Key == value2.Key && value2.Value == value1.Value {
				a := HieraMatchEntry{
					Path1: value1.Path,
					Path2: value2.Path,
					Key:   value1.Key,
				}
				arr = append(arr, a)
			}
		}
	}
	return arr
}

func PrintKeysNotLookedUp(values []HieraKeyFullEntry) {
	arr := []string{}
	for _, key := range values {
		if !key.InLookup && !stringInSlice(key.Key, arr) {
			arr = append(arr, key.Key)
			fmt.Println(key.Key)
		}
	}

}

func SetInLookup(values *[]HieraKeyFullEntry) {
	keys := GetAllKeys()

	for _, e := range keys {
		for index, e2 := range *values {
			if stringInSlice(e.Key, e2.SubKeys) {
				(*values)[index].InLookup = true
			}
		}
	}
}

func getAllValuesInYaml(paths []string) []HieraKeyFullEntry {
	values := []HieraKeyFullEntry{}
	for _, p := range paths {
		val := getHieraKeyValueEntriesForPath(p)
		values = append(values, val...)
	}

	return values
}

/////// HIERARCHY CODE

func getFactNameFromHieraVar(variable string) string {
	str := strings.ReplaceAll(variable, "%{", "")
	str = strings.TrimSuffix(str, "}")
	str = strings.TrimPrefix(str, "::")
	return str
}

func hieraVarToFact(facts map[string]string, variable string) *string {
	if val, ok := facts[variable]; ok {
		return &val
	}
	return nil
}

// getFactsFromPath gets the fact names from the hiera path
func getFactsFromPath(path string) []string {
	counter := 0
	str := path
	facts := []string{}
	for counter < 20 {
		found, _ := regexp.MatchString("(.*)?([%][{].*[}]).*", str)
		if found {
			regex := *regexp.MustCompile(`(.*)?([%][{].*[}]).*`)
			hostGroupMatch := regex.FindStringSubmatch(str)
			fact := getFactNameFromHieraVar(hostGroupMatch[2])
			facts = append(facts, fact)
			str = strings.ReplaceAll(str, hostGroupMatch[2], fact)

		} else {
			counter = 50
		}
		counter = counter + 1
	}
	return facts
}

// supply a string map of facts and it will find variables in path and replace them. This also searches the path for variables with a regex
func ReplaceVariableInPath(path string, facts map[string]string) string {
	str := path
	found, _ := regexp.MatchString("(.*)?([%][{].*[}]).*", str)
	if found {
		regex := *regexp.MustCompile(`(.*)?([%][{].*[}]).*`)
		hostGroupMatch := regex.FindStringSubmatch(str)
		fact := getFactNameFromHieraVar(hostGroupMatch[2])
		factVal := hieraVarToFact(facts, fact)
		if factVal != nil {
			str = strings.ReplaceAll(str, hostGroupMatch[2], *factVal)
		} else {
			str = strings.ReplaceAll(str, hostGroupMatch[2], "fact_not_found")
		}
		found, _ = regexp.MatchString("(.*)?([%][{].*[}]).*", str)
		if found {
			return ReplaceVariableInPath(str, facts)
		}
	}

	return str
}

//////// GET KEYS FROM HIERA-LOG
func GetAllKeys() []HieraKeyEntry {
	content, err := ioutil.ReadFile("/root/Desktop/go-scripts/hiera-clean/example-log")
	if err != nil {
		log.Fatal(err)
	}
	// Convert []byte to string and print to screen
	text := string(content)
	keys := []HieraKeyEntry{}
	lines := strings.Split(text, "\n")
	for _, l := range lines {
		k := lineToEntry(l)
		if k != nil {
			keys = append(keys, *k)
		}
	}
	//for _, k := range keys {
	//	println(k.Key)
	//
	//}
	return keys
}

func lineToEntry(line string) *HieraKeyEntry {
	colls1 := strings.Split(line, " ")
	if len(colls1) == 12 {
		e := HieraKeyEntry{
			Key:  colls1[11],
			Date: time.Now(),
		}
		return &e
	}

	return nil
}

/////// YAML TO GOLANG OBJECT
func getHieraKeyValueEntriesForPath(p string) []HieraKeyFullEntry {
	values := []HieraKeyFullEntry{}
	if DoesFileExist(p) {
		c := yamlToMap(p)
		values = getKeys(c, p)
	}

	return values

}

func yamlToMap(path string) *bson.M {
	c2 := bson.M{}
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c2)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return &c2
}

func getKeys(m *bson.M, path string) []HieraKeyFullEntry {
	keys := []HieraKeyFullEntry{}
	for k, v := range *m {
		key := HieraKeyFullEntry{
			Key:      k,
			SubKeys:  []string{},
			InLookup: false,
			Values:   []HieraKeyFullValueEntry{},
		}
		process(v, &k, &key, path)
		keys = append(keys, key)
	}
	return keys
}

//https://stackoverflow.com/questions/26975880/convert-mapinterface-interface-to-mapstringstring
func process(in interface{}, parentKey *string, key *HieraKeyFullEntry, path string) {
	//switch v := in.(type)
	if parentKey != nil {
		if !stringInSlice(*parentKey, key.SubKeys) {
			key.SubKeys = append(key.SubKeys, *parentKey)
		}
	}
	switch in.(type) {
	case map[interface{}]interface{}:
		m := MapToMap(in.(map[interface{}]interface{}))
		process(m, parentKey, key, path)
	case map[string]interface{}:
		for k, v := range in.(map[string]interface{}) {
			pk := k
			if parentKey != nil {
				pk = (*parentKey) + "::" + k
			}
			process(v, &pk, key, path)
		}
	case string:
		str := fmt.Sprintf("%s", *parentKey)
		e := HieraKeyFullValueEntry{
			Path:  path,
			Key:   str,
			Type:  "string",
			Value: in.(string),
		}
		key.Key = str
		key.Values = append(key.Values, e)
		//(*keys)[str] = in.(string)
		//println(fmt.Sprintf("%s: '%s'", *parentKey, in.(string)))
	case int:
		str := fmt.Sprintf("%s", *parentKey)
		//(*keys)[str] = in.(int)
		e := HieraKeyFullValueEntry{
			Path:  path,
			Key:   str,
			Type:  "int",
			Value: in.(int),
		}
		key.Key = str
		key.Values = append(key.Values, e)
		//println(fmt.Sprintf("%s: %d", *parentKey, in.(int)))
	case float64:
		str := fmt.Sprintf("%s", *parentKey)
		//(*keys)[str] = in.(float64)
		e := HieraKeyFullValueEntry{
			Path:  path,
			Key:   str,
			Type:  "float64",
			Value: in.(float64),
		}
		key.Key = str
		key.Values = append(key.Values, e)
		//println(fmt.Sprintf("%s: %d", *parentKey, in.(float64)))
	case bool:
		str := fmt.Sprintf("%s", *parentKey)
		//(*keys)[str] = in.(bool)
		e := HieraKeyFullValueEntry{
			Path:  path,
			Key:   str,
			Type:  "bool",
			Value: in.(bool),
		}
		key.Key = str
		key.Values = append(key.Values, e)
		//println(fmt.Sprintf("%s: %d", *parentKey, in.(float64)))
	//case []interface{}:
	//	str := fmt.Sprintf("%s", *parentKey)
	//
	//	// todo also validate types
	//	arr := []string{}
	//	for _, i := range in.([]interface{}){
	//		arr = append(arr,  i.(string))
	//	}
	//	(*keys)[str] = arr
	//	//println(fmt.Sprintf("%s: %d", *parentKey, in.(float64)))
	default:
		str := fmt.Sprintf("%s", *parentKey)
		e := HieraKeyFullValueEntry{
			Path:  path,
			Key:   str,
			Type:  "other",
			Value: in,
		}
		key.Key = str
		key.Values = append(key.Values, e)
		//(*keys)[str] = in
		//println(fmt.Sprintf("%s:", *parentKey))
	}
}

///////////// helper functions

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func MapToMap(mapInterface map[interface{}]interface{}) map[string]interface{} {
	mapString := make(map[string]interface{})
	for key, value := range mapInterface {
		strKey := fmt.Sprintf("%v", key)
		mapString[strKey] = value
	}
	return mapString
}

func DoesFileExist(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
