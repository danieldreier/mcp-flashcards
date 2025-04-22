# [Project Name] Implementation Plan - Phase [X] ([Phase Name])

This document outlines the implementation plan for [Project Name], focusing on Phase [X]: [Phase Name]. Each task is designed to be implemented as a standalone unit with comprehensive specifications.

**IMPORTANT:** Tests must be executed between tasks, not just at the end of the phase. Each task should include explicit test steps throughout its implementation to ensure incremental validation. Do not proceed to the next task until all tests for the current task pass.

## Previously Completed Tasks (if applicable)

These tasks were already implemented:

- [x] Task [X.1]: [Task Title]
- [x] Task [X.2]: [Task Title]
- [x] Task [X.3]: [Task Title]

## Implementation Plan for [Current Focus]

### Phase [X.A]: [Sub-phase Name]

#### Task [X.A.1]: [First Task Title]

```
# Task [X.A.1]: [First Task Title] ✅

## Background and Context
[Provide background information about the project and specific context for this task. Explain why this task is important and how it fits into the overall system architecture.]

## My Task
[Clearly define what needs to be accomplished in this task. Be specific about the requirements and constraints.]

## Files to Modify
1. `path/to/file1`: [Brief description of changes needed]
2. `path/to/file2`: [Brief description of changes needed]

## Implementation Details
In `path/to/file1`:
1. [Specific change to make]
   - [Sub-detail if necessary]
   - [Sub-detail if necessary]
2. [Specific change to make]
3. [Specific change to make, possibly with code sample]:
   ```[language]
   // Code example here
   function example() {
       return "example code";
   }
   ```

In `path/to/file2`:
1. [Specific change to make]
2. [Specific change to make]
3. [Specific change to make]

## Behaviors to Test
The following functional behaviors should be tested:
1. [First behavior] - [Why this behavior matters for users/system]
2. [Second behavior] - [Why this behavior matters for users/system]
3. [Error case: Invalid input] - [Expected error handling behavior]
4. [Edge case] - [How the system should handle this boundary condition]

## Success Criteria
- [x] [Behavior] [Success Criterion] [Test Case]
- [x] [Behavior] [Success Criterion] [Test Case]
- [x] [Behavior] [Success Criterion] [Test Case]
- [x] [Behavior] [Success Criterion] [Test Case]

## Step-by-Step Implementation
1. [x] [First component implementation step]
2. [x] Write tests for this component
3. [x] Run tests to verify this component (`go test ./path -run=TestSpecificComponent`)
4. [x] [Second component implementation step]
5. [x] Update tests for the second component
6. [x] Run tests to verify this component (`go test ./path -run=TestSpecificComponent`)
7. [x] [Final component implementation step]
8. [x] Ensure all tests pass for the complete task (`go test ./path`)
9. [x] Run linters and static code analysis

## Implementation Notes
- [Detail about what was actually implemented]
- [Detail about what was actually implemented]
- [Detail about what was actually implemented]
- [Detail about what was actually implemented]
- [Detail about any issues encountered and how they were resolved]
```

#### Task [X.A.2]: [Second Task Title]

```
# Task [X.A.2]: [Second Task Title]

## Background and Context
[Provide background information about the project and specific context for this task. This section should explain the problem being solved and why it matters.]

## My Task
[Clearly define what needs to be accomplished in this task. Be specific about the requirements and expected deliverables.]

## Interfaces and Method Signatures
I'll need to implement the following in file `path/to/file`:

```[language]
// TypeName describes what this type represents
type TypeName interface {
    // MethodName describes what this method does
    MethodName(param ParamType) ReturnType
    
    // OtherMethod describes what this other method does
    OtherMethod(ctx Context) (ResultType, error)
}

// Implementation of the interface
type implementation struct {
    field1 Type1
    field2 Type2
}

// Constructor function
func NewTypeName(param1 Type1, param2 Type2) TypeName {
    // Implementation here
}

// MethodName implements the TypeName interface
func (i *implementation) MethodName(param ParamType) ReturnType {
    // Implementation here
}

// OtherMethod implements the TypeName interface
func (i *implementation) OtherMethod(ctx Context) (ResultType, error) {
    // Implementation here
}
```

## Implementation Details
I should follow these guidelines for implementation:
1. [Guideline 1]
2. [Guideline 2]
3. [Guideline 3]
4. [Guideline 4] with example:
   ```[language]
   // Example code showing implementation approach
   ```
5. [Guideline 5]

## Behaviors to Test
The following functional behaviors should be tested:
1. [First behavior]: [specific behavior statement]
2. [Second behavior]: [specific behavior statement]
3. [Third behavior]: [specific behavior statement]
4. [Error handling behavior]: [specific behavior statement]
5. [Edge case behavior]: [specific behavior statement]

## Success Criteria
- [ ] [First behavior] [Success Criterion] [TestCase1]
- [ ] [Second behavior] [Success Criterion] [TestCase2]
- [ ] [Third behavior] [Success Criterion] [TestCase3]
- [ ] [Error handling behavior] [Success Criterion] [TestCase4]
- [ ] [Edge case behavior] [Success Criterion] [TestCase5]

## Step-by-Step Implementation
1. [ ] [First component implementation step]
2. [ ] Write tests for this component
3. [ ] Run tests to verify this component (`go test ./path -run=TestSpecificComponent`)
4. [ ] [Second component implementation step]
5. [ ] Update tests for the second component
6. [ ] Run tests to verify this component (`go test ./path -run=TestSpecificComponent`)
7. [ ] [Final component implementation step]
8. [ ] Ensure all tests pass for the complete task (`go test ./path`)
9. [ ] Run linters and static code analysis
```

## Execution Order

1. **First, [initial focus]**:
   - Task [X.A.1]: [First Task Title]
   - Task [X.A.2]: [Second Task Title]

2. **Then, [second focus]**:
   - Task [X.B.1]: [Task Title]
   - Task [X.B.2]: [Task Title]

This sequence ensures that we [explain rationale for this ordering].

## Phase [X] Completion Criteria

At the end of this phase, we should have:

1. [Deliverable/outcome 1]
2. [Deliverable/outcome 2]
3. [Deliverable/outcome 3]
4. [Deliverable/outcome 4]
5. [Deliverable/outcome 5]

The implementation should correctly handle:
- [Capability 1]
- [Capability 2]
- [Capability 3]
- [Capability 4]

All tests should pass and the code should adhere to the project's style guide.

## Examples

The following examples demonstrate how to use this template with concrete implementations.

### Example 1: Implementing a Logging System

```
# Task 2.1: Implement Structured Logging System ✅

## Background and Context
Our service framework requires a robust logging system to capture application events in a consistent, searchable format. Currently, log information is inconsistent and difficult to correlate across services. A structured logging system will provide standardized fields, log levels, and output formats to improve operational visibility and debugging capabilities.

## My Task
My task is to implement a structured logging system that supports multiple output formats (JSON, text), configurable log levels, and contextual metadata. The system should follow a singleton pattern to provide a consistent logging interface throughout the application.

## Files to Modify
1. `pkg/logging/logger.go`: Create a new structured logging implementation
2. `pkg/logging/formatter.go`: Implement formatting options for logs

## Implementation Details
In `pkg/logging/logger.go`:
1. Create a Logger interface with methods for different log levels:
   - Debug, Info, Warn, Error, Fatal
   - Each method should accept a message and optional key-value pairs
2. Implement a structured logger that:
   - Uses zap as the underlying logging library
   - Supports configurable log levels
   - Includes standard fields like timestamp, service name, and log level
3. Implement a singleton pattern for global logger access:
   ```go
   // GetLogger returns the global logger instance
   func GetLogger() Logger {
       once.Do(func() {
           defaultLogger = newStructuredLogger(config)
       })
       return defaultLogger
   }
   ```
In `pkg/logging/formatter.go`:
1. Implement JSON formatting for machine readability
2. Implement text formatting for development and console output
3. Create a configuration system for switching between formatters

## Behaviors to Test
The following functional behaviors should be tested:
1. Log level filtering: messages are logged based on configured log level
2. Structured data serialization: fields are properly escaped and formatted in both JSON and text formats
3. Context propagation: metadata from context appears in generated log entries
4. Error handling: malformed data is handled safely with log messages showing that a malformed message could not be logged
5. Thread safety: concurrent logging operations produce consistent and correctly formatted output

## Success Criteria
- [x] [Log level filtering] [Only logs at or above configured level are emitted] [TestLogLevelFiltering]
- [x] [Structured data serialization] [JSON output is valid and parseable] [TestJSONFormatter]
- [x] [Structured data serialization] [Text output is human-readable with proper formatting] [TestTextFormatter]
- [x] [Context propagation] [Context values appear in output for all log levels] [TestContextPropagation]
- [x] [Thread safety] [No data corruption when logging from multiple goroutines] [TestConcurrentLogging]
- [x] All tests pass and demonstrate proper functionality

## Step-by-Step Implementation
1. [x] Create the Logger interface with required methods
2. [x] Implement the structured logger using zap
3. [x] Add support for JSON and text formatters
4. [x] Implement the singleton pattern for global access
5. [x] Write comprehensive tests for all logger functionality

## Implementation Notes
- Successfully implemented the Logger interface with debug, info, warn, error, and fatal methods
- Used zap for the underlying implementation due to its performance characteristics
- Added support for both JSON and text output formats with appropriate escaping
- Implemented field enrichment to add standard fields to all log messages
- Added context support for request tracking across components
- Included sampling capabilities to prevent log flooding in high-volume scenarios
- All 23 tests are passing with 92% code coverage
```

### Example 2: Creating an Error Handling Framework

```
# Task 2.2: Implement Error Handling Framework

## Background and Context
A consistent error handling approach is essential for a reliable service framework. Currently, errors are handled inconsistently across the codebase, making it difficult to determine error causes and appropriate responses. We need a standardized error system that provides rich context, supports error wrapping, and enables precise error handling in API responses.

## My Task
My task is to implement an error handling framework that supports categorization, context enrichment, and proper error wrapping. The framework should make it easy to create, propagate, and handle errors consistently throughout the application.

## Interfaces and Method Signatures
I'll need to implement the following in file `pkg/errors/errors.go`:

```go
// ErrorCode identifies specific error categories
type ErrorCode string

// Common error codes in the system
const (
    // General errors
    CodeInternal      ErrorCode = "internal"
    CodeInvalidInput  ErrorCode = "invalid_input"
    CodeUnauthorized  ErrorCode = "unauthorized"
    CodeNotFound      ErrorCode = "not_found"
    CodeAlreadyExists ErrorCode = "already_exists"
    CodeUnavailable   ErrorCode = "unavailable"
)

// AppError represents a structured application error
type AppError struct {
    Code     ErrorCode
    Message  string
    Cause    error
    Context  map[string]interface{}
    HttpCode int
}

// Error implements the error interface
func (e *AppError) Error() string {
    // Implementation here
}

// Unwrap returns the underlying cause for errors.Is and errors.As support
func (e *AppError) Unwrap() error {
    // Implementation here
}

// WithContext adds additional context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
    // Implementation here
}

// StatusCode returns the HTTP status code for this error
func (e *AppError) StatusCode() int {
    // Implementation here
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
    // Implementation here
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code ErrorCode, message string) *AppError {
    // Implementation here
}

// Helper functions for common error types
func NewInternalError(message string) *AppError {
    // Implementation here
}

func NewInvalidInputError(message string) *AppError {
    // Implementation here
}

func NewNotFoundError(resource string, id interface{}) *AppError {
    // Implementation here
}
```

## Implementation Details
I should follow these guidelines for implementation:
1. Implement the AppError struct to store error details including code, message, and context
2. The Error method should format the error message to include all relevant information
3. Support proper error wrapping to maintain the full error chain
4. Provide helper functions for common error types to encourage consistent usage
5. Enable HTTP status code mapping for API responses with example:
   ```go
   // StatusCode maps error codes to HTTP status codes
   func (e *AppError) StatusCode() int {
       codeToStatus := map[ErrorCode]int{
           CodeInternal:      http.StatusInternalServerError,
           CodeInvalidInput:  http.StatusBadRequest,
           CodeUnauthorized:  http.StatusUnauthorized,
           CodeNotFound:      http.StatusNotFound,
           CodeAlreadyExists: http.StatusConflict,
           CodeUnavailable:   http.StatusServiceUnavailable,
       }
       
       if status, ok := codeToStatus[e.Code]; ok {
           return status
       }
       return http.StatusInternalServerError
   }
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Error creation and formatting: creating specific error types results in properly formatted messages
2. Error wrapping: wrapped errors maintain the original error chain intact
3. Error context: context can be added to errors for additional debugging information
4. HTTP status mapping: error codes properly map to appropriate HTTP status codes
5. Error type identification: errors can be identified by type using errors.Is and errors.As

## Success Criteria
- [ ] [Error creation and formatting] [Each error type produces a clear, descriptive message] [TestErrorFormatting]
- [ ] [Error wrapping] [Wrapped errors can be unwrapped to access the original error] [TestErrorUnwrapping]
- [ ] [Error context] [Context values are properly stored and retrieved] [TestErrorContext]
- [ ] [HTTP status mapping] [Error codes map to the correct HTTP status codes] [TestStatusCodeMapping]
- [ ] [Error type identification] [errors.Is and errors.As work correctly with custom errors] [TestErrorTypeChecking]

## Step-by-Step Implementation
1. [ ] Create the `pkg/errors/errors.go` file with ErrorCode and AppError types
2. [ ] Implement the Error and Unwrap methods
3. [ ] Implement the WithContext and StatusCode methods
4. [ ] Create helper functions for common error types
5. [ ] Write comprehensive tests for all functionality
```    }
       
       if status, ok := codeToStatus[e.Code]; ok {
           return status
       }
       return http.StatusInternalServerError
   }
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Error creation and formatting: creating specific error types results in properly formatted messages
2. Error wrapping: wrapped errors maintain the original error chain intact
3. Error context: context can be added to errors for additional debugging information
4. HTTP status mapping: error codes properly map to appropriate HTTP status codes
5. Error type identification: errors can be identified by type using errors.Is and errors.As

## Success Criteria
- [ ] [Error creation and formatting] [Each error type produces a clear, descriptive message] [TestErrorFormatting]
- [ ] [Error wrapping] [Wrapped errors can be unwrapped to access the original error] [TestErrorUnwrapping]
- [ ] [Error context] [Context values are properly stored and retrieved] [TestErrorContext]
- [ ] [HTTP status mapping] [Error codes map to the correct HTTP status codes] [TestStatusCodeMapping]
- [ ] [Error type identification] [errors.Is and errors.As work correctly with custom errors] [TestErrorTypeChecking]

## Step-by-Step Implementation
1. [ ] Create the `pkg/errors/errors.go` file with ErrorCode and AppError types
2. [ ] Implement the Error and Unwrap methods
3. [ ] Implement the WithContext and StatusCode methods
4. [ ] Create helper functions for common error types
5. [ ] Write comprehensive tests for all functionality
```