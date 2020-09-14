package api

import (
	"context"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	"log"
	"time"
)

func ExportToInfluxDB(c Conf) {
	CleanAll(c)
	ExportPerNodeMetrics(c)
	ExportCleanAllResultMetrics(c)

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
				writeApi.WritePoint(context.Background(), p)
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
