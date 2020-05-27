package api

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
)

// HieraIdsEndpoint example
// @Summary Get all hiera path ids
// @Description Getas all the ids of your paths so you can see which hiera paths are available.
// @Accept  json
// @Produce  json
// @Success 200 {object} APIArrayMessage
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/path [get]
func HieraIdsEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		defer c.Done()

		s, err := GetAllHieraEntries(d.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			ids := []string{}
			for _, m := range s {
				if m != nil {
					if val, ok := (*m)["_id"]; ok {
						ids = append(ids, val.(string))
					}
				}
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "paths": ids})
		}
	}
	return gin.HandlerFunc(fn)
}

// HieraIdEndpoint example
// @Summary Get a hiera path
// @Description Get the data from one hiera path
// @Param  id     path   string     true  "Some ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/path/{id} [get]
func HieraIdEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		defer c.Done()

		s, err := GetOneHieraEntry(d.DB, u1.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, *s)
		}
	}
	return gin.HandlerFunc(fn)
}

// DeleteOneHieraEntry example
// @Summary Delete a hiera path
// @Description Deletes a hiera path entry
// @Param  id     path   string     true  "Some ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} APIMessage
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/path/{id} [delete]
func DeleteHieraIdEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		defer c.Done()

		s, err := DeleteOneHieraEntry(d.DB, u1.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": *s})
		}
	}
	return gin.HandlerFunc(fn)
}

func DeleteOneHieraEntry(d Database, id string) (*string, error) {
	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}

	collection := dbConn.Database(d.Database).Collection("hiera")

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

func InsertHieraIdEntry(id string, e map[string]interface{}, d Database) (*string, error) {

	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}

	// also need to retrieve the thing first and do it that way.
	// we should also keep not of the old entries maybe

	e["_id"] = id
	collection := dbConn.Database(d.Database).Collection("hiera")
	insertResult, err := collection.InsertOne(context.TODO(), e)
	if err != nil {
		return nil, err
	}
	defer dbConn.Disconnect(context.TODO())
	str := fmt.Sprintf("Inserted one entry id: %s", insertResult.InsertedID)
	return &str, nil
}

// HieraIdUpdateEndpoint example
// @Summary Updates an existing hiera path entry
// @Description Creates a new hiera path entry if it does not exist yet.
// @Param  id     path   string     true  "Some ID"
// @Param   data      body HieraDataExample true  "data"
// @Accept  json
// @Produce  json
// @Success 200 {object} APIMessage
// @Failure 500 {object} APIMessage
// @Router /hiera/path/{id} [put]
func HieraIdUpdateEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)

		defer c.Done()
		s, err := GetOneHieraEntry(d.DB, u1.ID)
		if s == nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Hiera path not found", "updated": false})
		} else {
			var u map[string]interface{}
			c.BindJSON(&u)
			if err != nil {
				log.Println(err.Error())
			}
			u["_id"] = u1.ID

			res, err := UpdateHieraIdEntry(u1.ID, u, d.DB)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

			} else {
				c.JSON(http.StatusOK, gin.H{"success": true, "message": *res})
			}
		}
	}
	return gin.HandlerFunc(fn)
}

func UpdateHieraIdEntry(id string, e map[string]interface{}, d Database) (*string, error) {

	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}

	collection := dbConn.Database(d.Database).Collection("hiera")
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

// HieraIdInsertEndpoint example
// @Summary Creates a hiera path entry
// @Description Creates a new hiera path entry if it does not exist yet.
// @Param  id     path   string     true  "Some ID"
// @Param   data      body HieraDataExample true  "data"
// @Accept  json
// @Produce  json
// @Success 200 {object} APIMessage
// @Failure 500 {object} APIMessage
// @Router /hiera/path/{id} [post]
func HieraIdInsertEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		var u map[string]interface{}
		err := c.BindJSON(&u)

		defer c.Done()

		s, err := InsertHieraIdEntry(u1.ID, u, d.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": *s})
		}
	}
	return gin.HandlerFunc(fn)
}

func GetAllHieraEntries(d Database) ([]*map[string]interface{}, error) {
	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection("hiera")
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

func GetOneHieraEntry(d Database, id string) (*map[string]interface{}, error) {
	dbConn, err := NewClient(d)
	if err != nil {
		return nil, err
	}
	collection := dbConn.Database(d.Database).Collection("hiera")
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
