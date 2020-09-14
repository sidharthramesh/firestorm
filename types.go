package main

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

type baseConcept struct {
	ID            string                 `json:"id,omitempty"`
	Name          string                 `json:"name,omitempty"`
	IsA           []string               `json:"is_a"`
	Readable      bool                   `json:"readable,omitempty"`
	Writable      bool                   `json:"writable,omitempty"`
	Reader        map[string]interface{} `json:"reader,omitempty"`
	Writer        map[string]interface{} `json:"writer,omitempty"`
	Relationships *[]relationship        `json:"relationships,omitempty"`
	Department    []string               `json:"department,omitempty"`
	Codesystem    string                 `json:"codesystem,omitempty"`
}

type concept struct {
	Terms []string `json:"terms,omitempty"`
	baseConcept
}

type migrationAction struct {
	Concept concept `json:"concept"`
	Action  string  `json:"action"`
}

type weightedTerm struct {
	Input  string `json:"input,omitempty"`
	Weight int    `json:"weight,omitempty"`
}

type weightedConcept struct {
	Terms []weightedTerm `json:"terms,omitempty"`
	baseConcept
}

type esAction struct {
	ID     string
	Action string
	Index  string
	Body   []byte
}
