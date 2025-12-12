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

// CloudantSnap stores everything we need to proceed with a cloudantsnap
// execution.
type CloudantSnap struct {
	appConfig         *AppConfig             // our command-line options
	service           *cloudantv1.CloudantV1 // the Cloudant SDK client
	meta              *MetaData              // the meta data we need for the cloudantsnap run
	sanitisedFilename string                 // a sanitised form of the database name, safe for creating filenames
	metaFilename      string                 // the filename used to store the cloudantsnap run's meta data
	filename          string                 // the final filename containing the cloudantsnap output
	tmpFilename       string                 // the temporary filename used to store the cloudantsnap output, until it is renamed to "filename"
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

	// create a meta instance
	meta := NewMetaData(appConfig.DatabaseName)

	// create sanitised filename
	sanitisedFilename := sanitiseDatabaseName(appConfig.DatabaseName)

	// create filenames
	filename := generateFilename(sanitisedFilename)
	tmpFilename := "_tmp_" + filename

	cs := CloudantSnap{
		appConfig:         appConfig,
		service:           service,
		meta:              meta,
		sanitisedFilename: sanitisedFilename,
		metaFilename:      fmt.Sprintf("%v-meta.json", sanitisedFilename),
		filename:          filename,
		tmpFilename:       tmpFilename,
	}

	return &cs, nil
}

// Run fetches the Cloudant changes feed for the specified database,
// writing each document to a temporary file. When complete, the temp
// file is renamed to its final filename and a {db}-meta.json file is
// created, so that the next time the script is run, it starts off from
// where it left off.
func (cs *CloudantSnap) Run() error {
	var f *os.File = nil
	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	// keep a note of the last sequence token
	var since string

	// find a since token to resume from, if it exists
	cs.meta.LoadPreviousFile(cs.metaFilename)
	since = cs.meta.Since

	// open the output file - a temporary file at first
	fmt.Printf("spooling changes for %v since %v\n", cs.appConfig.DatabaseName, cs.meta.GetTruncatedSince())
	fmt.Println(cs.filename)
	f, err := os.OpenFile(cs.tmpFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	// build up the request parameters, including "since" if we know it from a
	// previous run
	postChangesOptions := cs.service.NewPostChangesOptions(cs.appConfig.DatabaseName)
	if len(cs.meta.Since) > 0 {
		postChangesOptions.SetSince(cs.meta.Since)
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
		// if this change represents a deleted document and
		// we haven't opted to include deletions, ignore it
		if item.Deleted != nil && *item.Deleted && cs.appConfig.Deletions == false {
			continue
		}
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
		if item.Seq != nil {
			since = *item.Seq
		}
	}

	// copy tmp file to final file
	f.Close()
	f = nil
	err = os.Rename(cs.tmpFilename, cs.filename)
	if err != nil {
		return err
	}

	// mark the end of the snapshot
	cs.meta.RecordEnd(since)
	err = cs.meta.WriteToFile(cs.metaFilename)
	if err != nil {
		return err
	}
	fmt.Println(cs.metaFilename)

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
