package api

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// VariableIdsEndpoint example
// @Summary Get all variable paths in your configuration
// @Description Gets all the variable paths that are in your configuration file.
// @Accept  json
// @Produce  json
// @Success 200 {object} APIArrayMessage
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/variable/hierarchy [get]
func VariableIdsEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		defer c.Done()
		c.JSON(http.StatusOK, gin.H{"success": true, "paths": d.Hierarchy})
	}
	return gin.HandlerFunc(fn)
}

// VariableIdEndpoint example
// @Summary Get all variable paths for a specific host
// @Description Translates the
// @Param  id     path   string     true  "Some ID"
//@Accept  json
// @Produce  json
// @Success 200 {object} HierarchyResult	""
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/variable/hierarchy/{id} [get]
func VariableIdEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {

		var u1 JSONID
		c.ShouldBindUri(&u1)
		defer c.Done()
		h, err := GetVirtualHierachyForNode(d, u1.ID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		} else {
			c.JSON(http.StatusOK, h)
		}

	}
	return gin.HandlerFunc(fn)
}

// VariablePathIdEndpoint example
// @Summary Get a hiera path
// @Description Get the data from one hiera path
// @Param  id     path   string     true  "Some ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/variable/path/{id} [get]
func VariablePathIdEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		defer c.Done()

		s, err := GetOneStringMapEntryFromCollection(d.DB, u1.ID, "variable")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, *s)
		}
	}
	return gin.HandlerFunc(fn)
}

// VariablePathIdEndpoint example
// @Summary Get a hiera path
// @Description Get the data from one hiera path
// @Param  id     path   string     true  "Some key"
// @Param  certname     path   string     true  "Some certname"
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/value/{id}/{certname} [get]
func HieraValueIdEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 HIERAKEYID
		err := c.ShouldBindUri(&u1)
		defer c.Done()
		if err != nil || u1.ID == "" || u1.Certname == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Id and certname need to be given!!"})
		} else {
			values := GetHieraValue(d, u1.ID, u1.Certname)
			if values != nil {
				c.JSON(http.StatusOK, values)
			} else {
				c.JSON(http.StatusOK, gin.H{})
			}
		}
	}
	return gin.HandlerFunc(fn)
}

//switch (fact.Value.Data()).(type) {
//case map[string]interface{}:
//nested := fact.Value.Data().(map[string]interface{})
//flat, err := flatten.Flatten(nested, "", flatten.DotStyle)
//if err != nil {
//log.Println(err.Error())
//}
//for k, v := range flat {
//mapy[fact.Name+"."+k] = v
//
//}
//case interface{}:
//mapy[fact.Name] = fact.Value.Data()
//default:

func GetHieraValue(conf Conf, key string, certname string) *map[string]interface{} {
	values, err := GetOneStringMapEntryFromCollection(conf.DB, key, "hiera")
	if err != nil {
		return nil
	}
	// first we need the paths
	h, err := GetVirtualHierachyForNode(conf, certname)

	if err != nil {
		return nil
	}

	maps := []map[string]interface{}{}
	for _, p := range h.Paths {
		s, err := GetOneStringMapEntryFromCollection(conf.DB, p, "variable")
		if err == nil && s != nil {
			maps = append(maps, *s)
		}
	}

	// TODO this must become a recursive function I think
	for key, val := range *values {
		switch val.(type) {
		case string:
			(*values)[key] = GetReplaceString(val, maps)
		default:

		}

	}

	return values
}

func GetReplaceString(val interface{}, maps []map[string]interface{}) string {
	if strings.Contains(val.(string), "${arvo::") {
		vars := GetVariableFromValue(val.(string))
		for _, v := range vars {
			variableValue := GetFirstValueFromMaps(maps, v)
			str := val.(string)
			if variableValue != nil {
				str = ReplaceVariabelInString(val.(string), v, (*variableValue).(string))
			}
			return str
		}
		//v := GetFirstValueFromMaps(maps, key)
		//if v != nil {
		//
		//}
	}
	return val.(string)
}

// TODO max of 50 VARIABLES THIS IS NOT A GOOD SOLUTION
func GetVariableFromValue(path string) []string {
	counter := 0
	str := path
	facts := []string{}
	for counter < 50 {
		found, _ := regexp.MatchString("(.*)?([$][{][arvo::].*[}]).*", str)
		if found {
			regex := *regexp.MustCompile(`(.*)?([$][{][arvo::].*[}]).*`)
			hostGroupMatch := regex.FindStringSubmatch(str)
			fact := getFactNameFromArvoVar(hostGroupMatch[2])
			fact = strings.TrimPrefix(fact, "arvo::")
			facts = append(facts, fact)
			str = strings.ReplaceAll(str, hostGroupMatch[2], fact)

		} else {
			counter = 50
		}
		counter = counter + 1
	}
	return facts
}

func getFactNameFromArvoVar(variable string) string {
	str := strings.ReplaceAll(variable, "${", "")
	str = strings.TrimSuffix(str, "}")
	str = strings.TrimPrefix(str, "::")
	return str
}

func ReplaceVariabelInString(path string, factName string, value string) string {
	op1 := fmt.Sprintf("${arvo::%s}", factName)
	str := strings.ReplaceAll(path, op1, value)
	return str
}

func GetFirstValueFromMaps(maps []map[string]interface{}, key string) *interface{} {
	for _, m := range maps {
		if val, ok := m[key]; ok {
			return &val
		}
	}
	return nil
}

// VariablePathIdsEndpoint example
// @Summary Get all variable path ids
// @Description Gets all the ids of your your variable paths
// @Accept  json
// @Produce  json
// @Success 200 {object} APIArrayMessage
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hiera/variable/path [get]
func VariablePathIdsEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		defer c.Done()

		s, err := GetAllStringMapEntriesFromDB(d.DB, "variable")
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

// VariablePathIdInsertEndpoint example
// @Summary Creates a variable path entry
// @Description Creates a new variable path entry if it does not exist yet
// @Param  id     path   string     true  "Some ID"
// @Param   data      body HieraDataExample true  "data"
// @Accept  json
// @Produce  json
// @Success 200 {object} APIMessage
// @Failure 500 {object} APIMessage
// @Router /hiera/variable/path/{id} [post]
func VariablePathIdInsertEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		var u map[string]interface{}
		err := c.BindJSON(&u)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Malformed json try again please!!"})

		}
		defer c.Done()

		s, err := InsertStringMapEntry(u1.ID, u, d.DB, "variable")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": *s})
		}
	}
	return gin.HandlerFunc(fn)
}

// VariablePathIdUpdateEndpoint example
// @Summary Updates an existing variable path entry
// @Description Creates a new variable path entry if it does not exist yet.
// @Param  id     path   string     true  "Some ID"
// @Param   data      body HieraDataExample true  "data"
// @Accept  json
// @Produce  json
// @Success 200 {object} APIMessage
// @Failure 500 {object} APIMessage
// @Router /hiera/variable/path/{id} [put]
func VariablePathIdUpdateEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)

		defer c.Done()
		s, err := GetOneStringMapEntryFromCollection(d.DB, u1.ID, "variable")
		if s == nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Variable path not found", "updated": false})
		} else {
			var u map[string]interface{}
			c.BindJSON(&u)
			if err != nil {
				log.Println(err.Error())
			}
			u["_id"] = u1.ID

			res, err := UpdateStringMapEntry(u1.ID, u, d.DB, "variable")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

			} else {
				c.JSON(http.StatusOK, gin.H{"success": true, "message": *res})
			}
		}
	}
	return gin.HandlerFunc(fn)
}

// DeleteVariablePathIdEndpoint example
// @Summary Delete a vairable path entry
// @Description Deletes a variable entry from the database
// @Param  id     path   string     true  "Some ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} APIMessage
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router/hiera/variable/path/{id} [delete]
func DeleteVariablePathIdEndpoint(d Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		defer c.Done()

		s, err := DeleteOneStringMapEntry(d.DB, u1.ID, "variable")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})

		} else {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": *s})
		}
	}
	return gin.HandlerFunc(fn)
}

func GetVirtualHierachyForNode(conf Conf, certname string) (*HierarchyResult, error) {
	h := GetVariablesFromVirtualHierarchy(conf)

	facts := GetFactsMapForCertName(conf, certname)
	if len(facts) == 0 {
		return nil, errors.New("No facts found for node are you sure node exists or PuppetDB connection is valid")

	} else {
		for _, v := range h.Variables {
			for index, p := range h.Paths {
				if val, ok := facts[v]; ok {
					str := ReplaceFactInString(p, v, val.(string))
					h.Paths[index] = str
				}
			}
		}
		return &h, nil
	}
}

func GetVariablesFromVirtualHierarchy(conf Conf) HierarchyResult {
	hiera_vars := []string{}
	for _, p := range conf.Hierarchy {
		arr := getFactsFromPath(p)
		for _, fact := range arr {
			if !stringInSlice(fact, hiera_vars) {
				hiera_vars = append(hiera_vars, fact)
			}

		}
	}
	h := HierarchyResult{
		Paths:     conf.Hierarchy,
		Variables: hiera_vars,
	}
	return h
}
