package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var data map[string]concept

func buildEverything() {
	data = make(map[string]concept)
	log.Printf("Building Snomed")
	buildSnomed("assets/snomed")
	log.Printf("Building Loinc")
	buildLoinc("assets/Loinc_2.68/LoincTable/Loinc.csv")
	yamlPath := "opthalmology.yaml"
	log.Printf("Building Yaml: %v", yamlPath)
	buildCustom(yamlPath)
	// buildSubterms()
}

func main() {
	makemigration := flag.NewFlagSet("makemigration", flag.ExitOnError)
	migrationFolder := makemigration.String("folder", "migrations", "Migrations folder")
	migrate := flag.NewFlagSet("migrate", flag.ExitOnError)
	reset := migrate.Bool("reset", false, "Resets the index")
	if len(os.Args) < 2 {
		fmt.Println("expected 'makemigration' or 'migrate' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "makemigration":
		os.MkdirAll(*migrationFolder, os.ModePerm)
		makemigration.Parse(os.Args[2:])
		lock := getLock("migrate.lock")
		buildEverything()
		actions := calculateDiff(*migrationFolder, lock)
		diffLength := len(actions)
		if diffLength == 0 {
			log.Println("No changes detected")
		} else {
			defer writeLock("migrate.lock", lock+1)
			fileName := fmt.Sprintf("%v.ndjson", lock+1)
			log.Printf("Detected %v changes. Writing to file: %v", diffLength, fileName)
			writeMigration(*migrationFolder, fileName, actions)
		}
	case "migrate":
		migrate.Parse(os.Args[2:])
		lockName := "es.lock"
		if *reset {
			writeLock(lockName, 0)
		}
		lock := getLock(lockName)
		unApplied := getUnappliedMigrations(*migrationFolder, lock)
		if len(unApplied) == 0 {
			log.Println("No new migrations to apply. Try with -reset?")
		}
		doMigration("http://localhost:9200", *reset, unApplied, lockName)

	default:
		fmt.Println("expected 'makemigration' or 'migrate' subcommands")
		os.Exit(1)
	}
}
