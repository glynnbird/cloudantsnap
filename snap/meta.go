package snap

import (
	"encoding/json"
	"os"
	"regexp"
	"time"
)

// MetaData records a cloudantsnap run's start and end time,
// the database name and the last sequence token. This data is
// converted to JSON and stored in the application's "meta" file.
type MetaData struct {
	Since        string    `json:"since"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	DatabaseName string    `json:"db"`
}

// NewMetaData returns a pointer to a new MetaData struct
func NewMetaData(databaseName string) *MetaData {
	md := MetaData{
		DatabaseName: databaseName,
		StartTime:    time.Now(),
	}
	return &md
}

// RecordEnd marks the end of a cloudantsnap run
func (md *MetaData) RecordEnd(since string) {
	md.Since = since
	md.EndTime = time.Now()
}

// Output generates a JSON-formatted representation of the
// run's meta data
func (md *MetaData) Output() string {
	outputStr, _ := json.Marshal(md)
	return string(outputStr)
}

// WriteToFile writes the meta data to a given file as JSON
func (md *MetaData) WriteToFile(filename string) error {
	output := md.Output()
	err := os.WriteFile(filename, []byte(output), 0644)
	return err
}

// LoadPreviousFile loads a previously saved meta data file
// and parses it as JSON, extracting the "since" value
func (md *MetaData) LoadPreviousFile(filename string) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	var data map[string]interface{}
	err = json.Unmarshal(contents, &data)
	if err != nil {
		return
	}
	md.Since = data["since"].(string)
}

// GetTruncatedSince converts the stored Since value to a shortened
// form, suitable for sending to the output
func (md *MetaData) GetTruncatedSince() string {
	if md.Since == "0" {
		return md.Since
	} else {
		re := regexp.MustCompile("-.*$")
		return re.ReplaceAllString(md.Since, "")
	}
}
