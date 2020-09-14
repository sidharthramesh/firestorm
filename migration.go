package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

func sortKeys(data map[string]concept) []string {
	log.Printf("Sorting in order")
	var sortedKeys []string
	for key := range data {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)
	return sortedKeys
}

func writeMigration(outputFolder string, filename string, actions []migrationAction) {
	filePath := path.Join(outputFolder, filename)
	log.Printf("Writing snapshot to file: %v", filePath)
	f, err := os.Create(filePath)
	if err != nil {
		log.Panicf("Error while creating %v: %v", filename, err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	for _, action := range actions {
		bytes, err := json.Marshal(action)
		if err != nil {
			log.Panicf("Error in json: %v", err)
		}
		w.Write(bytes)
		w.WriteString("\n")
	}
	defer w.Flush()
	log.Printf("Snapshot saved at: %v", filePath)
}

// Calculates diff the last applied migration and the snapshots afterwards
func calculateDiff(migrationFolder string, lastApplied int) []migrationAction {
	files, err := ioutil.ReadDir(migrationFolder)
	if err != nil {
		log.Panicf("%v", err)
	}
	oldData := make(map[string]concept)
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	for i, file := range files {
		if i+1 <= lastApplied {
			filePath := path.Join(migrationFolder, file.Name())
			log.Printf("Path: %v", filePath)
			f, err := os.Open(filePath)
			defer f.Close()
			if err != nil {
				log.Panicf("%v", err)
			}
			r := bufio.NewScanner(f)
			log.Printf("Building oldData from %v", filePath)
			for r.Scan() {
				bytes := r.Bytes()
				var action migrationAction
				json.Unmarshal(bytes, &action)
				switch action.Action {
				case "update", "index":
					oldData[action.Concept.ID] = action.Concept
				case "delete":
					delete(oldData, action.Concept.ID)
				}
			}
		}
	}
	log.Printf("Old data length: %v", len(oldData))
	log.Printf("Comparing...")
	var wg sync.WaitGroup
	wg.Add(2)
	var actions []migrationAction
	go func() {
		defer wg.Done()
		for conceptID, newValue := range data {
			oldValue, exists := oldData[conceptID]
			if !exists {
				// log.Printf("Does not exist\n%+v", newValue)
				actions = append(actions, migrationAction{Concept: newValue, Action: "index"})
			} else {
				if !reflect.DeepEqual(oldValue, newValue) {
					// log.Printf("%+v\n%+v", newValue, oldValue)
					actions = append(actions, migrationAction{Concept: newValue, Action: "update"})
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		for conceptID, oldValue := range oldData {
			_, exists := data[conceptID]
			if !exists {
				actions = append(actions, migrationAction{Concept: oldValue, Action: "delete"})
			}
		}
	}()
	wg.Wait()
	return actions
}

func getLock(name string) int {
	bytes, err := ioutil.ReadFile(name)
	if err != nil {
		if err.Error() == fmt.Sprintf("open %v: The system cannot find the file specified.", name) {
			ioutil.WriteFile(name, []byte("0"), 0644)
			return 0
		}
		log.Panicf("Error getting lock: %v", err)

	}
	i, err := strconv.Atoi(string(bytes))
	if err != nil {
		log.Panicf("Error getting lock: %v", err)
	}
	return i
}

func writeLock(name string, value int) {
	ioutil.WriteFile(name, []byte(fmt.Sprintf("%v", value)), 0644)
}
