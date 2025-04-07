# CPN Task Template Examples

## Example 1: Implement Color Interface and Basic Color Implementation

### Background and Context
I am building a Colored Petri Net (CPN) executor in Go. The CPN executor is responsible for managing the execution of complex workflows represented as Petri nets where tokens have specific "colors" (types). This standalone library implements Jensen's CPN model for control plane infrastructure systems.

The Color interface is a fundamental building block of this system that represents token types and provides type validation. This component is the foundation of the type system used throughout the entire CPN executor.

### My Task
My task is to implement the Color interface and a basic Color implementation for the CPN system. I need to create the necessary files, implement the required interfaces, and ensure everything is properly tested.

I should write clean, idiomatic, and elegant Go code that follows standard Go conventions and best practices.

### Interfaces and Method Signatures
I'll need to implement the following interfaces in file `pkg/cpn/color.go`:

```go
// Color represents a token color (type) in the CPN system
type Color interface {
    // ID returns the unique identifier for this color
    ID() string
    
    // Name returns the human-readable name for this color
    Name() string
    
    // Validate checks if a value is valid for this color
    Validate(value interface{}) error
}
```

I'll also need to implement a concrete color implementation in the same file:

```go
// BasicColor provides a simple implementation of the Color interface
type BasicColor struct {
    id          string
    name        string
    validateFn  func(interface{}) error
}

// NewColor creates a new BasicColor with the given ID, name, and validation function
func NewColor(id string, name string, validateFn func(interface{}) error) Color {
    // Implementation here
}
```

### Implementation Details
I should follow these guidelines for implementation:
- The ID must be a unique string identifier for the color
- The Name should be a human-readable description
- The Validate function should use the provided validateFn to check if a value is valid for this color
- If validateFn is nil, the Validate method should accept any value (return nil)
- IDs should not contain spaces and should be lowercase, following Go naming conventions
- Use pkg/errors for returning errors with appropriate context

### Success Criteria
My implementation will be considered successful if:

- [ ] All methods of the Color interface are properly implemented
- [ ] The NewColor function correctly creates a BasicColor
- [ ] Validation correctly accepts or rejects values based on the provided validateFn
- [ ] Test cases for string, int, and custom type validation pass
- [ ] All tests pass without any race conditions or memory leaks
- [ ] Code passes go vet and golangci-lint checks

### Step-by-Step Implementation
1. [ ] Create file `pkg/cpn/color.go` with the Color interface and BasicColor implementation
2. [ ] Create test file `pkg/cpn/color_test.go` using testify for assertions
3. [ ] Implement a test case for creating colors with different IDs and names
4. [ ] Run the test with `go test ./pkg/cpn -v -run=TestColorCreation` and fix any issues
5. [ ] Implement a test case for validating string values
6. [ ] Run the test with `go test ./pkg/cpn -v -run=TestColorValidateString` and fix any issues

### Status Update Format
After completing the implementation, I should provide a status update including:
- Files created/updated
- Methods implemented
- Test results (number of tests passed)
- Any issues encountered and how they were resolved
- Which success criteria were met

## Example 2: Implement Network Interface and Basic Network Implementation

### Background and Context
I am building a Colored Petri Net (CPN) executor in Go. The CPN executor is responsible for managing the execution of complex workflows represented as Petri nets where tokens have specific "colors" (types). This standalone library implements Jensen's CPN model for control plane infrastructure systems.

In a Colored Petri Net, the Network represents the entire structure of places and transitions and their connections. It serves as a central registry for all components and provides methods to query their relationships.

### My Task
My task is to implement the Network interface and a basic Network implementation for the CPN system. This component will maintain the structure of the CPN and provide methods to query the connections between places and transitions. I need to create the necessary files, implement the required interfaces, and ensure everything is properly tested.

I should write clean, idiomatic, and elegant Go code that follows standard Go conventions and best practices.

### Interfaces and Method Signatures
I'll need to implement the following interface in file `pkg/cpn/network.go`:

```go
// Network represents a CPN network structure
type Network interface {
    // Places returns a map of all places in the network
    Places() map[PlaceID]Place
    
    // Transitions returns a map of all transitions in the network
    Transitions() map[TransitionID]Transition
    
    // AddPlace adds a place to the network
    // Returns an error if a place with the same ID already exists
    AddPlace(place Place) error
    
    // AddTransition adds a transition to the network
    // Returns an error if a transition with the same ID already exists
    // or if any input or output place doesn't exist
    AddTransition(transition Transition) error
    
    // GetInputPlaces returns all input places for a transition
    GetInputPlaces(transitionID TransitionID) []PlaceID
}
```

### Implementation Details
I should follow these guidelines for implementation:
- The NewNetwork function should initialize an empty network with all necessary maps
- Pre-allocate maps with expected capacity as recommended in the style guide
- The Places and Transitions methods should return copies of the internal maps to prevent modification
- The AddPlace method should check if a place with the same ID already exists
- The AddTransition method should verify that all input and output places exist in the network
- Use pkg/errors for error handling with proper context

### Success Criteria
My implementation will be considered successful if:

- [ ] All methods of the Network interface are properly implemented
- [ ] The NewNetwork function correctly creates an empty network
- [ ] Adding places and transitions works correctly
- [ ] The network correctly validates that places exist before adding transitions
- [ ] The network correctly maintains the placeToTransitions mapping
- [ ] All getter methods return the correct values
- [ ] All tests pass without errors
- [ ] Code passes go vet and golangci-lint checks

### Step-by-Step Implementation
1. [ ] Create file `pkg/cpn/network.go` with the Network interface and BasicNetwork implementation
2. [ ] Create test file `pkg/cpn/network_test.go` using testify for assertions
3. [ ] Implement a test case for creating an empty network
4. [ ] Run the test with `go test ./pkg/cpn -v -run=TestNewNetwork` and fix any issues
5. [ ] Implement a test case for adding places
6. [ ] Run the test with `go test ./pkg/cpn -v -run=TestAddPlace` and fix any issues

### Status Update Format
After completing the implementation, I should provide a status update including:
- Files created/updated
- Methods implemented
- Test results (number of tests passed)
- Any issues encountered and how they were resolved
- Which success criteria were met
