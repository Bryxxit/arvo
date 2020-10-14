package api

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
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

func GetOneStringMapEntryFromCollection(d Database, id string, colName string) (*map[string]interface{}, error) {
	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection(colName)
	findOptions := options.Find()
	findOptions.SetLimit(1)
	filter := bson.D{{"_id", id}}
	var result *map[string]interface{}
	// Finding multiple documents returns a cursor
	cur, err := collection.Find(context.TODO(), filter, findOptions)
	for cur.Next(context.TODO()) {
		var elem map[string]interface{}
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

func GetAllStringMapEntriesFromDB(d Database, collName string) ([]*map[string]interface{}, error) {
	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection(collName)
	findOptions := options.Find()
	filter := bson.D{}
	var result []*map[string]interface{}
	// Finding multiple documents returns a cursor
	cur, err := collection.Find(context.TODO(), filter, findOptions)
	for cur.Next(context.TODO()) {
		var elem map[string]interface{}
		err := cur.Decode(&elem)
		if err != nil {
			log.Println(err.Error())
		} else {
			result = append(result, &elem)

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

func UpdateStringMapEntry(id string, e map[string]interface{}, d Database, colName string) (*string, error) {

	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}

	collection := dbConn.Database(d.Database).Collection(colName)
	var result *mongo.UpdateResult
	selector := bson.M{"_id": id}
	//updator :=  r.ToBSON()
	result, err = collection.ReplaceOne(context.TODO(), selector, e)

	if err != nil {
		return nil, err
	}
	defer dbConn.Disconnect(context.TODO())

	str := fmt.Sprintf("Updated number of entries: %d", result.MatchedCount)
	return &str, nil
}

func DeleteOneStringMapEntry(d Database, id string, colName string) (*string, error) {
	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}

	collection := dbConn.Database(d.Database).Collection(colName)

	var result *mongo.DeleteResult
	selector := bson.M{"_id": id}
	result, err = collection.DeleteOne(context.TODO(), selector)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer dbConn.Disconnect(context.TODO())
	if result != nil {
		str := fmt.Sprintf("Deleted number of entries: %d", result.DeletedCount)
		return &str, nil
	}
	return nil, errors.New("Entry not found")

}

func InsertStringMapEntry(id string, e map[string]interface{}, d Database, colName string) (*string, error) {

	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}

	// also need to retrieve the thing first and do it that way.
	// we should also keep not of the old entries maybe

	e["_id"] = id
	collection := dbConn.Database(d.Database).Collection(colName)
	insertResult, err := collection.InsertOne(context.TODO(), e)
	if err != nil {
		return nil, err
	}
	defer dbConn.Disconnect(context.TODO())
	str := fmt.Sprintf("Inserted one entry id: %s", insertResult.InsertedID)
	return &str, nil
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
