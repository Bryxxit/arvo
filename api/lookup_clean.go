package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jeremywohl/flatten"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type InLogAndHieraEntry struct {
	Key   string   `json:"key"yaml:"key"`
	Paths []string `json:"paths"yaml:"paths"`
}

// GetKeyLocationsForCertnameEndpoint example
// @Summary Get the clean result for a certname
// @Description Looks trough you logged entries and hierarchy files to find unused keys etc. That will help you clean up hiera data.
// @Param  id     path   string     true  "Some ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} YamlCleanResult ""
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /clean/{id} [get]
func GetKeyLocationsForCertnameEndpoint(conf Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		defer c.Done()
		//s, err := GetOneCertnameLogEntry(conf.DB, u1.ID)
		hierarchy, err := GetHierarchyForCertname(conf, u1.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			entries := []YamlMapEntry{}

			for _, p := range hierarchy.Paths {
				e := GetYamlMapEntryFromPath(p)
				entries = append(entries, e)
			}

			loggedKeys, err := GetOneCertnameLogEntry(conf.DB, u1.ID)
			res := YamlCleanResult{
				InLogNotInHiera: []string{},
				InLogAndHiera:   []InLogAndHieraEntry{},
				InHieraNotInLog: []InLogAndHieraEntry{},
				DuplicateData:   []InLogAndHieraEntry{},
			}
			if err == nil {
				// first get keys in log but not in hiera and in log and in hiera
				for _, e1 := range loggedKeys.Entries {
					check0 := false
					for _, e2 := range entries {
						if IsKeyInMap(e1.Key, e2.Content) || IsKeyInMap(e1.Key, e2.Flat) {
							check0 = true
							check1 := false
							for index, e3 := range res.InLogAndHiera {
								if e3.Key == e1.Key {
									res.InLogAndHiera[index].Paths = append(res.InLogAndHiera[index].Paths, e2.Path)
									check1 = true
								}
							}
							if !check1 {
								res.InLogAndHiera = append(res.InLogAndHiera, InLogAndHieraEntry{
									Key:   e1.Key,
									Paths: []string{e2.Path},
								})
							}
						}
					}
					if !check0 {
						res.InLogNotInHiera = append(res.InLogNotInHiera, e1.Key)
					}
				}
			}
			// now do the reverse
			for _, e1 := range entries {
				for key, _ := range e1.Content {
					if !keyInLog(key, loggedKeys.Entries) {
						check1 := false
						for index, e3 := range res.InHieraNotInLog {
							if e3.Key == key {
								res.InHieraNotInLog[index].Paths = append(res.InHieraNotInLog[index].Paths, e1.Path)
								check1 = true
							}
						}
						if !check1 {
							res.InHieraNotInLog = append(res.InHieraNotInLog, InLogAndHieraEntry{
								Key:   key,
								Paths: []string{e1.Path},
							})
						}
					}
				}
			}

			// lastly search for duplicates
			for _, e1 := range entries {
				for key1, val1 := range e1.Content {
					for _, e2 := range entries {
						if e1.Path != e2.Path {
							for key2, val2 := range e2.Content {
								switch val1.(type) {
								case map[string]interface{}:
									log.Println("skipping hashes for now")
								default:
									if key1 == key2 && val1 == val2 {
										check1 := false
										for index, e3 := range res.DuplicateData {
											if e3.Key == key1 {
												if !stringInSlice(e1.Path, res.DuplicateData[index].Paths) {
													res.DuplicateData[index].Paths = append(res.DuplicateData[index].Paths, e1.Path)
												}
												check1 = true
											}
										}
										if !check1 {
											res.DuplicateData = append(res.DuplicateData, InLogAndHieraEntry{
												Key:   key1,
												Paths: []string{e1.Path},
											})
										}
									}
								}
							}

						}
					}
				}
			}

			c.JSON(http.StatusOK, res)

		}
	}
	return gin.HandlerFunc(fn)
}

// CleanAllEndpoint example
// @Summary Returns the clean all result if it has been generated
// @Description After the resresh function has been done. You can call this method for the result.
// @Accept  json
// @Produce  json
// @Success 200 {object} CleanAllResult "The clean all result."
// @Failure 404 {object} APIMessage "No entry was found run the /v1/clean-all/refresh endpoint first"
// @Failure 500 {object} APIMessage "Something went wrong getting the entries"
// @Router /clean-all [get]
func CleanAllEndpoint(conf Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		res, err := GetFullCleanResultEntry(conf.DB)

		if res == nil {
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			} else {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "No entry was found run the /v1/clean-all/refresh endpoint first"})

			}
		} else {
			c.JSON(http.StatusOK, res)

		}

	}

	return fn
}

// CleanAllRefreshEndpoint example
// @Summary Starts generating an entry for the clean all result.
// @Description As parsing your whole environment may take a while this job starts doing the process in the background. You will get a json that that says the process has started
// @Accept  json
// @Produce  json
// @Success 200 {object} APIMessage "Gathering result may take a while check the clean endpoint for the result."
// @Failure 500 {object} APIMessage "Something went wrong getting the entries"
// @Router /clean-all/refresh [get]
func CleanAllRefreshEndpoint(conf Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		go func() {
			CleanAll(conf)
		}()
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Gathering result may take a while check the clean endpoint for the result."})

	}

	return fn
}

func keyInLog(a string, list []HieraHostDBLogEntry) bool {
	for _, b := range list {
		if b.Key == a {
			return true
		}
	}
	return false
}

func IsKeyInMap(key string, mapy map[string]interface{}) bool {
	for k, _ := range mapy {
		if k == key {
			return true
		}
	}
	return false
}

func FlattenYamlMap(yaml map[string]interface{}) map[string]interface{} {
	for key, val := range yaml {
		yaml[key] = Process(val)
	}
	return yaml
}

func Process(in interface{}) interface{} {
	switch in.(type) {
	case map[interface{}]interface{}:
		m := MapToMap(in.(map[interface{}]interface{}))
		for k, val := range m {
			m[k] = Process(val)

		}
		return m
	default:
		return in
	}
	return in
}

func GetYamlMapEntryFromPath(path string) YamlMapEntry {
	entry := YamlMapEntry{
		Path:    path,
		Content: make(map[string]interface{}),
		Flat:    make(map[string]interface{}),
	}
	mapy := YamlFileToStringMap(path)
	entry.Content = mapy
	f := FlattenYamlMap(mapy)
	flat, err := flatten.Flatten(f, "", flatten.DotStyle)
	if err != nil {
		log.Println(err.Error())
	} else {
		entry.Flat = flat

	}

	return entry
}

// https://stackoverflow.com/questions/40737122/convert-yaml-to-json-without-struct // ALso can be converted to json
func YamlFileToStringMap(path string) map[string]interface{} {
	mapy := make(map[string]interface{})
	if DoesFileExist(path) {
		content := ReadFile(path)
		var yamlFileKeys map[string]interface{}
		err := yaml.Unmarshal([]byte(content), &yamlFileKeys)
		if err == nil {
			return yamlFileKeys
		} else {
			log.Println(err.Error())
		}
	}
	return mapy
}

func ReadFile(path string) []byte {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	return content
}

func ReadAllFilesYaml(conf Conf) []string {
	yamlFiles := []string{}
	err := filepath.Walk(conf.DataDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
				p := path
				if runtime.GOOS == "windows" {
					p = strings.ReplaceAll(p, "\\", "/")
				}
				yamlFiles = append(yamlFiles, p)
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return yamlFiles
}

func CleanAll(conf Conf) {
	paths := ReadAllFilesYaml(conf)
	paths_matches := []string{}
	allLoggedHieraKeys := []string{}
	entries := []YamlMapEntry{}
	result := CleanAllResult{
		ID:             "full",
		PathsNeverUsed: []string{},
		KeysNeverUsed:  []YamlKeyPath{},
	}
	certnameLogEntries, _ := GetAllCertnameLogEntry(conf.DB)
	for _, k := range certnameLogEntries {
		for _, key := range k.Entries {
			if !stringInSlice(key.Key, allLoggedHieraKeys) {
				allLoggedHieraKeys = append(allLoggedHieraKeys, key.Key)
			}
		}
		hierarchy, err := GetHierarchyForCertname(conf, k.ID)
		if err != nil {
			log.Println(err.Error())
		} else {
			for _, p2 := range hierarchy.Paths {
				if stringInSlice(p2, paths) {
					if !stringInSlice(p2, paths_matches) {
						paths_matches = append(paths_matches, p2)
					}
				}
				e := GetYamlMapEntryFromPath(p2)
				entries = append(entries, e)
			}
		}
	}

	// these are the files that are in the directory but were never found in the log.
	for _, p := range paths {
		if !stringInSlice(p, paths_matches) {
			result.PathsNeverUsed = append(result.PathsNeverUsed, p)
		}
	}

	// this part gets all the keys that never appeared in any log
	allHieraKeysNotInAnyLogs := []YamlKeyPath{}
	for _, entry := range entries {
		for key, _ := range entry.Content {
			if !stringInSlice(key, allLoggedHieraKeys) {
				check := false
				for i, e := range allHieraKeysNotInAnyLogs {
					if e.Key == key {
						check = true
						if !stringInSlice(entry.Path, e.Paths) {
							allHieraKeysNotInAnyLogs[i].Paths = append(allHieraKeysNotInAnyLogs[i].Paths, entry.Path)
						}
					}
				}
				if !check {
					allHieraKeysNotInAnyLogs = append(allHieraKeysNotInAnyLogs, YamlKeyPath{
						Paths: []string{entry.Path},
						Key:   key,
					})
				}
			}
		}
	}
	// this is the result of the keys that are in some yaml file but never appeared in any log.
	for _, e := range allHieraKeysNotInAnyLogs {
		result.KeysNeverUsed = append(result.KeysNeverUsed, e)
	}
	InsertFullCleanResultWrapper(result, conf)

}

func InsertFullCleanResultWrapper(e CleanAllResult, d Conf) (*string, error) {
	// first see if entry exists
	e2, err := GetFullCleanResultEntry(d.DB)
	// if noet we can Insert
	if e2 == nil || err != nil {
		res, err := InsertFullCleanResult(e, d.DB)
		return res, err
	} else {
		if e2 != nil {
			res, err := UpdateFullCleanResult(e, d.DB)
			if err != nil {
				log.Println(err.Error())
			}
			return res, err
		}

	}
	return nil, errors.New("Something went wrong with code logic")
}

func GetFullCleanResultEntry(d Database) (*CleanAllResult, error) {
	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection("fullclean")
	findOptions := options.Find()
	findOptions.SetLimit(1)
	filter := bson.D{{"_id", "full"}}
	var result *CleanAllResult
	// Finding multiple documents returns a cursor
	cur, err := collection.Find(context.TODO(), filter, findOptions)
	for cur.Next(context.TODO()) {
		var elem CleanAllResult
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

func InsertFullCleanResult(e CleanAllResult, d Database) (*string, error) {

	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection("fullclean")
	insertResult, err := collection.InsertOne(context.TODO(), e)
	if err != nil {
		return nil, err
	}
	defer dbConn.Disconnect(context.TODO())
	str := fmt.Sprintf("Inserted one entry id: %s", insertResult.InsertedID)
	return &str, nil
}

func UpdateFullCleanResult(e CleanAllResult, d Database) (*string, error) {
	ks, _ := GetOneCertnameLogEntry(d, e.ID)
	if ks == nil {
		return nil, errors.New("Entry not found")
	}

	dbConn, err := NewClient(d)
	if err != nil {
		return nil, errors.New("Database connection failed")
	}

	collection := dbConn.Database(d.Database).Collection("fullclean")
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
