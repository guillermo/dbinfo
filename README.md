# DBInfo - Just give me the postgres schema

[![GoDock](https://godoc.org/github.com/guillermo/dbinfo?status.svg)](https://pkg.go.dev/github.com/guillermo/dbinfo)
[![Go Report Card](https://goreportcard.com/badge/github.com/guillermo/dbinfo)](https://goreportcard.com/report/github.com/guillermo/dbinfo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/guillermo/dbinfo/actions/workflows/go.yml/badge.svg)](https://github.com/guillermo/dbinfo/actions/workflows/go.yml)

![DBInfo Schema](dbinfo.png)



A Go package and a command line command that analyzes PostgreSQL databases and returns the schema as Go structures or as a yaml.

## Usage

### As a library

```go
package main

import (
	"fmt"
	"log"

	"github.com/guillermo/dbinfo"
)

func main() {
	// PostgreSQL connection string
	dsn := "postgres://username:password@localhost:5432/mydatabase"

	// Get database info
	info, err := dbinfo.GetDBInfo(dsn)
	if err != nil {
		log.Fatalf("Error getting DB info: %v", err)
	}

	// Print database name
	fmt.Printf("Database: %s\n", info.Name)

	// Iterate over tables
	for _, table := range info.Tables {
		fmt.Printf("\nTable: %s.%s\n", table.Schema, table.Name)

		// Print columns
		fmt.Println("Columns:")
		for _, col := range table.Columns {
			nullable := "NOT NULL"
			if col.IsNullable {
				nullable = "NULL"
			}

			primaryKey := ""
			if col.IsPrimaryKey {
				primaryKey = " (PRIMARY KEY)"
			}

			fmt.Printf("  - %s: %s %s%s\n", col.Name, col.Type, nullable, primaryKey)
		}

		// Print indexes
		if len(table.Indexes) > 0 {
			fmt.Println("Indexes:")
			for _, idx := range table.Indexes {
				unique := ""
				if idx.Unique {
					unique = " UNIQUE"
				}
				fmt.Printf("  - %s:%s %v\n", idx.Name, unique, idx.Columns)
			}
		}

		// Print foreign keys
		if len(table.ForeignKeys) > 0 {
			fmt.Println("Foreign Keys:")
			for _, fk := range table.ForeignKeys {
				fmt.Printf("  - %s: %v -> %s.%s %v\n",
					fk.Name, fk.ColumnNames, fk.RefTableSchema, fk.RefTableName, fk.RefColumnNames)
			}
		}

		// Print relationships
		if len(table.HasMany) > 0 {
			fmt.Println("Has Many:")
			for _, rel := range table.HasMany {
				fmt.Printf("  - %s.%s (%s.%s -> %s.%s)\n",
					rel.Schema, rel.Table,
					table.Name, rel.Columns[0],
					rel.Table, rel.References[0])
			}
		}

		if len(table.BelongsTo) > 0 {
			fmt.Println("Belongs To:")
			for _, rel := range table.BelongsTo {
				fmt.Printf("  - %s.%s (%s.%s -> %s.%s)\n",
					rel.Schema, rel.Table,
					table.Name, rel.Columns[0],
					rel.Table, rel.References[0])
			}
		}
	}
}
```

### As a command-line tool

DBInfo also comes with a command-line tool that can dump database schema as YAML.

#### Installation

```bash
# Install using go install
go install github.com/guillermo/dbinfo/cmd/dbinfo@latest
```
#### Usage

```bash
# Using environment variable
export DATABASE_URL="postgres://username:password@localhost:5432/mydatabase"
dbinfo

# Or pass connection string as argument
dbinfo "postgres://username:password@localhost:5432/mydatabase"
```

The command outputs a YAML representation of the database structure:

```yaml
name: mydatabase
tables:
- name: categories
  schema: public
  comment: Product categories
  columns:
  - name: id
    type: integer
    isnullable: false
    isprimarykey: true
  - name: name
    type: character varying
    isnullable: false
    comment: Category name
  # ... other columns, indexes, foreign keys ...
  hasmany:
  - table: products
    schema: public
    foreignkey: products_category_id_fkey
    columns: [id]
    references: [category_id]
    onupdate: NO ACTION
    ondelete: CASCADE
  belongsto: []
# ... other tables ...
```

### Important Notes

- **PostgreSQL Only**: Currently only PostgreSQL databases are supported, using the pgx driver.
- **Table Relationships**: The package automatically identifies relationships between tables:
  - `HasMany`: Shows which tables reference this table (parent-to-child relationships)
  - `BelongsTo`: Shows which tables this table references (child-to-parent relationships)

## Returned Structures

The main structure returned by `GetDBInfo` is:

```go
type DBInfo struct {
	Name   string
	Tables []*Table
}

type Relationship struct {
	Table           string   // The related table name
	Schema          string   // The related table schema
	ForeignKey      string   // The name of the foreign key constraint
	Columns         []string // Local columns in the relationship
	References      []string // Referenced columns in the relationship
	OnUpdate        string   // ON UPDATE action
	OnDelete        string   // ON DELETE action
}

type Table struct {
	Name        string
	Schema      string
	Columns     []*Column
	Indexes     []*Index
	ForeignKeys []*ForeignKey
	HasMany     []*Relationship // Tables that reference this table
	BelongsTo   []*Relationship // Tables this table references
	Comment     string
}

type Column struct {
	Name         string
	Type         string
	IsNullable   bool
	DefaultValue string
	Comment      string
	IsPrimaryKey bool
}

type Index struct {
	Name       string
	Unique     bool
	Columns    []string
	Expression string
}

type ForeignKey struct {
	Name           string
	ColumnNames    []string
	RefTableSchema string
	RefTableName   string
	RefColumnNames []string
	OnUpdate       string
	OnDelete       string
}
```

## Testing

### Using Docker (Recommended)

This project includes a Makefile with commands to run tests against a real PostgreSQL database in Docker:

```bash
# Run the tests (starts PostgreSQL, loads test schema, runs tests, stops PostgreSQL)
make test

# Or run the individual steps:
make postgres_start    # Start a PostgreSQL container on port 9875
make postgres_load     # Load the test schema into PostgreSQL
make postgres_stop     # Stop and remove the PostgreSQL container
```

The test environment:
- Uses PostgreSQL 14 running on port 9875
- Creates a database named `dbinfo_test`
- Loads a fixture schema with tables, indexes, foreign keys, etc.

### Without Docker

If you have your own PostgreSQL instance, you can run the tests manually:

```bash
export TEST_POSTGRES_DSN="postgres://username:password@localhost:5432/mydatabase"
go test
```

## License

MIT