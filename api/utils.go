package api

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"math"
	"os"
	"reflect"
	"strings"
	"time"
)

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
