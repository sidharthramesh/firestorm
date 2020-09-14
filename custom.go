package main

import (
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"github.com/Navops/yaml"
)

//A CustomConcept represents a derived concept
type CustomConcept struct {
	Name   string
	Reader map[string]interface{}
	Writer map[string]interface{}
}

// A Config represents the config
type Config struct {
	Department string
	Concepts   [] map[string] CustomConcept
	Refsets    map[string][]string
}

func buildCustom(path string) {
	config := readConfig(path)
	expressionRegex := regexp.MustCompile(`\<[\w-]+$`)
	refsetRegex := regexp.MustCompile(`\^[\w-]+$`)
	derivedRegex := regexp.MustCompile(`[\w-]:[\w-]+$`)
	for _, customConceptMap := range config.Concepts {
		for key, customConcept := range customConceptMap {
			var toBuild []string
			switch {
			case expressionRegex.MatchString(key):
				expression := strings.Trim(key, "<")
				for key, toSearch := range data {
				checkIsA:
					for _, i := range toSearch.IsA {
						if i == expression {
							toBuild = append(toBuild, key)
							break checkIsA
						}
					}
				}
			case refsetRegex.MatchString(key):
				refset := strings.Trim(key, "^")
				toBuild = append(toBuild, config.Refsets[refset]...)
			case derivedRegex.MatchString(key):
				parentKey := strings.Split(key, ":")[0]
				parentConcept, exists := data[parentKey]
				if customConcept.Name == "" {
					log.Printf("Name required for derived concept: %v. Skipping", key)
					break
				}
				if !exists {
					log.Printf("Parent concept for %v does not exist. Skipping", key)
					break
				}
				if parentConcept.Name == customConcept.Name {
					log.Printf("Parent concept and derived concept %v name cannot be the same. Skipping", key)
					break
				}
				derivedConcept := concept{}
				derivedConcept.ID = key
				derivedConcept.Name = customConcept.Name
				derivedConcept.IsA = parentConcept.IsA
				derivedConcept.Terms = append(parentConcept.Terms, customConcept.Name)
				derivedConcept.Relationships = parentConcept.Relationships
				derivedConcept.Codesystem = parentConcept.Codesystem
				log.Printf("Adding new concept: %v", key)
				data[key] = derivedConcept
				toBuild = append(toBuild, key)
			default:
				toBuild = append(toBuild, key)
			}
			// log.Printf("To build: %v", toBuild)
			for _, key := range toBuild {
				c, exists := data[key]
				if exists {
					if customConcept.Reader != nil {
						c.Reader = customConcept.Reader
						c.Readable = true
					}
					if customConcept.Writer != nil {
						c.Writer = customConcept.Writer
						c.Writable = true
					}
					c.Department = append(c.Department, config.Department)
					data[key] = c
				} else {
					log.Printf("Concept does not exist: %v. Skipping", key)
				}
		}
		// key := mapSlice.Key.(string)
		// customConcept := mapSlice.Value.(CustomConcept)
		}
	}
}

func readConfig(path string) Config {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("%v", err)
	}
	var configuration Config
	err = yaml.Unmarshal(bytes, &configuration)
	if err != nil {
		log.Printf("%v", err)
	}
	return configuration
}
