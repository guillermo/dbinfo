// Package dbinfo provides functionality to analyze PostgreSQL database schemas
// and extract information about tables, columns, indexes, foreign keys, and relationships.
package dbinfo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBQuerier is an interface that can be satisfied by both pgxpool.Pool and pgx.Conn
type DBQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// FromString creates a new connection pool from a PostgreSQL connection string.
// It accepts both URL format (postgresql://user:password@host:port/database)
// and DSN format (host=localhost port=5432 dbname=mydb user=myuser password=mypass).
// The caller is responsible for closing the pool when done.
func FromString(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return pool, nil
}

// DBInfo represents the structure of a database
type DBInfo struct {
	Name   string
	Tables []*Table
}

// Relationship represents a relationship between tables
type Relationship struct {
	Table      string   // The related table name
	Schema     string   // The related table schema
	ForeignKey string   // The name of the foreign key constraint
	Columns    []string // Local columns in the relationship
	References []string // Referenced columns in the relationship
	OnUpdate   string   // ON UPDATE action
	OnDelete   string   // ON DELETE action
}

// Table represents a database table
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

// Column represents a table column
type Column struct {
	Name         string
	Type         string
	IsNullable   bool
	DefaultValue string
	Comment      string
	IsPrimaryKey bool
}

// Index represents a table index
type Index struct {
	Name       string
	Unique     bool
	Columns    []string
	Expression string
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name           string
	ColumnNames    []string
	RefTableSchema string
	RefTableName   string
	RefColumnNames []string
	OnUpdate       string
	OnDelete       string
}

// GetDBInfo analyzes a PostgreSQL database and returns its structure
// using a provided DBQuerier (e.g., *pgxpool.Pool or *pgx.Conn)
func GetDBInfo(ctx context.Context, db DBQuerier) (*DBInfo, error) {
	// Get database name
	var dbName string
	err := db.QueryRow(ctx, "SELECT current_database()").Scan(&dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to get database name: %w", err)
	}

	dbInfo := &DBInfo{
		Name: dbName,
	}

	// Get all tables
	tables, err := getTables(ctx, db)
	if err != nil {
		return nil, err
	}
	dbInfo.Tables = tables

	// Build table relationships
	buildRelationships(dbInfo.Tables)

	return dbInfo, nil
}

// buildRelationships builds the HasMany and BelongsTo relationships between tables
func buildRelationships(tables []*Table) {
	// Create a map for faster table lookup by schema and name
	tableMap := make(map[string]*Table)
	for _, table := range tables {
		key := table.Schema + "." + table.Name
		tableMap[key] = table

		// Initialize relationship slices as empty, not nil
		if table.HasMany == nil {
			table.HasMany = make([]*Relationship, 0)
		}
		if table.BelongsTo == nil {
			table.BelongsTo = make([]*Relationship, 0)
		}
	}

	// Process each table's foreign keys to build relationships
	for _, table := range tables {
		// Process each foreign key
		for _, fk := range table.ForeignKeys {
			// Create a BelongsTo relationship for this table
			belongsTo := &Relationship{
				Table:      fk.RefTableName,
				Schema:     fk.RefTableSchema,
				ForeignKey: fk.Name,
				Columns:    fk.ColumnNames,
				References: fk.RefColumnNames,
				OnUpdate:   fk.OnUpdate,
				OnDelete:   fk.OnDelete,
			}
			table.BelongsTo = append(table.BelongsTo, belongsTo)

			// Add a HasMany relationship to the referenced table
			refTableKey := fk.RefTableSchema + "." + fk.RefTableName
			if refTable, ok := tableMap[refTableKey]; ok {
				hasMany := &Relationship{
					Table:      table.Name,
					Schema:     table.Schema,
					ForeignKey: fk.Name,
					Columns:    fk.RefColumnNames,
					References: fk.ColumnNames,
					OnUpdate:   fk.OnUpdate,
					OnDelete:   fk.OnDelete,
				}
				refTable.HasMany = append(refTable.HasMany, hasMany)
			}
		}
	}
}

// getTables retrieves all tables from the database
func getTables(ctx context.Context, db DBQuerier) ([]*Table, error) {
	// Query to get all tables in the database
	query := `
	SELECT t.table_schema, t.table_name, obj_description(pg_class.oid) as table_comment
	FROM information_schema.tables t
	JOIN pg_class ON pg_class.relname = t.table_name
	JOIN pg_namespace ON pg_namespace.oid = pg_class.relnamespace AND pg_namespace.nspname = t.table_schema
	WHERE t.table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
	AND t.table_type = 'BASE TABLE'
	ORDER BY t.table_schema, t.table_name`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []*Table
	for rows.Next() {
		table := &Table{}
		var comment *string // Use a pointer to handle NULL
		err := rows.Scan(&table.Schema, &table.Name, &comment)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}

		// Set empty string if comment is NULL
		if comment != nil {
			table.Comment = *comment
		}

		// Get columns for this table
		columns, err := getColumns(ctx, db, table.Schema, table.Name)
		if err != nil {
			return nil, err
		}
		table.Columns = columns

		// Get indexes for this table
		indexes, err := getIndexes(ctx, db, table.Schema, table.Name)
		if err != nil {
			return nil, err
		}
		table.Indexes = indexes

		// Get foreign keys for this table
		foreignKeys, err := getForeignKeys(ctx, db, table.Schema, table.Name)
		if err != nil {
			return nil, err
		}
		table.ForeignKeys = foreignKeys

		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return tables, nil
}

// getColumns retrieves all columns for a given table
func getColumns(ctx context.Context, db DBQuerier, schema, tableName string) ([]*Column, error) {
	// Query to get columns
	query := `
	SELECT c.column_name, c.data_type,
	       CASE WHEN c.is_nullable = 'YES' THEN TRUE ELSE FALSE END as is_nullable,
	       c.column_default,
	       pg_catalog.col_description(format('%s.%s', c.table_schema, c.table_name)::regclass::oid, c.ordinal_position) as column_comment,
	       CASE WHEN pk.column_name IS NOT NULL THEN TRUE ELSE FALSE END as is_primary_key
	FROM information_schema.columns c
	LEFT JOIN (
	    SELECT kcu.column_name
	    FROM information_schema.table_constraints tc
	    JOIN information_schema.key_column_usage kcu ON kcu.constraint_name = tc.constraint_name
	        AND kcu.table_schema = tc.table_schema
	        AND kcu.table_name = tc.table_name
	    WHERE tc.constraint_type = 'PRIMARY KEY'
	        AND tc.table_schema = $1
	        AND tc.table_name = $2
	) pk ON pk.column_name = c.column_name
	WHERE c.table_schema = $1
	  AND c.table_name = $2
	ORDER BY c.ordinal_position`

	rows, err := db.Query(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for %s.%s: %w", schema, tableName, err)
	}
	defer rows.Close()

	var columns []*Column
	for rows.Next() {
		column := &Column{}
		var comment *string      // Use a pointer to handle NULL
		var defaultValue *string // Use a pointer to handle NULL default values

		err := rows.Scan(
			&column.Name,
			&column.Type,
			&column.IsNullable,
			&defaultValue,
			&comment,
			&column.IsPrimaryKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column row: %w", err)
		}

		// Set empty string if comment is NULL
		if comment != nil {
			column.Comment = *comment
		}

		// Set empty string if default value is NULL
		if defaultValue != nil {
			column.DefaultValue = *defaultValue
		}

		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating column rows: %w", err)
	}

	return columns, nil
}

// getIndexes retrieves all indexes for a given table
func getIndexes(ctx context.Context, db DBQuerier, schema, tableName string) ([]*Index, error) {
	// Query to get indexes
	query := `
	SELECT
	    i.relname as index_name,
	    CASE WHEN ix.indisunique THEN TRUE ELSE FALSE END as is_unique,
	    array_remove(array_agg(a.attname), NULL) as column_names,
	    pg_get_expr(ix.indexprs, ix.indrelid) as expression
	FROM
	    pg_index ix
	    JOIN pg_class i ON i.oid = ix.indexrelid
	    JOIN pg_class t ON t.oid = ix.indrelid
	    JOIN pg_namespace n ON n.oid = t.relnamespace
	    LEFT JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
	WHERE
	    n.nspname = $1
	    AND t.relname = $2
	    AND ix.indisprimary = false
	GROUP BY
	    i.relname, ix.indisunique, ix.indexprs, ix.indrelid
	ORDER BY
	    i.relname`

	rows, err := db.Query(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes for %s.%s: %w", schema, tableName, err)
	}
	defer rows.Close()

	var indexes []*Index
	for rows.Next() {
		index := &Index{}
		var columnNames []string
		var expression *string // Use a pointer to handle NULL

		err := rows.Scan(
			&index.Name,
			&index.Unique,
			&columnNames,
			&expression,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index row: %w", err)
		}

		// Set empty string if expression is NULL
		if expression != nil {
			index.Expression = *expression
		}

		index.Columns = columnNames
		indexes = append(indexes, index)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating index rows: %w", err)
	}

	return indexes, nil
}

// getForeignKeys retrieves all foreign keys for a given table
func getForeignKeys(ctx context.Context, db DBQuerier, schema, tableName string) ([]*ForeignKey, error) {
	// Query to get foreign keys
	query := `
	SELECT
	    tc.constraint_name,
	    array_remove(array_agg(kcu.column_name), NULL) as column_names,
	    ccu.table_schema as foreign_table_schema,
	    ccu.table_name as foreign_table_name,
	    array_remove(array_agg(ccu.column_name), NULL) as foreign_column_names,
	    rc.update_rule,
	    rc.delete_rule
	FROM
	    information_schema.table_constraints tc
	    JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
	    JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name
	    JOIN information_schema.referential_constraints rc ON rc.constraint_name = tc.constraint_name
	WHERE
	    tc.constraint_type = 'FOREIGN KEY'
	    AND tc.table_schema = $1
	    AND tc.table_name = $2
	GROUP BY
	    tc.constraint_name,
	    ccu.table_schema,
	    ccu.table_name,
	    rc.update_rule,
	    rc.delete_rule
	ORDER BY
	    tc.constraint_name`

	rows, err := db.Query(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys for %s.%s: %w", schema, tableName, err)
	}
	defer rows.Close()

	var foreignKeys []*ForeignKey
	for rows.Next() {
		fk := &ForeignKey{}
		var columnNames []string
		var refColumnNames []string
		err := rows.Scan(
			&fk.Name,
			&columnNames,
			&fk.RefTableSchema,
			&fk.RefTableName,
			&refColumnNames,
			&fk.OnUpdate,
			&fk.OnDelete,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan foreign key row: %w", err)
		}
		fk.ColumnNames = columnNames
		fk.RefColumnNames = refColumnNames
		foreignKeys = append(foreignKeys, fk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating foreign key rows: %w", err)
	}

	return foreignKeys, nil
}
