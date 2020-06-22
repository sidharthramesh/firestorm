package main
import (
	"testing"
	"log"
	jsoniter "github.com/json-iterator/go"
)

// func TestEverything(t *testing.T) {
// 	data = make(map[string]concept)
// 	fmt.Println("Building Snomed")
// 	buildConcepts("SnomedCT_InternationalRF2_PRODUCTION_20200309T120000Z/Snapshot/Terminology/sct2_Concept_Snapshot_INT_20200309.txt")
// 	buildTerms("SnomedCT_InternationalRF2_PRODUCTION_20200309T120000Z/Snapshot/Terminology/sct2_Description_Snapshot-en_INT_20200309.txt")
// 	buildRelationships("SnomedCT_InternationalRF2_PRODUCTION_20200309T120000Z/Snapshot/Terminology/sct2_Relationship_Snapshot_INT_20200309.txt")
// 	fmt.Println("Building Snomed Drug extensions")
// 	buildConcepts("SnomedCT_IndiaDrugExtensionRF2_BETA_IN1000189_20191122T120000Z/Snapshot/Terminology/sct2_Concept_Snapshot_IN1000189_20191122.txt")
// 	buildTerms("SnomedCT_IndiaDrugExtensionRF2_BETA_IN1000189_20191122T120000Z/Snapshot/Terminology/sct2_Description_Snapshot-en_IN1000189_20191122.txt")
// 	buildRelationships("SnomedCT_IndiaDrugExtensionRF2_BETA_IN1000189_20191122T120000Z/Snapshot/Terminology/sct2_Relationship_Snapshot_IN1000189_20191122.txt")
// 	fmt.Println("Building Is A")
// 	buildIsA()
// 	fmt.Println("Building Loinc")
// 	buildLoinc("Loinc_2.67/LoincTable/Loinc.csv")
// 	yamlPath := "opthalmology.yaml"
// 	buildCustom(yamlPath)
// 	bytes, err := json.Marshal(data["193570009"])
// 	if err != nil {
// 		t.Errorf("%v", err)
// 	}
	
// 	if !(string(bytes) == `{"id":"193570009","name":"Cataract (disorder)","is_a":["10810001","11131000119108","118234003","118235002","118254002","118934005","123946008","128127008","128535002","138875005","246915008","247049005","247052002","301857004","301905003","362965005","362975008","371405004","371409005","404684003","406122000","418727003","62585004","64572001"],"readable":false,"writable":true,"writer":{"name":"eye-concept","props":{"data":123}},"terms":["Cataract","Lenticular opacity","Cataract (disorder)","Lens opacity"],"relationships":[{"type":{"id":"116680003","name":"Is a (attribute)"},"destination":{"id":"10810001","name":"Disorder of lens (disorder)"}},{"type":{"id":"116680003","name":"Is a (attribute)"},"destination":{"id":"62585004","name":"Degenerative disorder of eye (disorder)"}},{"type":{"id":"116676008","name":"Associated morphology (attribute)"},"destination":{"id":"128305008","name":"Abnormally opaque structure (morphologic abnormality)"}},{"type":{"id":"363698007","name":"Finding site (attribute)"},"destination":{"id":"78076003","name":"Structure of lens of eye (body structure)"}},{"type":{"id":"116680003","name":"Is a (attribute)"},"destination":{"id":"247052002","name":"Cataract finding (finding)"}}],"department":["opthalmology"],"codesystem":"snomed"}`){
// 		t.Errorf("JSON not equal")
// 	}
// }

func TestYaml(t *testing.T){
	// data = make(map[string]concept)
	config := readConfig("opthalmology.yaml")
	// log.Printf("%+v", config)
	json := jsoniter.ConfigFastest
	b , _ := json.Marshal(config)
	log.Printf(string(b))
	// fmt.Println("hello")
}