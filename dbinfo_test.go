package dbinfo

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestGetDBInfo(t *testing.T) {
	// Get connection string from environment variable
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("Skipping test: TEST_POSTGRES_DSN environment variable not set")
	}

	ctx := context.Background()

	// Create connection pool
	pool, err := FromString(ctx, dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Get database info
	dbInfo, err := GetDBInfo(ctx, pool)
	if err != nil {
		t.Fatalf("Failed to get database info: %v", err)
	}

	// Basic validation
	if dbInfo == nil {
		t.Fatal("DBInfo is nil")
	}

	if dbInfo.Name == "" {
		t.Error("Database name is empty")
	}

	// Output basic database info
	t.Logf("Database: %s", dbInfo.Name)
	t.Logf("Number of tables: %d", len(dbInfo.Tables))

	// Test if we have the expected number of tables
	if len(dbInfo.Tables) < 5 {
		t.Errorf("Expected at least 5 tables, got %d", len(dbInfo.Tables))
	}

	// Create map for easier lookup
	tableMap := make(map[string]*Table)
	for _, table := range dbInfo.Tables {
		tableMap[table.Name] = table
	}

	// Test specific tables
	testCategoriesTable(t, tableMap)
	testProductsTable(t, tableMap)
	testOrderItemsTable(t, tableMap)

	// Test foreign keys
	testForeignKeys(t, tableMap)

	// Test indexes
	testIndexes(t, tableMap)

	// Test relationships
	testRelationships(t, tableMap)
}

func testCategoriesTable(t *testing.T, tableMap map[string]*Table) {
	t.Run("Categories Table", func(t *testing.T) {
		table, ok := tableMap["categories"]
		if !ok {
			t.Fatal("Categories table not found")
		}

		// Check table comment
		if table.Comment != "Product categories" {
			t.Errorf("Expected table comment 'Product categories', got %q", table.Comment)
		}

		// Check columns
		if len(table.Columns) != 4 {
			t.Errorf("Expected 4 columns in categories table, got %d", len(table.Columns))
		}

		// Create column map for lookup
		columnMap := make(map[string]*Column)
		for _, col := range table.Columns {
			columnMap[col.Name] = col
		}

		// Test specific columns
		idCol, ok := columnMap["id"]
		if !ok {
			t.Fatal("id column not found in categories table")
		}
		if !idCol.IsPrimaryKey {
			t.Error("id column should be a primary key")
		}

		nameCol, ok := columnMap["name"]
		if !ok {
			t.Fatal("name column not found in categories table")
		}
		if nameCol.IsNullable {
			t.Error("name column should not be nullable")
		}
		if nameCol.Comment != "Category name" {
			t.Errorf("Expected name column comment 'Category name', got %q", nameCol.Comment)
		}

		descCol, ok := columnMap["description"]
		if !ok {
			t.Fatal("description column not found in categories table")
		}
		if !descCol.IsNullable {
			t.Error("description column should be nullable")
		}
	})
}

func testProductsTable(t *testing.T, tableMap map[string]*Table) {
	t.Run("Products Table", func(t *testing.T) {
		table, ok := tableMap["products"]
		if !ok {
			t.Fatal("Products table not found")
		}

		// Check table comment
		if table.Comment != "Available products" {
			t.Errorf("Expected table comment 'Available products', got %q", table.Comment)
		}

		// Check columns
		if len(table.Columns) != 10 {
			t.Errorf("Expected 10 columns in products table, got %d", len(table.Columns))
		}

		// Check indexes (excluding primary key)
		if len(table.Indexes) < 3 {
			t.Errorf("Expected at least 3 indexes in products table, got %d", len(table.Indexes))
		}

		// Check for specific indexes
		var foundCategoryIdx, foundNameIdx, foundSkuIdx bool
		for _, idx := range table.Indexes {
			switch idx.Name {
			case "idx_products_category":
				foundCategoryIdx = true
				if len(idx.Columns) != 1 || idx.Columns[0] != "category_id" {
					t.Errorf("Unexpected columns for idx_products_category: %v", idx.Columns)
				}
			case "idx_products_name":
				foundNameIdx = true
				if len(idx.Columns) != 1 || idx.Columns[0] != "name" {
					t.Errorf("Unexpected columns for idx_products_name: %v", idx.Columns)
				}
			case "idx_products_sku":
				foundSkuIdx = true
				if len(idx.Columns) != 1 || idx.Columns[0] != "sku" {
					t.Errorf("Unexpected columns for idx_products_sku: %v", idx.Columns)
				}
				if !idx.Unique {
					t.Error("idx_products_sku should be unique")
				}
			}
		}

		if !foundCategoryIdx {
			t.Error("idx_products_category not found")
		}
		if !foundNameIdx {
			t.Error("idx_products_name not found")
		}
		if !foundSkuIdx {
			t.Error("idx_products_sku not found")
		}
	})
}

func testOrderItemsTable(t *testing.T, tableMap map[string]*Table) {
	t.Run("Order Items Table", func(t *testing.T) {
		table, ok := tableMap["order_items"]
		if !ok {
			t.Fatal("Order_items table not found")
		}

		// Check table comment
		if table.Comment != "Individual items within an order" {
			t.Errorf("Expected table comment 'Individual items within an order', got %q", table.Comment)
		}

		// Create column map for lookup
		columnMap := make(map[string]*Column)
		for _, col := range table.Columns {
			columnMap[col.Name] = col
		}

		// Test specific columns
		idCol, ok := columnMap["id"]
		if !ok {
			t.Fatal("id column not found in order_items table")
		}
		if !idCol.IsPrimaryKey {
			t.Error("id column should be a primary key")
		}

		orderIdCol, ok := columnMap["order_id"]
		if !ok {
			t.Fatal("order_id column not found in order_items table")
		}
		if orderIdCol.IsNullable {
			t.Error("order_id column should not be nullable")
		}

		productIdCol, ok := columnMap["product_id"]
		if !ok {
			t.Fatal("product_id column not found in order_items table")
		}
		if productIdCol.IsNullable {
			t.Error("product_id column should not be nullable")
		}
	})
}

func testForeignKeys(t *testing.T, tableMap map[string]*Table) {
	t.Run("Foreign Keys", func(t *testing.T) {
		// Test products foreign keys
		productsTable, ok := tableMap["products"]
		if !ok {
			t.Fatal("Products table not found")
		}

		if len(productsTable.ForeignKeys) < 1 {
			t.Errorf("Expected at least 1 foreign key in products table, got %d", len(productsTable.ForeignKeys))
		} else {
			categoryFk := productsTable.ForeignKeys[0]
			if categoryFk.RefTableName != "categories" {
				t.Errorf("Expected foreign key reference to categories, got %s", categoryFk.RefTableName)
			}
			if len(categoryFk.ColumnNames) != 1 || categoryFk.ColumnNames[0] != "category_id" {
				t.Errorf("Unexpected column name for category foreign key: %v", categoryFk.ColumnNames)
			}
			if categoryFk.OnDelete != "CASCADE" {
				t.Errorf("Expected ON DELETE CASCADE, got %s", categoryFk.OnDelete)
			}
		}

		// Test order_items foreign keys
		orderItemsTable, ok := tableMap["order_items"]
		if !ok {
			t.Fatal("Order_items table not found")
		}

		if len(orderItemsTable.ForeignKeys) < 2 {
			t.Errorf("Expected at least 2 foreign keys in order_items table, got %d", len(orderItemsTable.ForeignKeys))
		} else {
			// Check if we have both expected foreign keys
			var foundOrderFk, foundProductFk bool
			for _, fk := range orderItemsTable.ForeignKeys {
				if fk.RefTableName == "orders" {
					foundOrderFk = true
					if len(fk.ColumnNames) != 1 || fk.ColumnNames[0] != "order_id" {
						t.Errorf("Unexpected column name for order foreign key: %v", fk.ColumnNames)
					}
					if fk.OnDelete != "CASCADE" {
						t.Errorf("Expected ON DELETE CASCADE for order_id FK, got %s", fk.OnDelete)
					}
				}
				if fk.RefTableName == "products" {
					foundProductFk = true
					if len(fk.ColumnNames) != 1 || fk.ColumnNames[0] != "product_id" {
						t.Errorf("Unexpected column name for product foreign key: %v", fk.ColumnNames)
					}
					if fk.OnDelete != "RESTRICT" {
						t.Errorf("Expected ON DELETE RESTRICT for product_id FK, got %s", fk.OnDelete)
					}
				}
			}

			if !foundOrderFk {
				t.Error("Foreign key to orders table not found")
			}
			if !foundProductFk {
				t.Error("Foreign key to products table not found")
			}
		}
	})
}

func testIndexes(t *testing.T, tableMap map[string]*Table) {
	t.Run("Indexes", func(t *testing.T) {
		// Test orders indexes
		ordersTable, ok := tableMap["orders"]
		if !ok {
			t.Fatal("Orders table not found")
		}

		if len(ordersTable.Indexes) < 2 {
			t.Errorf("Expected at least 2 indexes in orders table, got %d", len(ordersTable.Indexes))
		} else {
			// Check if we have both expected indexes
			var foundCustomerIdx, foundDateIdx bool
			for _, idx := range ordersTable.Indexes {
				if idx.Name == "idx_orders_customer_id" {
					foundCustomerIdx = true
					if len(idx.Columns) != 1 || idx.Columns[0] != "customer_id" {
						t.Errorf("Unexpected columns for idx_orders_customer_id: %v", idx.Columns)
					}
				}
				if idx.Name == "idx_orders_date" {
					foundDateIdx = true
					if len(idx.Columns) != 1 || idx.Columns[0] != "order_date" {
						t.Errorf("Unexpected columns for idx_orders_date: %v", idx.Columns)
					}
				}
			}

			if !foundCustomerIdx {
				t.Error("idx_orders_customer_id not found")
			}
			if !foundDateIdx {
				t.Error("idx_orders_date not found")
			}
		}

		// Test order_items indexes
		orderItemsTable, ok := tableMap["order_items"]
		if !ok {
			t.Fatal("Order_items table not found")
		}

		if len(orderItemsTable.Indexes) < 2 {
			t.Errorf("Expected at least 2 indexes in order_items table, got %d", len(orderItemsTable.Indexes))
		} else {
			// Check if we have both expected indexes
			var foundOrderIdIdx, foundProductIdIdx bool
			for _, idx := range orderItemsTable.Indexes {
				if idx.Name == "idx_order_items_order_id" {
					foundOrderIdIdx = true
					if len(idx.Columns) != 1 || idx.Columns[0] != "order_id" {
						t.Errorf("Unexpected columns for idx_order_items_order_id: %v", idx.Columns)
					}
				}
				if idx.Name == "idx_order_items_product_id" {
					foundProductIdIdx = true
					if len(idx.Columns) != 1 || idx.Columns[0] != "product_id" {
						t.Errorf("Unexpected columns for idx_order_items_product_id: %v", idx.Columns)
					}
				}
			}

			if !foundOrderIdIdx {
				t.Error("idx_order_items_order_id not found")
			}
			if !foundProductIdIdx {
				t.Error("idx_order_items_product_id not found")
			}
		}
	})
}

func testRelationships(t *testing.T, tableMap map[string]*Table) {
	t.Run("Table Relationships", func(t *testing.T) {
		// Test HasMany relationships
		t.Run("HasMany Relationships", func(t *testing.T) {
			// Test categories HasMany products
			categoriesTable, ok := tableMap["categories"]
			if !ok {
				t.Fatal("Categories table not found")
			}

			if len(categoriesTable.HasMany) == 0 {
				t.Fatal("Expected categories to have HasMany relationships")
			}

			var foundProductsRel bool
			for _, rel := range categoriesTable.HasMany {
				if rel.Table == "products" {
					foundProductsRel = true
					if rel.ForeignKey == "" {
						t.Error("Foreign key name is empty in HasMany relationship")
					}
					if len(rel.Columns) != 1 || rel.Columns[0] != "id" {
						t.Errorf("Unexpected local columns in HasMany relationship: %v", rel.Columns)
					}
					if len(rel.References) != 1 || rel.References[0] != "category_id" {
						t.Errorf("Unexpected reference columns in HasMany relationship: %v", rel.References)
					}
				}
			}

			if !foundProductsRel {
				t.Error("Expected categories to have HasMany relationship with products")
			}

			// Test orders HasMany order_items
			ordersTable, ok := tableMap["orders"]
			if !ok {
				t.Fatal("Orders table not found")
			}

			if len(ordersTable.HasMany) == 0 {
				t.Fatal("Expected orders to have HasMany relationships")
			}

			var foundOrderItemsRel bool
			for _, rel := range ordersTable.HasMany {
				if rel.Table == "order_items" {
					foundOrderItemsRel = true
					if len(rel.Columns) != 1 || rel.Columns[0] != "id" {
						t.Errorf("Unexpected local columns in HasMany relationship: %v", rel.Columns)
					}
					if len(rel.References) != 1 || rel.References[0] != "order_id" {
						t.Errorf("Unexpected reference columns in HasMany relationship: %v", rel.References)
					}
				}
			}

			if !foundOrderItemsRel {
				t.Error("Expected orders to have HasMany relationship with order_items")
			}
		})

		// Test BelongsTo relationships
		t.Run("BelongsTo Relationships", func(t *testing.T) {
			// Test products BelongsTo categories
			productsTable, ok := tableMap["products"]
			if !ok {
				t.Fatal("Products table not found")
			}

			if len(productsTable.BelongsTo) == 0 {
				t.Fatal("Expected products to have BelongsTo relationships")
			}

			var foundCategoriesRel bool
			for _, rel := range productsTable.BelongsTo {
				if rel.Table == "categories" {
					foundCategoriesRel = true
					if rel.ForeignKey == "" {
						t.Error("Foreign key name is empty in BelongsTo relationship")
					}
					if len(rel.Columns) != 1 || rel.Columns[0] != "category_id" {
						t.Errorf("Unexpected local columns in BelongsTo relationship: %v", rel.Columns)
					}
					if len(rel.References) != 1 || rel.References[0] != "id" {
						t.Errorf("Unexpected reference columns in BelongsTo relationship: %v", rel.References)
					}
				}
			}

			if !foundCategoriesRel {
				t.Error("Expected products to have BelongsTo relationship with categories")
			}

			// Test order_items BelongsTo relationships
			orderItemsTable, ok := tableMap["order_items"]
			if !ok {
				t.Fatal("Order_items table not found")
			}

			if len(orderItemsTable.BelongsTo) < 2 {
				t.Fatalf("Expected order_items to have at least 2 BelongsTo relationships, got %d", len(orderItemsTable.BelongsTo))
			}

			var foundOrdersRel, foundProductsRel bool
			for _, rel := range orderItemsTable.BelongsTo {
				if rel.Table == "orders" {
					foundOrdersRel = true
					if len(rel.Columns) != 1 || rel.Columns[0] != "order_id" {
						t.Errorf("Unexpected local columns in BelongsTo relationship: %v", rel.Columns)
					}
					if len(rel.References) != 1 || rel.References[0] != "id" {
						t.Errorf("Unexpected reference columns in BelongsTo relationship: %v", rel.References)
					}
				}
				if rel.Table == "products" {
					foundProductsRel = true
					if len(rel.Columns) != 1 || rel.Columns[0] != "product_id" {
						t.Errorf("Unexpected local columns in BelongsTo relationship: %v", rel.Columns)
					}
					if len(rel.References) != 1 || rel.References[0] != "id" {
						t.Errorf("Unexpected reference columns in BelongsTo relationship: %v", rel.References)
					}
				}
			}

			if !foundOrdersRel {
				t.Error("Expected order_items to have BelongsTo relationship with orders")
			}
			if !foundProductsRel {
				t.Error("Expected order_items to have BelongsTo relationship with products")
			}
		})
	})
}

// TestGetDBInfoStructure uses go-cmp to compare the output structure with expected structure
func TestGetDBInfoStructure(t *testing.T) {
	// Get connection string from environment variable
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("Skipping test: TEST_POSTGRES_DSN environment variable not set")
	}

	ctx := context.Background()

	// Create connection pool
	pool, err := FromString(ctx, dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Get actual database info
	actual, err := GetDBInfo(ctx, pool)
	if err != nil {
		t.Fatalf("Failed to get database info: %v", err)
	}

	// Define expected structure (focusing on core relationships)
	// Note: We define only a subset of the tables and relationships for comparison
	expected := &DBInfo{
		// Name not checked as it depends on the test database name
		Tables: []*Table{
			{
				Name:   "categories",
				Schema: "public",
				HasMany: []*Relationship{
					{
						Table:      "products",
						Schema:     "public",
						Columns:    []string{"id"},
						References: []string{"category_id"},
						OnDelete:   "CASCADE",
					},
				},
				BelongsTo: []*Relationship{},
			},
			{
				Name:   "products",
				Schema: "public",
				HasMany: []*Relationship{
					{
						Table:      "order_items",
						Schema:     "public",
						Columns:    []string{"id"},
						References: []string{"product_id"},
						OnDelete:   "RESTRICT",
					},
				},
				BelongsTo: []*Relationship{
					{
						Table:      "categories",
						Schema:     "public",
						Columns:    []string{"category_id"},
						References: []string{"id"},
						OnDelete:   "CASCADE",
					},
				},
			},
			{
				Name:   "customers",
				Schema: "public",
				HasMany: []*Relationship{
					{
						Table:      "orders",
						Schema:     "public",
						Columns:    []string{"id"},
						References: []string{"customer_id"},
						OnDelete:   "NO ACTION",
					},
				},
				BelongsTo: []*Relationship{},
			},
			{
				Name:   "orders",
				Schema: "public",
				HasMany: []*Relationship{
					{
						Table:      "order_items",
						Schema:     "public",
						Columns:    []string{"id"},
						References: []string{"order_id"},
						OnDelete:   "CASCADE",
					},
				},
				BelongsTo: []*Relationship{
					{
						Table:      "customers",
						Schema:     "public",
						Columns:    []string{"customer_id"},
						References: []string{"id"},
						OnDelete:   "NO ACTION",
					},
				},
			},
			{
				Name:    "order_items",
				Schema:  "public",
				HasMany: []*Relationship{}, // Empty slice, not nil
				BelongsTo: []*Relationship{
					{
						Table:      "orders",
						Schema:     "public",
						Columns:    []string{"order_id"},
						References: []string{"id"},
						OnDelete:   "CASCADE",
					},
					{
						Table:      "products",
						Schema:     "public",
						Columns:    []string{"product_id"},
						References: []string{"id"},
						OnDelete:   "RESTRICT",
					},
				},
			},
		},
	}

	// Ensure all expected tables have initialized slices
	for _, table := range expected.Tables {
		if table.HasMany == nil {
			table.HasMany = []*Relationship{}
		}
		if table.BelongsTo == nil {
			table.BelongsTo = []*Relationship{}
		}
	}

	// Ensure all actual tables have initialized slices (not nil)
	for _, table := range actual.Tables {
		if table.HasMany == nil {
			table.HasMany = []*Relationship{}
		}
		if table.BelongsTo == nil {
			table.BelongsTo = []*Relationship{}
		}
	}

	// Options for comparison
	opts := []cmp.Option{
		// Ignore fields that can vary or aren't relevant for structure comparison
		cmpopts.IgnoreFields(DBInfo{}, "Name"),
		cmpopts.IgnoreFields(Table{}, "Columns", "Indexes", "ForeignKeys", "Comment"),
		cmpopts.IgnoreFields(Relationship{}, "ForeignKey", "OnUpdate"),

		// Only compare the tables we've defined in our expected structure
		cmpopts.IgnoreSliceElements(func(t *Table) bool {
			for _, expectedTable := range expected.Tables {
				if t.Name == expectedTable.Name && t.Schema == expectedTable.Schema {
					return false
				}
			}
			return true
		}),

		// Sort slices for comparison
		cmpopts.SortSlices(func(a, b *Table) bool {
			return a.Name < b.Name
		}),
		cmpopts.SortSlices(func(a, b *Relationship) bool {
			if a.Table != b.Table {
				return a.Table < b.Table
			}
			if len(a.Columns) == 0 || len(b.Columns) == 0 {
				return false
			}
			return a.Columns[0] < b.Columns[0]
		}),
	}

	// Compare actual vs expected
	if diff := cmp.Diff(expected, actual, opts...); diff != "" {
		t.Errorf("Unexpected database structure (-expected +actual):\n%s", diff)
	}
}
