package api

import (
	"errors"
	"fmt"
	"github.com/akira/go-puppetdb"
	"github.com/gin-gonic/gin"
	"github.com/jeremywohl/flatten"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// GetHierarchyEndPoint example
// @Summary Shows the hierarchies in your hiera.yaml file
// @Description Reads all the hierarchies from your hiera file and returns them.
// @Accept  json
// @Produce  json
// @Success 200 {object} HierarchyResult	""
// @Router /hierarchy [get]
func GetHierarchyEndPoint(conf Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		defer c.Done()

		h := GetPathsAndVarsInHierarchy(conf)
		c.JSON(http.StatusOK, h)

	}
	return gin.HandlerFunc(fn)
}

func GetPathsAndVarsInHierarchy(conf Conf) HierarchyResult {
	var hier HierarchyYamlFile
	hier.getConf(conf.HieraFile)
	paths_to_read := []string{}

	for _, h := range hier.Hierarchy {
		if h.Paths != nil {
			for _, p := range *h.Paths {
				paths_to_read = append(paths_to_read, conf.DataDir+"/"+p)
			}
		}
		if h.Path != nil {
			paths_to_read = append(paths_to_read, conf.DataDir+"/"+*h.Path)
			//log.Println(*h.Path)

		}
	}

	hiera_vars := []string{}
	for _, p := range paths_to_read {
		arr := getFactsFromPath(p)
		for _, fact := range arr {
			if !stringInSlice(fact, hiera_vars) {
				hiera_vars = append(hiera_vars, fact)
			}

		}
	}
	h := HierarchyResult{
		Paths:     paths_to_read,
		Variables: hiera_vars,
	}
	return h
}

// GetHierarchyForCertnameEndpoint example
// @Summary Get the hierachies for a specific host.
// @Description Transaltes the hierarchies in your hiera file into actual paths. By getting the facts from puppetdb.
// @Param  id     path   string     true  "Some ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} HierarchyResult	""
// @Failure 500 {object} APIMessage "Something went wrong getting the entry"
// @Router /hierarchy/{id} [get]
func GetHierarchyForCertnameEndpoint(conf Conf) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var u1 JSONID
		c.ShouldBindUri(&u1)
		defer c.Done()
		h, err := GetHierarchyForCertname(conf, u1.ID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		} else {
			c.JSON(http.StatusOK, h)
		}

	}
	return gin.HandlerFunc(fn)
}

// GetHierarchyForCertname Gets the hierarchy result for a certname if it exists
func GetHierarchyForCertname(conf Conf, certname string) (*HierarchyResult, error) {
	h := GetPathsAndVarsInHierarchy(conf)

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

func ReplaceFactInString(path string, factName string, value string) string {
	op1 := fmt.Sprintf("%%{::%s}", factName)
	op2 := fmt.Sprintf("%%{%s}", factName)
	str := strings.ReplaceAll(path, op1, value)
	str = strings.ReplaceAll(str, op2, value)
	return str
}

func GetFactsMapForCertName(conf Conf, certname string) map[string]interface{} {
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
	mapy := make(map[string]interface{})
	for i, fact := range facts {
		if i == 0 {
			mapy["environment"] = fact.Environment
		}
		switch (fact.Value.Data()).(type) {
		case map[string]interface{}:
			nested := fact.Value.Data().(map[string]interface{})
			flat, err := flatten.Flatten(nested, "", flatten.DotStyle)
			if err != nil {
				log.Println(err.Error())
			}
			for k, v := range flat {
				mapy[fact.Name+"."+k] = v

			}
		case interface{}:
			mapy[fact.Name] = fact.Value.Data()
		default:
			log.Println("Unknown data type was parsed in facts of this host " + certname + " fact " + fact.Name)
		}
	}
	return mapy
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

func getFactNameFromHieraVar(variable string) string {
	str := strings.ReplaceAll(variable, "%{", "")
	str = strings.TrimSuffix(str, "}")
	str = strings.TrimPrefix(str, "::")
	return str
}
