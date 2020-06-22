package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	jsoniter "github.com/json-iterator/go"
)

type node struct {
	children    []string
	parentCount int
	hits        int
	isA         map[string]bool
}
type idName struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}
type relationship struct {
	Type        idName `json:"type,omitempty"`
	Destination idName `json:"destination,omitempty"`
}
type concept struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	IsA           []string               `json:"is_a,omitempty"`
	Readable      bool                   `json:"readable"`
	Writable      bool                   `json:"writable"`
	Reader        map[string]interface{} `json:"reader,omitempty"`
	Writer        map[string]interface{} `json:"writer,omitempty"`
	Terms         []string               `json:"terms,omitempty"`
	Relationships *[]relationship        `json:"relationships,omitempty"`
	Department    []string               `json:"department,omitempty"`
	Codesystem    string                 `json:"codesystem"`
}

type elasticSearchAction struct {
	Action  string  `json:"action"`
	Concept concept `json:"concept"`
}

var data map[string]concept

func buildTree(tree map[string]node) {
	var child string
	var parent string
	for _, concept := range data {
		if concept.Relationships != nil {
			for _, rel := range *concept.Relationships {
				// isA relationship
				if rel.Type.ID == "116680003" {
					child = concept.ID
					parent = rel.Destination.ID
					parentNode, found := tree[parent]
					if !found {
						tree[parent] = node{children: []string{child}}
					} else {
						parentNode.children = append(parentNode.children, child)
						tree[parent] = parentNode
					}
					childNode, found := tree[child]
					if !found {
						tree[child] = node{parentCount: 1}
					} else {
						childNode.parentCount++
						tree[child] = childNode
					}
				}
			}
		}
	}
}

func bfs(tree map[string]node, start string) {
	var queue []string
	var current string
	queue = append(queue, start)
	for len(queue) > 0 {
		current, queue = queue[0], queue[1:]
		node := tree[current]
		for _, child := range node.children {
			childNode := tree[child]
			childNode.hits++
			if childNode.isA == nil {
				childNode.isA = make(map[string]bool)
			}
			childNode.isA[current] = true
			for key := range node.isA {
				childNode.isA[key] = true
			}
			tree[child] = childNode
			if childNode.hits == childNode.parentCount {
				queue = append(queue, child)
			}
		}
		concept := data[current]
		var isA []string
		for i := range node.isA {
			isA = append(isA, i)
		}
		sort.Strings(isA)
		concept.IsA = isA
		data[current] = concept

	}
}

func buildConcepts(path string) {
	file, _ := os.Open(path)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, "\t")
		id := tokens[0]
		active := tokens[2]
		var activeBool bool
		if active == "1" {
			activeBool = true
		}
		if activeBool {
			data[id] = concept{ID: id, Codesystem: "snomed"}
		}
	}
}

func buildTerms(path string) {
	file, _ := os.Open(path)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, "\t")
		active := tokens[2]
		conceptID := tokens[4]
		term := tokens[7]
		typeID := tokens[6]

		var fsn bool
		if typeID == "900000000000003001" {
			fsn = true
		}
		if active == "1" {
			concept, exists := data[conceptID]
			if exists {
				concept.Terms = append(concept.Terms, term)
				if fsn {
					concept.Name = term
				}
				data[conceptID] = concept
			}
		}
	}
}
func buildRelationships(path string) {
	file, _ := os.Open(path)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, "\t")
		active := tokens[2]
		relType := tokens[7]
		source := tokens[4]
		dst := tokens[5]
		var activeBool bool
		if active == "1" {
			activeBool = true
		}
		if activeBool {
			concept, exists := data[source]
			if exists {
				rel := new(relationship)
				rel.Type.ID = relType
				rel.Type.Name = data[relType].Name
				rel.Destination.ID = dst
				rel.Destination.Name = data[dst].Name
				var relationships []relationship
				if concept.Relationships != nil {
					relationships = append(*concept.Relationships, *rel)
				} else {
					relationships = []relationship{*rel}
				}
				concept.Relationships = &relationships
				data[source] = concept
			}
		}
	}
}
func buildIsA() {
	tree := make(map[string]node)
	buildTree(tree)
	bfs(tree, "138875005")
}

func buildLoinc(path string) {
	file, _ := os.Open(path)
	defer file.Close()
	r := csv.NewReader(file)

	for {
		tokens, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		id := tokens[0]
		status := tokens[11]
		shortName := tokens[22]
		longName := tokens[28]
		units := tokens[30]
		if status == "ACTIVE" {
			var terms []string
			terms = append(terms, longName, shortName)
			loincConcept := concept{ID: id, Name: longName, Terms: terms, Codesystem: "loinc"}
			unitsRel := new(relationship)
			if units != "" {
				unitsRel.Type.ID = "EXAMPLE_UCUM_UNITS"
				unitsRel.Type.Name = "units"
				unitsRel.Destination.ID = units
				unitsRel.Destination.Name = units
				relationships := []relationship{*unitsRel}
				loincConcept.Relationships = &relationships
			}
			data[id] = loincConcept
		}
	}
}

func buildEverything() {
	data = make(map[string]concept)
	log.Printf("Building Snomed")
	log.Printf("Building Concepts")
	buildConcepts("SnomedCT_InternationalRF2_PRODUCTION_20200309T120000Z/Snapshot/Terminology/sct2_Concept_Snapshot_INT_20200309.txt")
	log.Printf("Building Terms")
	buildTerms("SnomedCT_InternationalRF2_PRODUCTION_20200309T120000Z/Snapshot/Terminology/sct2_Description_Snapshot-en_INT_20200309.txt")
	log.Printf("Building Relationships")
	buildRelationships("SnomedCT_InternationalRF2_PRODUCTION_20200309T120000Z/Snapshot/Terminology/sct2_Relationship_Snapshot_INT_20200309.txt")
	log.Printf("Building Concepts: Drug Extension")
	buildConcepts("SnomedCT_IndiaDrugExtensionRF2_BETA_IN1000189_20191122T120000Z/Snapshot/Terminology/sct2_Concept_Snapshot_IN1000189_20191122.txt")
	log.Printf("Building Terms: Drug Extension")
	buildTerms("SnomedCT_IndiaDrugExtensionRF2_BETA_IN1000189_20191122T120000Z/Snapshot/Terminology/sct2_Description_Snapshot-en_IN1000189_20191122.txt")
	log.Printf("Building Relationships: Drug Extension")
	buildRelationships("SnomedCT_IndiaDrugExtensionRF2_BETA_IN1000189_20191122T120000Z/Snapshot/Terminology/sct2_Relationship_Snapshot_IN1000189_20191122.txt")
	log.Printf("Building Is A")
	buildIsA()
	log.Printf("Building Loinc")
	buildLoinc("Loinc_2.67/LoincTable/Loinc.csv")
	yamlPath := "opthalmology.yaml"
	log.Printf("Building Yaml: %v", yamlPath)
	buildCustom(yamlPath)
}
func sortKeys(data map[string]concept) []string {
	log.Printf("Sorting in order")
	var sortedKeys []string
	for key := range data {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)
	return sortedKeys
}
func writeSnapshot(outputPath string) {
	sortedKeys := sortKeys(data)
	log.Printf("Writing snapshot to file: %v", outputPath)
	f, err := os.Create(outputPath)
	defer f.Close()
	if err != nil {
		log.Panicf("%v", err)
	}
	w := bufio.NewWriter(f)
	defer w.Flush()
	json := jsoniter.ConfigFastest
	for _, key := range sortedKeys {
		concept := data[key]
		action := elasticSearchAction{Concept: concept, Action: "index"}
		bytes, err := json.Marshal(action)
		if err != nil {
			log.Panicf("%v", err)
		}
		w.Write(bytes)
		w.WriteString("\n")
	}
	log.Printf("Done")
}

func compareAndWriteMigration(migrationsFolder string) error {
	appliedMigrations, err := ioutil.ReadDir(migrationsFolder)
	if err != nil {
		return err
	}
	buildEverything()
	if len(appliedMigrations) == 0 {
		writeSnapshot(path.Join(migrationsFolder, "1.ndjson"))
		return nil
	}
	oldData := make(map[string]concept)
	var lastName string
	json := jsoniter.ConfigFastest
	log.Printf("Loading previous migrations")
	for _, file := range appliedMigrations {
		name := file.Name()
		f, err := os.Open(path.Join(migrationsFolder, name))
		defer f.Close()
		if err != nil {
			return err
		}
		reader := bufio.NewScanner(f)
		for reader.Scan() {
			var action elasticSearchAction
			json.Unmarshal(reader.Bytes(), &action)
			switch action.Action {
			case "index", "update":
				oldData[action.Concept.ID] = action.Concept
			case "delete":
				delete(oldData, action.Concept.ID)
			default:
				log.Panicf("Cannot understand action: %v", action.Action)
			}
		}
		lastName = name
	}
	lastNumber, err := strconv.Atoi(strings.Trim(lastName, ".ndjson"))
	if err != nil {
		return err
	}
	log.Printf("Comparing with previous migrations")
	var actions []elasticSearchAction
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for conceptID, newValue := range data {
			oldValue, exists := oldData[conceptID]
			if !exists {
				actions = append(actions, elasticSearchAction{Action: "index", Concept: newValue})
			} else {
				if !reflect.DeepEqual(oldValue, newValue) {
					actions = append(actions, elasticSearchAction{Action: "update", Concept: newValue})
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		for conceptID, oldValue := range oldData {
			_, exists := data[conceptID]
			if !exists {
				actions = append(actions, elasticSearchAction{Action: "delete", Concept: oldValue})
			}
		}
	}()
	wg.Wait()
	changes := len(actions)
	if changes == 0 {
		log.Printf("No changes detected")
		return nil
	} else {
		log.Printf("%v changes detected", changes)
	}
	newPath := path.Join(migrationsFolder, fmt.Sprintf("%v.ndjson", lastNumber+1))
	wf, err := os.Create(newPath)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(wf)
	defer writer.Flush()
	for _, action := range actions {
		b, err := json.Marshal(action)
		if err != nil {
			return err
		}
		writer.Write(b)
		writer.WriteString("\n")
	}
	return nil
}

func initIndex(url string, index string, recreate bool) (*elasticsearch.Client, bool) {
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
	if recreate {
		log.Printf("Deleting index %v", index)
		es.Indices.Delete([]string{index})
	}
	res, err = es.Indices.Exists([]string{index})
	if res.StatusCode == 404 {
		log.Printf("Index %v not present. Creating...", index)
		file, _ := os.Open("schema.json")
		defer file.Close()
		es.Indices.Create(index, es.Indices.Create.WithBody(file))
		// Delete lock file if present
		lockName := "migrate.lock"
		err := os.Remove(lockName)
		if err != nil {
			log.Printf("Lockfile does not exist. Did not remove.")
		} else {
			log.Printf("Removed Lockfile")
		}
		return es, true
	}
	log.Printf("Index %v exists", index)
	return es, false

}
func doMigrate(url string, index string, migrationFolder string, reset bool) error {
	es, _ := initIndex(url, index, reset)
	lockName := "migrate.lock"
	_, err := os.Stat(lockName)
	if os.IsNotExist(err) {
		log.Printf("Creating migration lock file")
		wf, err := os.Create("migrate.lock")
		if err != nil {
			return err
		}
		_, err = wf.WriteString("0")
		if err != nil {
			return err
		}
		err = wf.Sync()
		if err != nil {
			return err
		}
		wf.Close()
	}
	b, err := ioutil.ReadFile(lockName)
	currentInt, err := strconv.Atoi(string(b))
	var unappliedMigrations []string
	migrations, err := ioutil.ReadDir(migrationFolder)
	if err != nil {
		return err
	}

	for _, file := range migrations {
		i, err := strconv.Atoi(strings.Trim(file.Name(), ".ndjson"))
		if err != nil {
			return err
		}
		if i > currentInt {
			unappliedMigrations = append(unappliedMigrations, file.Name())
		}
	}
	if len(unappliedMigrations) == 0 {
		log.Printf("No unapplied migrations")
	} else {
		log.Printf("Unapplied migrations: %v", unappliedMigrations)
	}
	json := jsoniter.ConfigFastest

	for _, p := range unappliedMigrations {
		migrationPath := path.Join(migrationFolder, p)
		log.Printf("Applying %v", migrationPath)
		file, _ := os.Open(migrationPath)
		defer file.Close()
		r := bufio.NewScanner(file)
		var lines int
		for r.Scan() {
			lines++
		}
		file.Seek(0, io.SeekStart)

		log.Printf("Found %v entries", lines)

		bar := pb.StartNew(lines)
		r = bufio.NewScanner(file)
		bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
			Index:         index,            // The default index name
			Client:        es,               // The Elasticsearch client
			NumWorkers:    4,                // The number of worker goroutines
			FlushBytes:    int(5e+6),        // The flush threshold in bytes
			FlushInterval: 30 * time.Second, // The periodic flush interval
		})
		if err != nil {
			log.Fatalf("Error creating the indexer: %s", err)
		}

		for r.Scan() {
			var action elasticSearchAction
			json.Unmarshal(r.Bytes(), &action)
			b, _ := json.Marshal(action.Concept)
			switch action.Action {
			case "index", "update":
				err = bi.Add(
					context.Background(),
					esutil.BulkIndexerItem{
						Index: index,
						// Action field configures the operation to perform (index, create, delete, update)
						Action:     "index",
						DocumentID: action.Concept.ID,
						Body:       bytes.NewReader(b),
						OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
							bar.Increment()
						},
						OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
							log.Printf("%v", err)
							if err != nil {
								log.Printf("ERROR: %s", err)
							} else {
								log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
							}
						},
					},
				)
			case "delete":
				err = bi.Add(
					context.Background(),
					esutil.BulkIndexerItem{
						Index: index,
						// Action field configures the operation to perform (index, create, delete, update)
						Action:     "delete",
						DocumentID: action.Concept.ID,
						OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
							bar.Increment()
						},
						OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
							log.Printf("%v", err)
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
		i, err := strconv.Atoi(strings.Trim(p, ".ndjson"))

		ioutil.WriteFile(lockName, []byte(strconv.Itoa(i)), 0644)

	}
	return nil
}

func main() {
	makemigration := flag.NewFlagSet("makemigration", flag.ExitOnError)
	migrationFolder := makemigration.String("folder", "migrations", "Migrations folder")
	migrate := flag.NewFlagSet("migrate", flag.ExitOnError)
	reset := migrate.Bool("reset", false, "Resets the index")
	index := migrate.String("index", "concept", "Elasticsearch index name")
	if len(os.Args) < 2 {
		fmt.Println("expected 'makemigration' or 'migrate' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "makemigration":
		makemigration.Parse(os.Args[2:])
		compareAndWriteMigration(*migrationFolder)
	case "migrate":
		migrate.Parse(os.Args[2:])
		doMigrate("http://localhost:9200", *index, *migrationFolder, *reset)
		_ = index

	default:
		fmt.Println("expected 'makemigration' or 'migrate' subcommands")
		os.Exit(1)
	}
}
