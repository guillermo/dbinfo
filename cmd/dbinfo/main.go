package main

import (
	"context"
	"fmt"
	"os"

	"github.com/guillermo/dbinfo"
	"gopkg.in/yaml.v3"
)

// Define structs that match the dbinfo package structs
// but with yaml tags for better YAML output

type DBInfoYAML struct {
	Name   string       `yaml:"name"`
	Tables []*TableYAML `yaml:"tables"`
}

type TableYAML struct {
	Name        string               `yaml:"name"`
	Schema      string               `yaml:"schema"`
	Columns     []*dbinfo.Column     `yaml:"columns,omitempty"`
	Indexes     []*dbinfo.Index      `yaml:"indexes,omitempty"`
	ForeignKeys []*dbinfo.ForeignKey `yaml:"foreignkeys,omitempty"`
	HasMany     []*RelationshipYAML  `yaml:"hasmany,omitempty"`
	BelongsTo   []*RelationshipYAML  `yaml:"belongsto,omitempty"`
	Comment     string               `yaml:"comment,omitempty"`
}

type RelationshipYAML struct {
	Table      string   `yaml:"table"`
	Schema     string   `yaml:"schema"`
	ForeignKey string   `yaml:"foreignkey"`
	Columns    []string `yaml:"columns"`
	References []string `yaml:"references"`
	OnUpdate   string   `yaml:"onupdate,omitempty"`
	OnDelete   string   `yaml:"ondelete,omitempty"`
}

func convertToYAML(info *dbinfo.DBInfo) *DBInfoYAML {
	yamlInfo := &DBInfoYAML{
		Name:   info.Name,
		Tables: make([]*TableYAML, len(info.Tables)),
	}

	for i, table := range info.Tables {
		yamlTable := &TableYAML{
			Name:        table.Name,
			Schema:      table.Schema,
			Columns:     table.Columns,
			Indexes:     table.Indexes,
			ForeignKeys: table.ForeignKeys,
			Comment:     table.Comment,
		}

		// Convert HasMany relationships
		if len(table.HasMany) > 0 {
			yamlTable.HasMany = make([]*RelationshipYAML, len(table.HasMany))
			for j, rel := range table.HasMany {
				yamlTable.HasMany[j] = &RelationshipYAML{
					Table:      rel.Table,
					Schema:     rel.Schema,
					ForeignKey: rel.ForeignKey,
					Columns:    rel.Columns,
					References: rel.References,
					OnUpdate:   rel.OnUpdate,
					OnDelete:   rel.OnDelete,
				}
			}
		}

		// Convert BelongsTo relationships
		if len(table.BelongsTo) > 0 {
			yamlTable.BelongsTo = make([]*RelationshipYAML, len(table.BelongsTo))
			for j, rel := range table.BelongsTo {
				yamlTable.BelongsTo[j] = &RelationshipYAML{
					Table:      rel.Table,
					Schema:     rel.Schema,
					ForeignKey: rel.ForeignKey,
					Columns:    rel.Columns,
					References: rel.References,
					OnUpdate:   rel.OnUpdate,
					OnDelete:   rel.OnDelete,
				}
			}
		}

		yamlInfo.Tables[i] = yamlTable
	}

	return yamlInfo
}

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

	ctx := context.Background()

	// Create connection pool
	pool, err := dbinfo.FromString(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Get database schema information
	info, err := dbinfo.GetDBInfo(ctx, pool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting database info: %v\n", err)
		os.Exit(1)
	}

	// Convert to our YAML-friendly structs
	yamlInfo := convertToYAML(info)

	// Convert to YAML and print to stdout
	yamlData, err := yaml.Marshal(yamlInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(yamlData))
}
