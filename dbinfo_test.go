package dbinfo

import (
	"os"
	"testing"
)

func TestGetDBInfo(t *testing.T) {
	// Get connection string from environment variable
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("Skipping test: TEST_POSTGRES_DSN environment variable not set")
	}

	// Get database info
	dbInfo, err := GetDBInfo(dsn)
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
