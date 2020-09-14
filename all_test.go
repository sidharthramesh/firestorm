package main

import (
	"log"
	"testing"
)
func TestBuild(t *testing.T) {
	buildEverything()
	actions := calculateDiff("migrations", 1)
	writeMigration("migrations", "2.ndjson", actions)
	log.Printf("%v", "Done")
}

func TestLock(t *testing.T){
	log.Printf("%v", getUnappliedMigrations("migrations", 1))
}


func TestEs(t *testing.T){
	files := getUnappliedMigrations("migrations", 0)
	files = files[:1]
	doMigration("http://localhost:9200", true, files, "es.lock")
}

func TestDiff(t *testing.T){
	buildEverything()
	diff := calculateDiff("migrations", 0)
	log.Printf("%v", diff)
}
func TestInitIndex(t *testing.T){
	initIndex("http://localhost:9200", true)
}

func TestIndexCreate(t *testing.T) {

}
