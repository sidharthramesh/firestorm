package main

import (
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	jsoniter "github.com/json-iterator/go"
)

func getUnappliedMigrations(migrationFolder string, lockValue int) []string {
	var migrations []string
	folders, _ := ioutil.ReadDir(migrationFolder)
	for _, f := range folders {
		if !f.IsDir() {
			fileName := f.Name()
			i, err := strconv.Atoi(strings.Trim(fileName, ".ndjson"))
			if err != nil {
				log.Panicf("Cannot convert to int : %v", err)
			}
			if i > lockValue {
				fileName = path.Join(migrationFolder, fileName)
				migrations = append(migrations, fileName)
			}
		}
	}
	return migrations
}

func getSubTerms(terms []string) []weightedTerm {
	var subterms []weightedTerm
	for _, term := range terms {
		subterms = append(subterms, weightedTerm{Input: term, Weight: 10})
		for _, substring := range strings.Split(term, " ") {
			subterms = append(subterms, weightedTerm{Input: substring, Weight: 9})
		}
	}
	return subterms
}
func getEsActions(action migrationAction) []esAction {
	json := jsoniter.ConfigFastest
	var actions []esAction
	a := action.Concept
	method := action.Action
	if a.Writable {
		var b weightedConcept
		b.ID = a.ID
		b.Name = a.Name
		b.IsA = a.IsA
		b.Writer = a.Writer
		b.Terms = getSubTerms(a.Terms)
		b.Relationships = a.Relationships
		b.Department = a.Department
		bts, err := json.Marshal(b)
		if err != nil {
			log.Panicf("%v", err)
		}
		esaction := esAction{ID: a.ID, Body: bts, Action: method, Index: "writable"}
		actions = append(actions, esaction)
	}
	if a.Readable {
		var b concept
		b.ID = a.ID
		b.Name = a.Name
		b.IsA = a.IsA
		b.Reader = a.Reader
		b.Terms = a.Terms
		b.Relationships = a.Relationships
		bts, err := json.Marshal(b)
		if err != nil {
			log.Panicf("%v", err)
		}
		esaction := esAction{ID: a.ID, Body: bts, Action: method, Index: "readable"}
		actions = append(actions, esaction)
	}
	var b weightedConcept
	b.ID = a.ID
	b.Name = a.Name
	b.IsA = a.IsA
	b.Terms = getSubTerms(a.Terms)
	b.Relationships = a.Relationships
	bts, err := json.Marshal(b)
	if err != nil {
		log.Panicf("%v", err)
	}
	esaction := esAction{ID: a.ID, Body: bts, Action: method, Index: "concept"}
	actions = append(actions, esaction)
	return actions
}

func initIndex(url string, recreate bool) (*elasticsearch.Client, bool) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			url,
		},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Panicf("%v", err)
	}
	res, err := es.Info()
	if err != nil {
		log.Panicf("%v", err)
	}
	indices := []string{"concept", "readable", "writable"}
	if recreate {
		log.Printf("Deleting index %v", indices)
		es.Indices.Delete(indices)
	}
	res, err = es.Indices.Exists(indices)
	if res.StatusCode == 404 {
		for _, index := range indices {
			log.Printf("Creating %v", index)
			file, _ := os.Open(index + ".json")
			defer file.Close()
			resp, _ := es.Indices.Create(index, es.Indices.Create.WithBody(file))
			log.Printf("%v", resp.Status())
		}
		return es, true
	}
	log.Printf("Index %v exists", indices)
	return es, false
}
func applyActions(url string, actions []esAction, reset bool) error {
	es, _ := initIndex(url, reset)
	bar := pb.StartNew(len(actions))
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        es,               // The Elasticsearch client
		NumWorkers:    4,                // The number of worker goroutines
		FlushBytes:    int(5e+6),        // The flush threshold in bytes
		FlushInterval: 30 * time.Second, // The periodic flush interval
	})
	for _, action := range actions {
		// log.Printf("%v - %v:  %v - %v", action.Concept["is_a"], action.Concept["id"], action.Concept["name"], action.Index)
		switch action.Action {
		case "index", "update":
			err = bi.Add(
				context.Background(),
				esutil.BulkIndexerItem{
					Index:      action.Index,
					Action:     "index",
					DocumentID: action.ID,
					Body:       bytes.NewReader(action.Body),
					OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
						bar.Increment()
					},
					OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
						if err != nil {
							log.Printf("ERROR: %s", err)
						} else {
							log.Printf("ERROR: %s", res.Error.Reason)
						}
					},
				},
			)
		case "delete":
			err = bi.Add(
				context.Background(),
				esutil.BulkIndexerItem{
					Index: action.Index,
					// Action field configures the operation to perform (index, create, delete, update)
					Action:     "delete",
					DocumentID: action.ID,
					OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
						bar.Increment()
					},
					OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
						if err != nil {
							log.Printf("ERROR: %s", err)
						} else {
							log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
						}
					},
				},
			)
		}

		if err != nil {
			log.Fatalf("Unexpected error: %s", err)
		}
	}
	err = bi.Close(context.Background())
	if err != nil {
		log.Panicf("%v", err)
	}
	bar.Finish()
	stats := bi.Stats()
	log.Printf("Stats: %+v", stats)
	log.Printf("Finished: %v. Errors: %v", stats.NumFlushed, stats.NumFailed)
	log.Printf("Supposed to fail: %v (%v)", 2, "SNOMED CT Concept and Cefsac")
	return nil
}

func doMigration(url string, reset bool, paths []string, lockName string) {
	json := jsoniter.ConfigFastest
	for i, p := range paths {
		log.Printf("Applying %v: ", p)
		number := strings.Trim(path.Base(p), ".ndjson")
		numberInt, err := strconv.Atoi(number)
		if err != nil {
			log.Panicf("Error converting lockfile to int: %v", err)
		}
		defer writeLock(lockName, numberInt)
		f, err := os.Open(p)
		if err != nil {
			log.Panicf("Error opening file: %v", err)
		}
		defer f.Close()
		r := bufio.NewScanner(f)
		var allActions []esAction
		for r.Scan() {
			line := r.Bytes()
			var action migrationAction
			json.Unmarshal(line, &action)
			allActions = append(allActions, getEsActions(action)...)
		}
		if i > 0 {
			reset = false
		}
		log.Printf("Found %v actions", len(allActions))
		err = applyActions(url, allActions, reset)
		if err != nil {
			log.Printf("Error in applying actions: %v", err)
		}
	}
}
