package api

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

// HieraIdsEndpoint example
// @Summary Get all hiera path ids
// @Description Gets all the ids of your paths so you can see which hiera paths are available.
// @Accept  json
// @Produce  json
// @Success 200 {object} APIArrayMessage
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/path [get]
func HieraIdsEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		defer c.Done()

		s, err := GetAllStringMapEntriesFromDB(d.DB, "hiera")
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

		s, err := GetOneStringMapEntryFromCollection(d.DB, u1.ID, "hiera")
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

		s, err := DeleteOneStringMapEntry(d.DB, u1.ID, "hiera")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": *s})
		}
	}
	return gin.HandlerFunc(fn)
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
		s, err := GetOneStringMapEntryFromCollection(d.DB, u1.ID, "hiera")
		if s == nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Hiera path not found", "updated": false})
		} else {
			var u map[string]interface{}
			c.BindJSON(&u)
			if err != nil {
				log.Println(err.Error())
			}
			u["_id"] = u1.ID

			res, err := UpdateStringMapEntry(u1.ID, u, d.DB, "hiera")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

			} else {
				c.JSON(http.StatusOK, gin.H{"success": true, "message": *res})
			}
		}
	}
	return gin.HandlerFunc(fn)
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

		s, err := InsertStringMapEntry(u1.ID, u, d.DB, "hiera")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": *s})
		}
	}
	return gin.HandlerFunc(fn)
}
