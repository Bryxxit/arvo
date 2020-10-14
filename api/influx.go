package api

import (
	"context"
	"fmt"
	"github.com/akira/go-puppetdb"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func ExportToInfluxDB(c Conf) {
	CleanAll(c)
	//drop databse
	query := fmt.Sprintf("DROP DATABASE %s", c.Bucket)
	c1 := DoRequest(c, query)
	time.Sleep(2 * time.Second)
	query = fmt.Sprintf("CREATE DATABASE %s", c.Bucket)
	c2 := DoRequest(c, query)
	if c1 && c2 {
		ExportPerNodeMetrics(c)
		ExportCleanAllResultMetrics(c)
		if c.ScanFiles {
			exportFilesToInfluxDB(c)
		}
		if c.ScanClasses {
			exportUnusedClassesToInflux(c)
		}
	}
}

func exportFilesToInfluxDB(c Conf) {
	unused := GetUnusedFiles(c)
	client := influxdb2.NewClient(c.Url, "")

	writeApi := client.WriteApiBlocking("", c.Bucket)
	for _, file := range unused {
		module := strings.Split(file, "/")[0]
		p := influxdb2.NewPoint("arvo-unused-files",
			map[string]string{"module": module},
			map[string]interface{}{"path": file},
			time.Now())
		err := writeApi.WritePoint(context.Background(), p)
		if err != nil {
			client = influxdb2.NewClient(c.Url, "")
			writeApi = client.WriteApiBlocking("", c.Bucket)
			writeApi.WritePoint(context.Background(), p)

		}
	}

}

func exportUnusedClassesToInflux(c Conf) {
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

	allClasses := GetPuppetClassesFromDir(c.CodeDir, c.PuppetEnv)

	for _, n := range nodes {
		nodeClasses := GetResourceTypePerHost(*client, n.Certname, "Class")
		for _, c := range *nodeClasses {
			CheckAndAddToUsage(&allClasses, c)
		}
	}

	clientI := influxdb2.NewClient(c.Url, "")
	writeApi := clientI.WriteApiBlocking("", c.Bucket)
	for _, class := range allClasses {
		if class.Usage == 0 && !class.IsDefined {
			p := influxdb2.NewPoint("arvo-unused-classes",
				map[string]string{"module": class.Module, "path": class.Path, "classType": class.ClassType},
				map[string]interface{}{"usage": 0, "name": class.Name},
				time.Now())
			err := writeApi.WritePoint(context.Background(), p)
			if err != nil {
				clientI = influxdb2.NewClient(c.Url, "")
				writeApi = clientI.WriteApiBlocking("", c.Bucket)
				writeApi.WritePoint(context.Background(), p)

			}
		}
	}
}

func DoRequest(c Conf, query string) bool {
	params := url.Values{}
	params.Add("q", query)
	output := params.Encode()

	uri := fmt.Sprintf("%squery?%s", c.Url, output)
	if !strings.HasSuffix(c.Url, "/") {
		uri = fmt.Sprintf("%s/query?%s", c.Url, output)
	}
	//log.Println(uri)
	res, err := http.Post(uri, "", nil)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Println(fmt.Sprintf("status code error: %d %s", res.StatusCode, res.Status))
		return false
	}
	return true
}

func ExportCleanAllResultMetrics(c Conf) {
	client := influxdb2.NewClient(c.Url, "")

	writeApi := client.WriteApiBlocking("", c.Bucket)
	res, _ := GetFullCleanResultEntry(c.DB)
	if res != nil {
		for _, key := range res.KeysNeverUsed {
			for _, path := range key.Paths {
				p := influxdb2.NewPoint("arvo-all-keys-unused",
					map[string]string{"key": key.Key},
					map[string]interface{}{"path": path},
					time.Now())
				// write point immediately
				err := writeApi.WritePoint(context.Background(), p)
				if err != nil {
					client = influxdb2.NewClient(c.Url, "")
					writeApi = client.WriteApiBlocking("", c.Bucket)
					writeApi.WritePoint(context.Background(), p)

				}
			}
		}
		for _, path := range res.PathsNeverUsed {
			p := influxdb2.NewPoint("arvo-all-paths-unused",
				map[string]string{"path": path},
				map[string]interface{}{"unused": 1},
				time.Now())
			// write point immediately
			writeApi.WritePoint(context.Background(), p)
		}
	}
}

//func pathsToMapStringInterface(paths []string) {}

func ExportPerNodeMetrics(c Conf) {
	client := influxdb2.NewClient(c.Url, "")
	writeApi := client.WriteApiBlocking("", c.Bucket)
	//nodes
	nodes, err := GetAllCertnameLogEntry(c.DB)
	if err != nil {
		log.Println(err.Error())

	} else {
		for _, n := range nodes {
			res, _ := CleanUpResultLookupForOneCertname(c, n.ID)
			if res != nil {
				/// export the duplicates
				for _, key := range res.DuplicateData {
					for _, path := range key.Paths {
						p := influxdb2.NewPoint("arvo-duplicates",
							map[string]string{"certname": n.ID, "path": path},
							map[string]interface{}{"key": key.Key},
							time.Now())
						// write point immediately
						writeApi.WritePoint(context.Background(), p)
					}

				}
				/// export in hiera not in log
				for _, key := range res.InHieraNotInLog {
					for _, path := range key.Paths {
						p := influxdb2.NewPoint("arvo-not-log",
							map[string]string{"certname": n.ID, "path": path},
							map[string]interface{}{"key": key.Key},
							time.Now())
						// write point immediately
						writeApi.WritePoint(context.Background(), p)
					}

				}
				/// export in log and hiera
				for _, key := range res.InHieraNotInLog {
					for _, path := range key.Paths {
						p := influxdb2.NewPoint("arvo-log-hiera",
							map[string]string{"certname": n.ID, "path": path},
							map[string]interface{}{"key": key.Key},
							time.Now())
						// write point immediately
						writeApi.WritePoint(context.Background(), p)
					}

				}
				/// export in log not in hiera
				for _, key := range res.InLogNotInHiera {
					p := influxdb2.NewPoint("arvo-not-hiera",
						map[string]string{"certname": n.ID},
						map[string]interface{}{"key": key},
						time.Now())
					// write point immediately
					writeApi.WritePoint(context.Background(), p)

				}

			}

		}

	}

}
