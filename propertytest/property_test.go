package propertytest

// "github.com/leanovate/gopter/testutil" // Removed

// Map and mutex are now defined in commands.go
// var commandCounts = make(map[string]int)
// var commandCountMutex sync.Mutex

// recordCommandExecution is now defined in commands.go

// NOTE: We fixed the property test issues by implementing custom generators
// that avoid the type conversion problems associated with chained Map() calls.
// The approach uses custom generator functions that handle type conversions properly
// and more carefully manage the sequence of operations to ensure the model
// state stays in sync with the actual system under test.

// This file might contain shared helper functions or setup logic in the future,
// but the main test functions have been moved to separate files.
