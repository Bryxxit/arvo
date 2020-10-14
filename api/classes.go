package api

import (
	"fmt"
	"github.com/akira/go-puppetdb"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"io/ioutil"
	"log"
	"os"
)

type PuppetClass struct {
	IsPuppetManaged bool                          `json:"is_puppet_managed"`
	Environment     string                        `json:"environment"`
	Name            string                        `json:"name"`
	FileName        string                        `bson:"filename"json:"file_name"`
	IsDefined       bool                          `bson:"isdefined"json:"is_defined"`
	ClassType       string                        `bson:"classtype"json:"class_type"`
	Module          string                        `json:"module"`
	Path            string                        `json:"path"`
	Usage           int                           `json:"usage,omitempty"`
	Hosts           []string                      `json:"hosts,omitempty"`
	Code            string                        `json:"code"`
	SimpleCode      *[]PuppetClassSimpleCodeEntry `json:"simple_code"`
}

type PuppetClassSimpleCodeEntry struct {
	Name       string   `json:"name"`
	Depdencies []string `json:"dependencies"`
}

func getClassNameModuleFromPath(path string, env_dir string, fileName string) (string, string) {
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "\\", "/")

	}
	// if not absolute path remove ./ as path will be without it
	env_dir = strings.TrimPrefix(env_dir, "./")
	module := strings.Replace(path, env_dir, "", 1)
	// classes are mostly either in

	module = strings.TrimPrefix(module, "")
	module = strings.TrimPrefix(module, "/site/")
	module = strings.TrimPrefix(module, "/modules/")
	module = strings.ReplaceAll(module, "/manifests", "")
	module = strings.TrimPrefix(module, "/")
	dirs := strings.Split(module, "/")
	module = dirs[0]
	className := ""
	if len(dirs) > 2 {
		for _, d := range dirs {
			className = className + d + "::"
		}
		className = strings.TrimSuffix(className, ".pp::")
		// we have more dirs inside
	} else if len(dirs) == 2 {
		if fileName != "init.pp" {
			className = module + "::" + strings.TrimSuffix(dirs[1], ".pp")
		} else {
			className = module
		}
	}
	return module, className
}

func readFileToString(path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err.Error())
		return err.Error()
	}
	text := string(content)
	return text
}

func isDefinedOrNot(code string, name string) bool {
	lines := strings.Split(code, "\n")
	s := fmt.Sprintf("^([ \t]?)(class)[ \t]+(%s)(.*)", name)
	s2 := fmt.Sprintf("^([ \t]?)(define)[ \t]+(%s)(.*)", name)

	for _, line := range lines {
		match, _ := regexp.MatchString(s, line)
		match2, _ := regexp.MatchString(s2, line)
		if match {
			return false
		}
		if match2 {
			return true
		}
	}
	return false
}

func GetPuppetClassesFromDir(dir string, env string) []PuppetClass {
	var classes []PuppetClass
	env_dir := strings.TrimSuffix(dir, "/")
	err := filepath.Walk(env_dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, ".pp") {
				fileName := info.Name()
				if !strings.Contains(path, "/examples/") && !strings.Contains(path, "/spec/fixtures/") &&
					!strings.Contains(path, "\\spec\\fixtures\\") && !strings.Contains(path, "\\examples\\") {
					module, class := getClassNameModuleFromPath(path, env_dir, fileName)
					code := readFileToString(path)
					classType := "class"
					if strings.HasPrefix(class, "profile") {
						classType = "profile"
					}
					if strings.HasPrefix(class, "role") {
						classType = "role"
					}
					def := false
					if isDefinedOrNot(code, class) {
						classType = "defined"
						def = true
					}
					c := PuppetClass{
						IsPuppetManaged: true,
						Environment:     env,
						Name:            class,
						FileName:        fileName,
						IsDefined:       def,
						Module:          module,
						ClassType:       classType,
						Path:            path,
						Usage:           0,
						Code:            code,
						Hosts:           []string{},
					}
					if c.Name != "" && !strings.Contains(path, "/"+c.Module+"/types/") &&
						!strings.Contains(path, "\\"+c.Module+"\\types\\") {
						classes = append(classes, c)
					}
				}
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return classes
}

func CheckAndAddToUsage(usageArr *[]PuppetClass, resource puppetdb.Resource) {
	for index, c := range *usageArr {
		if strings.ToLower(c.Name) == strings.ToLower(resource.Title) {
			(*usageArr)[index].Usage = c.Usage + 1
			if !stringInSlice(resource.Certname, c.Hosts) {
				(*usageArr)[index].Hosts = append(c.Hosts, resource.Certname)
			}
			break
		}
	}
}
