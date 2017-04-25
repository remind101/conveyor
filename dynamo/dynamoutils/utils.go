package dynamoutils

import (
	"os"
	"strings"
)

// Helper function to parse environment variable in the form
// VAR=table1,table2,table3 into a map
func TablesMapFromEnv(env string) map[string]bool {
	tablesString := os.Getenv(env)
	tableList := strings.Split(tablesString, ",")
	tableMap := make(map[string]bool)
	for _, name := range tableList {
		tableMap[name] = true
		// Also save the unscoped tablename
		index := strings.Index(name, "_")
		if index != -1 {
			tableMap[name[(index+1):len(name)]] = true
		}
	}
	return tableMap
}

// Helper function to determine if table is enabled for the given environment variable
// Supports scoped and unscoped tablenames
func IsTableEnabledFromEnv(env string, tableName string) bool {
	tablesMap := TablesMapFromEnv(env)
	unscopedTablename := tableName[(strings.Index(tableName, "_") + 1):len(tableName)]
	return tablesMap[tableName] || tablesMap[unscopedTablename]
}
