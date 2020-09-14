# Firestorm

A Terminology Indexer in Go

## Setup
* Get all your snomed packages inside ./assets/snomed/
* Get Loinc under ./assets/Loinc_2.68
```
go build
```

## Usage
* Run elasticsearch on http://localhost:9200
```
firestorm makemigration
firestorm migrate
```