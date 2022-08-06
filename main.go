package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bdxio/cfp-to-trello/cfp"
	"github.com/bdxio/cfp-to-trello/geo"
	"github.com/bdxio/cfp-to-trello/importer"
	"github.com/bdxio/cfp-to-trello/publisher"
	"github.com/bdxio/cfp-to-trello/trello"
)

func main() {
	var organizationName string
	var trelloKey string
	var trelloSecret string
	var eventID string
	var jsonPath string
	var cfpKey string
	var importCFP bool
	var accept bool
	var reject bool
	var dryRun bool

	flag.StringVar(&organizationName, "org", "bdxio", "Organization name in Trello")
	flag.StringVar(&trelloKey, "trello-key", "", "Trello consumer key")
	flag.StringVar(&trelloSecret, "trello-secret", "", "Trello consumer secret")
	flag.StringVar(&eventID, "event-id", "", "Conference-Hall event ID")
	flag.StringVar(&jsonPath, "json", "", "Path to CFP export JSON file")
	flag.StringVar(&cfpKey, "cfp-key", "", "Conference-Hall API key")
	flag.BoolVar(&importCFP, "import", false, "Import CFP in Trello")
	flag.BoolVar(&accept, "accept", false, "Accept proposals in CFP")
	flag.BoolVar(&reject, "reject", false, "Reject proposals in CFP")
	flag.BoolVar(&dryRun, "dry-run", false, "Don't publish proposals, only logs the requests")
	flag.Parse()

	switch {
	case importCFP:
		runImport(organizationName, trelloKey, trelloSecret, eventID, jsonPath)
	case accept:
		runPublish(organizationName, trelloKey, trelloSecret, eventID, cfpKey, publisher.PublicationAccept, dryRun)
	case reject:
		runPublish(organizationName, trelloKey, trelloSecret, eventID, cfpKey, publisher.PublicationReject, dryRun)
	default:
		fmt.Println("One action is required: import, accept or reject")
		flag.Usage()
		os.Exit(1)
	}
}

func runImport(organizationName, trelloKey, trelloSecret, eventID, jsonPath string) {
	requireArg(organizationName, "org")
	requireArg(trelloKey, "trello-key")
	requireArg(trelloSecret, "trello-secret")
	requireArg(eventID, "event-id")
	requireArg(jsonPath, "json")

	client, err := trello.New(trelloKey, trelloSecret)
	if err != nil {
		log.Fatalf("Error while creating Trello Client: %v", err)
	}

	if err := importer.ImportCFP(organizationName, eventID, jsonPath, geo.FindLocation, client); err != nil {
		log.Fatalf("Error while importing CFP into Trello: %v", err)
	}
}

func runPublish(organizationName, trelloKey, trelloSecret, eventID, cfpKey string, pub publisher.Publication, dryRun bool) {
	requireArg(organizationName, "org")
	requireArg(trelloKey, "trello-key")
	requireArg(trelloSecret, "trello-secret")
	requireArg(eventID, "event-id")
	requireArg(cfpKey, "api-key")

	trelloClient, err := trello.New(trelloKey, trelloSecret)
	if err != nil {
		log.Fatalf("Error while creating Trello Client: %v", err)
	}

	cfpClient := cfp.NewConferenceHallClient(
		cfp.WithURL(cfp.URL),
		cfp.WithEventID(eventID),
		cfp.WithAPIKey(cfpKey),
		cfp.WithDryRun(dryRun),
	)

	if err := publisher.Publish(organizationName, cfpClient, trelloClient, pub); err != nil {
		log.Fatalf("Error while publishing to Conference-Hall: %v", err)
	}
}

func requireArg(value, name string) {
	if value != "" {
		return
	}
	fmt.Printf("%s argument is required\n", name)
	flag.Usage()
	os.Exit(1)
}
