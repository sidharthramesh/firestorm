package main

import (
	"encoding/csv"
	"io"
	"os"
)


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
			loincConcept := concept{}
			loincConcept.ID = id
			loincConcept.Name = longName
			loincConcept.Terms = terms
			loincConcept.Codesystem = "loinc"
			loincConcept.IsA = []string{"loinc"}

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