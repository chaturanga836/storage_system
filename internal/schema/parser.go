package schema

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SQLParser parses SQL DDL statements into internal schema representations
type SQLParser struct {
	caseSensitive bool
	strictMode    bool
}

// NewSQLParser creates a new SQL parser
func NewSQLParser() *SQLParser {
	return &SQLParser{
		caseSensitive: false,
		strictMode:    false,
	}
}

// ParseCreateTable parses a CREATE TABLE statement
func (p *SQLParser) ParseCreateTable(sql string) (*TableSchema, error) {
	// Normalize the SQL
	sql = strings.TrimSpace(sql)
	if !p.caseSensitive {
		sql = strings.ToLower(sql)
	}

	// Basic regex to extract table name and column definitions
	createTableRegex := regexp.MustCompile(`create\s+table\s+(?:if\s+not\s+exists\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)\s*\(\s*(.*)\s*\)`)

	matches := createTableRegex.FindStringSubmatch(sql)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid CREATE TABLE syntax")
	}

	tableName := matches[1]
	columnDefsStr := matches[2]

	// Parse table name (handle namespace.table format)
	var namespace, name string
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		namespace = parts[0]
		name = parts[1]
	} else {
		name = tableName
	}

	// Create table schema
	schema := NewTableSchema(name, namespace)

	// Parse column definitions
	err := p.parseColumnDefinitions(columnDefsStr, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse column definitions: %w", err)
	}

	return schema, nil
}

// parseColumnDefinitions parses the column definitions part of CREATE TABLE
func (p *SQLParser) parseColumnDefinitions(columnDefsStr string, schema *TableSchema) error {
	// Split by commas, but be careful about commas inside parentheses
	columnDefs := p.splitColumnDefinitions(columnDefsStr)

	var primaryKeyColumns []string

	for _, columnDef := range columnDefs {
		columnDef = strings.TrimSpace(columnDef)

		// Check for table-level constraints
		if strings.HasPrefix(columnDef, "primary key") {
			pkCols, err := p.parsePrimaryKeyConstraint(columnDef)
			if err != nil {
				return err
			}
			primaryKeyColumns = append(primaryKeyColumns, pkCols...)
			continue
		}

		if strings.HasPrefix(columnDef, "unique") ||
			strings.HasPrefix(columnDef, "index") ||
			strings.HasPrefix(columnDef, "key") {
			// Handle index definitions
			index, err := p.parseIndexDefinition(columnDef)
			if err != nil {
				return err
			}
			if index != nil {
				schema.AddIndex(*index)
			}
			continue
		}

		// Parse column definition
		column, isPrimaryKey, err := p.parseColumnDefinition(columnDef)
		if err != nil {
			return fmt.Errorf("failed to parse column '%s': %w", columnDef, err)
		}

		err = schema.AddColumn(*column)
		if err != nil {
			return err
		}

		if isPrimaryKey {
			primaryKeyColumns = append(primaryKeyColumns, column.Name)
		}
	}

	// Set primary key if any were found
	if len(primaryKeyColumns) > 0 {
		err := schema.SetPrimaryKey(primaryKeyColumns)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseColumnDefinition parses a single column definition
func (p *SQLParser) parseColumnDefinition(columnDef string) (*ColumnSchema, bool, error) {
	// Basic column definition pattern: name type [constraints]
	parts := strings.Fields(columnDef)
	if len(parts) < 2 {
		return nil, false, fmt.Errorf("invalid column definition: %s", columnDef)
	}

	columnName := parts[0]
	columnTypeStr := parts[1]
	constraints := parts[2:]

	// Parse data type
	dataType, length, precision, scale, err := p.parseDataType(columnTypeStr)
	if err != nil {
		return nil, false, err
	}

	// Create column schema
	column, err := NewColumnSchema(columnName, dataType, true) // Default to nullable
	if err != nil {
		return nil, false, err
	}

	// Set type-specific properties
	if length != nil {
		column.Length = length
	}
	if precision != nil {
		column.Precision = precision
	}
	if scale != nil {
		column.Scale = scale
	}

	// Parse constraints
	isPrimaryKey := false
	for _, constraint := range constraints {
		constraint = strings.ToLower(constraint)

		switch constraint {
		case "not", "null":
			if len(constraints) > 1 && constraints[1] == "null" {
				column.Nullable = false
			}
		case "primary":
			isPrimaryKey = true
		case "unique":
			// Add unique index
			indexName := fmt.Sprintf("idx_%s_%s_unique", column.Name, "unique")
			index := IndexSchema{
				Name:    indexName,
				Columns: []string{column.Name},
				Type:    "btree",
				Unique:  true,
			}
			// Note: This would need to be added to the table schema later
			_ = index
		case "default":
			// Handle default values (simplified)
			// In a real implementation, you'd parse the actual default value
		}
	}

	return column, isPrimaryKey, nil
}

// parseDataType parses SQL data types into internal representations
func (p *SQLParser) parseDataType(typeStr string) (DataType, *int, *int, *int, error) {
	typeStr = strings.ToLower(strings.TrimSpace(typeStr))

	// Handle types with parameters like VARCHAR(255) or DECIMAL(10,2)
	paramRegex := regexp.MustCompile(`([a-zA-Z]+)(?:\((\d+)(?:,(\d+))?\))?`)
	matches := paramRegex.FindStringSubmatch(typeStr)

	if len(matches) == 0 {
		return "", nil, nil, nil, fmt.Errorf("invalid data type: %s", typeStr)
	}

	baseType := matches[1]
	var length, precision, scale *int

	if len(matches) > 2 && matches[2] != "" {
		l, err := strconv.Atoi(matches[2])
		if err == nil {
			length = &l
			precision = &l
		}
	}

	if len(matches) > 3 && matches[3] != "" {
		s, err := strconv.Atoi(matches[3])
		if err == nil {
			scale = &s
		}
	}

	// Map SQL types to internal types
	var dataType DataType

	switch baseType {
	case "varchar", "char", "text", "string":
		dataType = TypeString
	case "int", "integer", "int32":
		dataType = TypeInt32
	case "bigint", "long", "int64":
		dataType = TypeInt64
	case "float", "real", "float32":
		dataType = TypeFloat32
	case "double", "float64":
		dataType = TypeFloat64
	case "boolean", "bool":
		dataType = TypeBoolean
	case "bytea", "bytes", "binary":
		dataType = TypeBytes
	case "timestamp", "datetime":
		dataType = TypeTimestamp
	case "date":
		dataType = TypeDate
	case "time":
		dataType = TypeTime
	case "uuid":
		dataType = TypeUUID
	case "decimal", "numeric":
		dataType = TypeDecimal
	case "array":
		dataType = TypeArray
	case "json", "jsonb":
		dataType = TypeString // Store as string for now
	default:
		return "", nil, nil, nil, fmt.Errorf("unsupported data type: %s", baseType)
	}

	return dataType, length, precision, scale, nil
}

// splitColumnDefinitions splits column definitions while respecting parentheses
func (p *SQLParser) splitColumnDefinitions(columnDefsStr string) []string {
	var result []string
	var current strings.Builder
	var parenCount int

	for _, char := range columnDefsStr {
		switch char {
		case '(':
			parenCount++
			current.WriteRune(char)
		case ')':
			parenCount--
			current.WriteRune(char)
		case ',':
			if parenCount == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// Add the last part
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// parsePrimaryKeyConstraint parses a primary key constraint
func (p *SQLParser) parsePrimaryKeyConstraint(constraintDef string) ([]string, error) {
	// Extract column names from PRIMARY KEY (col1, col2)
	pkRegex := regexp.MustCompile(`primary\s+key\s*\(\s*([^)]+)\s*\)`)
	matches := pkRegex.FindStringSubmatch(constraintDef)

	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid primary key constraint: %s", constraintDef)
	}

	columnList := matches[1]
	columns := strings.Split(columnList, ",")

	for i, col := range columns {
		columns[i] = strings.TrimSpace(col)
	}

	return columns, nil
}

// parseIndexDefinition parses an index definition
func (p *SQLParser) parseIndexDefinition(indexDef string) (*IndexSchema, error) {
	// This is a simplified parser for index definitions
	// In practice, you'd need much more sophisticated parsing

	indexDef = strings.ToLower(strings.TrimSpace(indexDef))

	// Handle UNIQUE INDEX name (columns)
	uniqueRegex := regexp.MustCompile(`unique(?:\s+index)?\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(\s*([^)]+)\s*\)`)
	matches := uniqueRegex.FindStringSubmatch(indexDef)

	if len(matches) == 3 {
		indexName := matches[1]
		columnList := matches[2]
		columns := strings.Split(columnList, ",")

		for i, col := range columns {
			columns[i] = strings.TrimSpace(col)
		}

		return &IndexSchema{
			Name:    indexName,
			Columns: columns,
			Type:    "btree",
			Unique:  true,
		}, nil
	}

	// Handle INDEX name (columns)
	indexRegex := regexp.MustCompile(`index\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(\s*([^)]+)\s*\)`)
	matches = indexRegex.FindStringSubmatch(indexDef)

	if len(matches) == 3 {
		indexName := matches[1]
		columnList := matches[2]
		columns := strings.Split(columnList, ",")

		for i, col := range columns {
			columns[i] = strings.TrimSpace(col)
		}

		return &IndexSchema{
			Name:    indexName,
			Columns: columns,
			Type:    "btree",
			Unique:  false,
		}, nil
	}

	return nil, nil // Not a recognized index definition
}

// ParseAlterTable parses ALTER TABLE statements into schema evolutions
func (p *SQLParser) ParseAlterTable(sql string) ([]SchemaEvolution, error) {
	sql = strings.TrimSpace(sql)
	if !p.caseSensitive {
		sql = strings.ToLower(sql)
	}

	// Extract table name
	alterRegex := regexp.MustCompile(`alter\s+table\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)\s+(.+)`)
	matches := alterRegex.FindStringSubmatch(sql)

	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid ALTER TABLE syntax")
	}

	tableName := matches[1]
	alterActions := matches[2]

	var evolutions []SchemaEvolution

	// Parse different ALTER actions
	if strings.HasPrefix(alterActions, "add column") {
		evolution, err := p.parseAddColumn(tableName, alterActions)
		if err != nil {
			return nil, err
		}
		evolutions = append(evolutions, *evolution)
	} else if strings.HasPrefix(alterActions, "drop column") {
		evolution, err := p.parseDropColumn(tableName, alterActions)
		if err != nil {
			return nil, err
		}
		evolutions = append(evolutions, *evolution)
	} else if strings.HasPrefix(alterActions, "modify column") || strings.HasPrefix(alterActions, "alter column") {
		evolution, err := p.parseModifyColumn(tableName, alterActions)
		if err != nil {
			return nil, err
		}
		evolutions = append(evolutions, *evolution)
	}

	return evolutions, nil
}

// parseAddColumn parses ADD COLUMN statements
func (p *SQLParser) parseAddColumn(tableName, alterAction string) (*SchemaEvolution, error) {
	// Extract column definition from "ADD COLUMN name type ..."
	addColRegex := regexp.MustCompile(`add\s+column\s+(.+)`)
	matches := addColRegex.FindStringSubmatch(alterAction)

	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid ADD COLUMN syntax")
	}

	columnDef := matches[1]
	column, _, err := p.parseColumnDefinition(columnDef)
	if err != nil {
		return nil, err
	}

	return &SchemaEvolution{
		Type:       "add_column",
		TableName:  tableName,
		ColumnName: column.Name,
		NewSchema:  column,
		Timestamp:  time.Now(),
	}, nil
}

// parseDropColumn parses DROP COLUMN statements
func (p *SQLParser) parseDropColumn(tableName, alterAction string) (*SchemaEvolution, error) {
	// Extract column name from "DROP COLUMN name"
	dropColRegex := regexp.MustCompile(`drop\s+column\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := dropColRegex.FindStringSubmatch(alterAction)

	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid DROP COLUMN syntax")
	}

	columnName := matches[1]

	return &SchemaEvolution{
		Type:       "drop_column",
		TableName:  tableName,
		ColumnName: columnName,
		Timestamp:  time.Now(),
	}, nil
}

// parseModifyColumn parses MODIFY/ALTER COLUMN statements
func (p *SQLParser) parseModifyColumn(tableName, alterAction string) (*SchemaEvolution, error) {
	// Extract column definition from "MODIFY COLUMN name type ..." or "ALTER COLUMN name type ..."
	modifyColRegex := regexp.MustCompile(`(?:modify|alter)\s+column\s+(.+)`)
	matches := modifyColRegex.FindStringSubmatch(alterAction)

	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid MODIFY/ALTER COLUMN syntax")
	}

	columnDef := matches[1]
	column, _, err := p.parseColumnDefinition(columnDef)
	if err != nil {
		return nil, err
	}

	return &SchemaEvolution{
		Type:       "modify_column",
		TableName:  tableName,
		ColumnName: column.Name,
		NewSchema:  column,
		Timestamp:  time.Now(),
	}, nil
}

// ParseDropTable parses DROP TABLE statements
func (p *SQLParser) ParseDropTable(sql string) (string, error) {
	sql = strings.TrimSpace(sql)
	if !p.caseSensitive {
		sql = strings.ToLower(sql)
	}

	dropRegex := regexp.MustCompile(`drop\s+table\s+(?:if\s+exists\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	matches := dropRegex.FindStringSubmatch(sql)

	if len(matches) != 2 {
		return "", fmt.Errorf("invalid DROP TABLE syntax")
	}

	return matches[1], nil
}

// ValidateSQL performs basic SQL syntax validation
func (p *SQLParser) ValidateSQL(sql string) error {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return fmt.Errorf("empty SQL statement")
	}

	// Check for basic SQL injection patterns (very basic)
	suspiciousPatterns := []string{
		"';",
		"--",
		"/*",
		"*/",
		"xp_",
		"sp_",
	}

	lowerSQL := strings.ToLower(sql)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerSQL, pattern) {
			return fmt.Errorf("potentially unsafe SQL pattern detected: %s", pattern)
		}
	}

	return nil
}
