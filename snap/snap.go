package snap

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IBM/cloudant-go-sdk/cloudantv1"
	"github.com/IBM/cloudant-go-sdk/features"
)

type CloudantSnap struct {
	appConfig *AppConfig             // our command-line options
	service   *cloudantv1.CloudantV1 // the Cloudant SDK client
}

// New creates a new CloudantSnap struct, loading the CLI parameters
// and instantiating the Cloudant SDK client
func New() (*CloudantSnap, error) {
	// load the CLI parameters
	appConfig, err := NewAppConfig()
	if err != nil {
		return nil, err
	}

	// set up the Cloudant service
	service, err := cloudantv1.NewCloudantV1UsingExternalConfig(&cloudantv1.CloudantV1Options{})
	if err != nil {
		return nil, err
	}
	service.EnableRetries(3, 5*time.Second)

	cs := CloudantSnap{
		appConfig: appConfig,
		service:   service,
	}

	return &cs, nil
}

// Run fetches the Cloudant changes feed for the specified database,
// writing each document to a temporary file. When complete, the temp
// file is renamed to its final filename and a {db}-meta.json file is
// created, so that the next time the script is run, it starts off from
// where it left off.
func (cs *CloudantSnap) Run() error {

	// keep a note of the last sequence token
	var since string

	// start a new MetaData record and load any old meta data to get
	// a previous value of since
	meta := NewMetaData(cs.appConfig.DatabaseName)
	sanitisedFilename := sanitiseDatabaseName(cs.appConfig.DatabaseName)
	metaFilename := fmt.Sprintf("%v-meta.json", sanitisedFilename)
	meta.LoadPreviousFile(metaFilename)
	since = meta.Since

	// open the output file - a temporary file at first
	filename := generateFilename(sanitisedFilename)
	tmpFilename := "_tmp_" + filename
	fmt.Printf("spooling changes for %v since %v\n", cs.appConfig.DatabaseName, meta.GetTruncatedSince())
	fmt.Println(filename)
	f, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// build up the request parameters, including "since" if we know it from a
	// previous run
	postChangesOptions := cs.service.NewPostChangesOptions(cs.appConfig.DatabaseName)
	if len(meta.Since) > 0 {
		postChangesOptions.SetSince(meta.Since)
	} else {
		postChangesOptions.SetSince("0")
	}
	postChangesOptions.SetIncludeDocs(true)

	// create a new changes follower
	follower, err := features.NewChangesFollower(cs.service, postChangesOptions)
	if err != nil {
		return err
	}

	// start the follower, in one-off mode
	changesCh, err := follower.StartOneOff()
	if err != nil {
		return err
	}

	// run through each change
	for changesItem := range changesCh {
		// changes item returns an error on failed requests
		item, err := changesItem.Item()
		if err != nil {
			continue
		}

		// output as JSON
		item.Doc.Rev = nil
		outputStr, err := json.Marshal(item.Doc)
		if err != nil {
			continue
		}
		_, err = f.WriteString(string(outputStr) + "\n")
		if err != nil {
			continue
		}

		// record the last sequence
		since = *item.Seq
	}

	// copy tmp file to final file
	err = os.Rename(tmpFilename, filename)
	if err != nil {
		return err
	}

	// mark the end of the snapshot
	meta.RecordEnd(since)
	meta.WriteToFile(metaFilename)
	fmt.Println(metaFilename)

	return nil
}

// generateFilename creates a filename for a given database by sanitising
// the databaseName and combining it with a timestamp
func generateFilename(databaseName string) string {
	t := time.Now()
	timestamp := t.Format(time.RFC3339)
	return fmt.Sprintf("%v-snapshot-%v.jsonl", databaseName, timestamp)
}

// sanitiseDatabaseName sanitises the database name by replacing whitespace for underscores
func sanitiseDatabaseName(databaseName string) string {
	return strings.ReplaceAll(databaseName, " ", "_")
}
