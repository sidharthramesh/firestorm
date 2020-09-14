package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
)

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
			c := concept{}
			c.ID = id
			c.Codesystem = "snomed"
			data[id] = c
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

func buildPackage(dir string) {
	directory := path.Join(dir, "Snapshot", "Terminology")
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		log.Fatalf("Cannot read snomed directory: %v", err)
	}
	for _, file := range files {
		name := file.Name()
		filePath := path.Join(directory, name)
		if strings.HasPrefix(name, "sct2_Concept_Snapshot") {
			log.Printf("Building concepts from %v", filePath)
			buildConcepts(filePath)
		}
		if strings.HasPrefix(name, "sct2_Description_Snapshot") {
			log.Printf("Building terms from %v", filePath)
			buildTerms(filePath)
		}
		if strings.HasPrefix(name, "sct2_Relationship_Snapshot") {
			log.Printf("Building relationships from %v", filePath)
			buildRelationships(filePath)
		}
	}
}

func getSnomedPackages(directory string) []string {
	var snomedPackages []string
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		log.Fatalf("Error while reading snomed directory")
	}
	for _, file := range files {
		if file.IsDir() {
			dir := file.Name()
			log.Printf("Reading snomed package %v", dir)
			snomedPackages = append(snomedPackages, path.Join(directory, dir))
		}
	}
	return snomedPackages
}

func buildSnomed(directory string) {
	packages := getSnomedPackages(directory)
	for _, p := range packages {
		buildPackage(p)
	}
	log.Printf("Building Is A")
	buildIsA()
}
