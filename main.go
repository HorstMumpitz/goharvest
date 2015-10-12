package main

import (
	"fmt"

	"github.com/horstmumpitz/goharvest/oai"
)

func main() {
	fmt.Println("Hello Harvester!")
	req := &oai.Request{
		BaseUrl: "http://services.kb.nl/mdo/oai", Set: "DTS", MetadataPrefix: "dcx",
		From: "2012-09-06T014:00:00.000Z"}

	req.HarvestRecords(func(record *oai.Record) {
		fmt.Printf("%s\n\n", record.Metadata.Body[0:500])
	})

}
