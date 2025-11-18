package snap

import (
	"errors"
	"flag"
	"fmt"
)

// AppConfig contains the application configuration collected from command-line flags
type AppConfig struct {
	DatabaseName string
	Deletions    bool
}

func (ac AppConfig) Print() {
	fmt.Println("APP CONFIG")
	fmt.Println("----------")
	fmt.Printf("DatabaseName: %v\n", ac.DatabaseName)
	fmt.Printf("Deletions: %v\n", ac.Deletions)
}

func NewAppConfig() (*AppConfig, error) {
	appConfig := AppConfig{}

	// parse command-line options
	flag.StringVar(&appConfig.DatabaseName, "dbname", "", "The Cloudant database name to write to")
	flag.StringVar(&appConfig.DatabaseName, "db", "", "The Cloudant database name to write to")
	flag.BoolVar(&appConfig.Deletions, "deletions", false, "Whether to include deleted documents in the output")
	flag.Parse()

	// if we don't have a database name after parsing
	if appConfig.DatabaseName == "" {
		return nil, errors.New("missing dbname/db")
	} else {
		return &appConfig, nil
	}
}
