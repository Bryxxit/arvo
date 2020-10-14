package api

import (
	"encoding/json"
	"fmt"
	"github.com/akira/go-puppetdb"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func GetResourceTypePerHost(client puppetdb.Client, certname string, resourceType string) *[]puppetdb.Resource {
	query := fmt.Sprintf("resources { type = \"%s\" and nodes { certname = \"%s\"} }", resourceType, certname)
	nUrl := fmt.Sprintf("%s/pdb/query/v4?query=%s", client.BaseURL, url.QueryEscape(query))
	httpClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}
	req, err := http.NewRequest(http.MethodGet, nUrl, nil)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Println(getErr.Error())
		return nil
	}
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Println(readErr.Error())
		return nil
	}
	resources := []puppetdb.Resource{}
	json.Unmarshal(body, &resources)
	return &resources
}

func GetUnusedFiles(c Conf) []string {

	puppetFileNames := GetModuleFilesFromPuppetdb(c)
	dirFileNames := GetModuleFilesFromDir(c.CodeDir)
	unusedFiles := []string{}
	for _, file := range dirFileNames {
		if !stringInSlice(file, puppetFileNames) {
			unusedFiles = append(unusedFiles, file)
		}
	}

	return unusedFiles
}

func GetModuleFilesFromPuppetdb(c Conf) []string {
	var client *puppetdb.Client
	if !c.Puppet.SSL {
		client = puppetdb.NewClient(c.Puppet.Host, c.Puppet.Port, false)

	} else {
		if c.Puppet.Insecure {
			client = puppetdb.NewClientSSLInsecure(c.Puppet.Host, c.Puppet.Port, false)

		} else {
			client = puppetdb.NewClientSSL(c.Puppet.Host, c.Puppet.Port, c.Puppet.Key, c.Puppet.Cert, c.Puppet.Ca, false)

		}
	}
	nodes, _ := client.Nodes()
	puppetFileNames := []string{}
	for _, n := range nodes {
		puppetUsedFiles := GetResourceTypePerHost(*client, n.Certname, "File")
		if puppetUsedFiles != nil {

			for _, rec := range *puppetUsedFiles {
				// We only need two types of files and that ares filew wwith the source key and files
				if val, ok := rec.Paramaters["source"]; ok {
					switch v := val.(type) {
					case string:
						str := fmt.Sprintf("%v", v)
						test, loc := IsPuppetSourceAndLoc(str)
						if test {
							if !stringInSlice(loc, puppetFileNames) {
								puppetFileNames = append(puppetFileNames, loc)
							}
						}
					case []interface{}:
						for _, source := range v {
							str := fmt.Sprintf("%v", source)
							test, loc := IsPuppetSourceAndLoc(str)
							if test {
								if !stringInSlice(loc, puppetFileNames) {
									puppetFileNames = append(puppetFileNames, loc)
								}
							}
						}
					default:
						log.Println("Unrecognized type source should be string or array of string")
					}
				}

			}
		}
	}
	return puppetFileNames
}

func GetModuleFilesFromDir(dir string) []string {
	dirFileNames := []string{}
	filesInModules := ListAllFilesInDir(dir)

	for _, f := range filesInModules {
		filesInstring := regexp.MustCompile("files")
		matches := filesInstring.FindAllStringIndex(f, -1)
		if len(matches) >= 2 {
			splited := strings.Split(f, "files")
			splitted2 := splited[1:]
			f = splited[0] + "files" + strings.Join(splitted2, "files-arvo-temp-replacement")
		}

		re := regexp.MustCompile("(.*\\/|)(.*)(\\/)(files)(\\/)(.*)")
		result := re.FindAllStringSubmatch(f, -1)
		if result != nil {
			for _, r := range result {
				module, file := GetModuleAndFilePathFromRegexResult(r)
				if module != "" && file != "" {
					str := fmt.Sprintf("%s/%s", module, file)
					dirFileNames = append(dirFileNames, str)
				}

			}
		}
	}
	return dirFileNames
}

func GetModuleAndFilePathFromRegexResult(result []string) (string, string) {
	fileIndex := -1
	for i, r := range result {
		if r == "files" {
			if i > 2 {
				fileIndex = i
			}
		}
	}
	if len(result) > fileIndex+2 {
		module := result[fileIndex-2]
		file := result[fileIndex+2]
		file = strings.ReplaceAll(file, "files-arvo-temp-replacement", "files")
		return module, file
	}

	return "", ""

}

func ListAllFilesInDir(dir string) []string {
	paths := []string{}
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// ensure windows \ becomes /
			dir = strings.ReplaceAll(dir, "\\", "/")
			dir = strings.TrimPrefix(dir, "./")
			path = strings.ReplaceAll(path, "\\", "/")
			path = strings.TrimPrefix(path, dir)
			path = strings.TrimPrefix(path, "/")
			if !info.IsDir() {
				paths = append(paths, path)
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return paths
}

// IsPuppetSourceAndLoc Tests if the file is retrieved from inside the puppet code base and returns location if true
func IsPuppetSourceAndLoc(str string) (bool, string) {
	loc := ""
	check := false
	if strings.HasPrefix(str, "puppet:///modules/") {
		check = true
		loc = strings.TrimPrefix(str, "puppet:///modules/")

	}

	return check, loc
}
