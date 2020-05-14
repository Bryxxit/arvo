package cmd

import (
	"github.com/gin-gonic/gin"
	"github.com/jeremywohl/flatten"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
)

type InLogAndHieraEntry struct {
	Key   string   `json:"key"yaml:"key"`
	Paths []string `json:"paths"yaml:"paths"`
}

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
