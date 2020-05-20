package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"time"
)

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
		var u HieraHostDBLogEntry
		err := c.BindJSON(&u)
		defer c.Done()

		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"inserted": false, "message": err.Error()})
		} else {
			currentTime := time.Now()
			str := currentTime.Format(LAYOUT)
			e := HieraHostDBEntry{
				ID: u.Certname,
				Entries: []HieraHostDBLogEntry{
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

func GetAllCertnameLogEntry(d Database) ([]HieraHostDBEntry, error) {
	dbConn, err := NewClient(d)
	arr := []HieraHostDBEntry{}
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
		var elem HieraHostDBEntry

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

func GetOneCertnameLogEntry(d Database, certname string) (*HieraHostDBEntry, error) {
	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection("logging")
	findOptions := options.Find()
	findOptions.SetLimit(1)
	filter := bson.D{{"_id", certname}}
	var result *HieraHostDBEntry
	// Finding multiple documents returns a cursor
	cur, err := collection.Find(context.TODO(), filter, findOptions)
	for cur.Next(context.TODO()) {
		var elem HieraHostDBEntry
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

func InsertLogEntry(e HieraHostDBEntry, d Database) (*string, error) {

	dbConn, err := NewClient(d)
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

func InsertLogEntryWrapper(e HieraHostDBEntry, d Conf) (*string, error) {
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

func UpdateLogEntryDB(e HieraHostDBEntry, d Database) (*string, error) {
	ks, _ := GetOneCertnameLogEntry(d, e.ID)
	if ks == nil {
		return nil, errors.New("Entry not found")
	}

	dbConn, err := NewClient(d)
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

func RemoveOldLogEntries(e HieraHostDBEntry, conf Conf, entry HieraHostDBLogEntry) HieraHostDBEntry {
	entries := []HieraHostDBLogEntry{}
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
