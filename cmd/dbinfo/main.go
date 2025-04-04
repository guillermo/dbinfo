package main

import (
	"fmt"
	"os"

	"github.com/guillermo/dbinfo"
	"gopkg.in/yaml.v3"
)

func main() {
	// Get connection string from environment or command line
	dsn := os.Getenv("DATABASE_URL")
	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}

	if dsn == "" {
		fmt.Println("Error: No database connection string provided")
		fmt.Println("Usage: dbinfo [connection_string]")
		fmt.Println("  or set the DATABASE_URL environment variable")
		os.Exit(1)
	}

	// Get database schema information
	info, err := dbinfo.GetDBInfo(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}

	// Convert to YAML and print to stdout
	yamlData, err := yaml.Marshal(info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(yamlData))
}
