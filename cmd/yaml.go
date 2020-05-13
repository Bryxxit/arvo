package cmd

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

func getHieraKeyValueEntriesForPath(p string) []HieraKey {
	values := []HieraKey{}
	if DoesFileExist(p) {
		c := yamlToMap(p)
		values = getKeys(c, p)
	}

	return values

}

func getKeys(m *bson.M, path string) []HieraKey {
	keys := []HieraKey{}
	for k, v := range *m {
		key := HieraKey{
			Key:      k,
			SubKeys:  []string{},
			InLookup: false,
			Values:   []HieraKeyEntry{},
		}
		process(v, &k, &key, path)
		keys = append(keys, key)
	}
	return keys
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

//https://stackoverflow.com/questions/26975880/convert-mapinterface-interface-to-mapstringstring
func process(in interface{}, parentKey *string, key *HieraKey, path string) {
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
		e := HieraKeyEntry{
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
		e := HieraKeyEntry{
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
		e := HieraKeyEntry{
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
		e := HieraKeyEntry{
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
		e := HieraKeyEntry{
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
