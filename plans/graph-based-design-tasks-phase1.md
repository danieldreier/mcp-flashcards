# Graph-Based Flashcards MCP - Phase 1 Implementation Tasks

## Background and Context

We are enhancing the Flashcards MCP server with graph-based functionality. The current implementation is a simple spaced repetition system that schedules flashcards according to the Free Spaced Repetition Scheduler (FSRS) algorithm, without any understanding of the relationships between knowledge concepts.

This enhancement will transform the system into an intelligent learning platform that can represent prerequisite relationships between concepts, ensure learners master foundational concepts before advancing, detect knowledge gaps, and adaptively teach missing concepts. Phase 1 focuses on implementing the core graph data structure that will support all these capabilities.

The graph structure will represent two types of relationships:
1. Card-to-card relationships (which cards are prerequisites for other cards)
2. Concept-to-concept relationships (which knowledge concepts depend on others)

## My Task

My task is to implement the basic directed graph structure that will form the foundation of the graph-based flashcard system. I need to:

1. Define the necessary data structures for representing a directed graph
2. Extend the existing FlashcardStore to include these graph structures
3. Implement the core graph operations (adding/removing nodes and edges)
4. Update the persistence methods to properly store and load graph data
5. Create comprehensive tests for all graph operations

I should write clean, idiomatic Go code that integrates well with the existing codebase while ensuring thread safety and data integrity.

## Interfaces and Method Signatures

I'll need to implement the following structures and methods in `internal/storage/storage.go`:

```go
// DirectedGraph represents a simple directed graph structure
type DirectedGraph struct {
    // Map from node ID to its prerequisite node IDs
    Prerequisites map[string][]string `json:"prerequisites"`
    // Map from node ID to the IDs of nodes that depend on it
    Dependents map[string][]string `json:"dependents"`
    // Mutex for concurrent access protection
    mu sync.RWMutex `json:"-"`
}

// NewDirectedGraph creates a new empty directed graph
func NewDirectedGraph() *DirectedGraph {
    // Implementation here
}

// AddNode adds a new node to the graph
func (g *DirectedGraph) AddNode(nodeID string) {
    // Implementation here
}

// RemoveNode removes a node and all its edges from the graph
func (g *DirectedGraph) RemoveNode(nodeID string) {
    // Implementation here
}

// AddEdge creates a directed edge from source to target
func (g *DirectedGraph) AddEdge(sourceID, targetID string) error {
    // Implementation here
}

// RemoveEdge removes a directed edge from source to target
func (g *DirectedGraph) RemoveEdge(sourceID, targetID string) error {
    // Implementation here
}

// GetPrerequisites returns all prerequisites for a given node
func (g *DirectedGraph) GetPrerequisites(nodeID string) []string {
    // Implementation here
}

// GetDependents returns all nodes that depend on the given node
func (g *DirectedGraph) GetDependents(nodeID string) []string {
    // Implementation here
}

// HasCycle detects if the graph contains any cycles
func (g *DirectedGraph) HasCycle() bool {
    // Implementation here
}

// ConceptNode represents a knowledge concept in the graph
type ConceptNode struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`
    Description   string    `json:"description,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
}

// Enhanced FlashcardStore with graph structures
type FlashcardStore struct {
    Cards           map[string]Card       `json:"cards"`
    Reviews         []Review              `json:"reviews"`
    Concepts        map[string]ConceptNode `json:"concepts"`
    CardGraph       DirectedGraph         `json:"card_graph"`
    ConceptGraph    DirectedGraph         `json:"concept_graph"`
    Relationships   CardRelationship      `json:"relationships"`
    LastUpdated     time.Time             `json:"last_updated"`
    mu              sync.RWMutex          `json:"-"`
}

// CardRelationship represents how cards relate to concepts
type CardRelationship struct {
    // Map from concept ID to card IDs that teach this concept
    ConceptToCards map[string][]string `json:"concept_to_cards"`
    // Map from card ID to concept IDs it teaches
    CardToConcepts map[string][]string `json:"card_to_concepts"`
}
```

I'll also need to extend the Storage interface with these methods in `internal/storage/storage.go`:

```go
// CreateConcept creates a new concept node
func (s *FileStorage) CreateConcept(name, description string) (ConceptNode, error) {
    // Implementation here
}

// GetConcept retrieves a concept by ID
func (s *FileStorage) GetConcept(id string) (ConceptNode, error) {
    // Implementation here
}

// ListConcepts returns all concepts
func (s *FileStorage) ListConcepts() ([]ConceptNode, error) {
    // Implementation here
}

// AddPrerequisiteCard creates a prerequisite relationship between cards
func (s *FileStorage) AddPrerequisiteCard(prerequisiteID, dependentID string) error {
    // Implementation here
}

// GetCardPrerequisites returns all prerequisite cards for a given card
func (s *FileStorage) GetCardPrerequisites(cardID string) ([]Card, error) {
    // Implementation here
}

// GetCardDependents returns all cards that depend on the given card
func (s *FileStorage) GetCardDependents(cardID string) ([]Card, error) {
    // Implementation here
}

// AddPrerequisiteConcept creates a prerequisite relationship between concepts
func (s *FileStorage) AddPrerequisiteConcept(prerequisiteID, dependentID string) error {
    // Implementation here
}

// GetConceptPrerequisites returns all prerequisite concepts for a given concept
func (s *FileStorage) GetConceptPrerequisites(conceptID string) ([]ConceptNode, error) {
    // Implementation here
}

// GetConceptDependents returns all concepts that depend on the given concept
func (s *FileStorage) GetConceptDependents(conceptID string) ([]ConceptNode, error) {
    // Implementation here
}
```

## Implementation Details

I should follow these guidelines for implementation:

1. **Thread Safety**: All graph operations must be thread-safe using read/write mutexes
   - Use RLock() for read operations
   - Use Lock() for write operations

2. **Data Integrity**: 
   - Validate node existence before adding edges
   - Prevent cycles in the graph (it should be a directed acyclic graph)
   - Check for duplicate edges before adding

3. **Memory Management**:
   - Initialize maps to appropriate capacity when capacity is known
   - Make defensive copies of slices returned from public methods

4. **Error Handling**:
   - Return descriptive errors with proper context
   - Use errors.New() for simple errors and fmt.Errorf() for formatted errors

5. **JSON Persistence**:
   - Ensure all added fields are properly tagged for JSON serialization
   - The mutex field should be tagged with `json:"-"` to exclude it from serialization
   - Update Load/Save methods to handle the new graph structures

6. **Backward Compatibility**:
   - Ensure the enhanced storage can still load older JSON files without graph data
   - Initialize empty graph structures when loading older data

## Success Criteria

My implementation will be considered successful if:

- [ ] All graph data structures are properly defined and documented
- [ ] FlashcardStore structure is extended with graph components
- [ ] Basic graph operations (add/remove nodes and edges) are correctly implemented
- [ ] Thread safety is ensured with proper mutex usage
- [ ] Load/Save methods correctly handle graph data persistence
- [ ] JSON serialization/deserialization works correctly for graph structures
- [ ] Older JSON files without graph data can still be loaded
- [ ] Cycle detection prevents creation of circular dependencies
- [ ] All new methods have appropriate error handling
- [ ] Unit tests achieve at least 85% code coverage for new functionality
- [ ] Integration tests confirm proper serialization/deserialization
- [ ] All tests pass without race conditions (when run with -race flag)

## Step-by-Step Implementation

1. [ ] Extend the structs in `internal/storage/storage.go`:
   - [ ] Define DirectedGraph struct
   - [ ] Define ConceptNode struct
   - [ ] Define CardRelationship struct
   - [ ] Update FlashcardStore struct to include graph fields

2. [ ] Implement DirectedGraph methods:
   - [ ] Implement NewDirectedGraph()
   - [ ] Implement AddNode()
   - [ ] Implement RemoveNode()
   - [ ] Implement AddEdge()
   - [ ] Implement RemoveEdge()
   - [ ] Implement GetPrerequisites()
   - [ ] Implement GetDependents()
   - [ ] Implement HasCycle()

3. [ ] Update the FileStorage implementation:
   - [ ] Implement CreateConcept()
   - [ ] Implement GetConcept()
   - [ ] Implement ListConcepts()
   - [ ] Implement AddPrerequisiteCard()
   - [ ] Implement GetCardPrerequisites()
   - [ ] Implement GetCardDependents()
   - [ ] Implement AddPrerequisiteConcept()
   - [ ] Implement GetConceptPrerequisites()
   - [ ] Implement GetConceptDependents()

4. [ ] Update persistence methods:
   - [ ] Update Load() to handle loading graph data
   - [ ] Update Save() to handle saving graph data
   - [ ] Ensure backward compatibility with older JSON files

5. [ ] Create test file `internal/storage/graph_test.go`:
   - [ ] Implement TestNewDirectedGraph
   - [ ] Implement TestAddNode
   - [ ] Implement TestRemoveNode
   - [ ] Implement TestAddEdge
   - [ ] Implement TestRemoveEdge
   - [ ] Implement TestGetPrerequisites
   - [ ] Implement TestGetDependents
   - [ ] Implement TestHasCycle

6. [ ] Create test file `internal/storage/concept_test.go`:
   - [ ] Implement TestCreateConcept
   - [ ] Implement TestGetConcept
   - [ ] Implement TestListConcepts
   - [ ] Implement TestAddPrerequisiteConcept
   - [ ] Implement TestGetConceptPrerequisites
   - [ ] Implement TestGetConceptDependents

7. [ ] Create test file `internal/storage/card_relationship_test.go`:
   - [ ] Implement TestAddPrerequisiteCard
   - [ ] Implement TestGetCardPrerequisites
   - [ ] Implement TestGetCardDependents

8. [ ] Update persistence tests in `internal/storage/storage_test.go`:
   - [ ] Implement TestLoadSaveWithGraphData
   - [ ] Implement TestBackwardCompatibility

9. [ ] Run tests and ensure all pass:
   - [ ] Run `go test ./internal/storage -v` to verify basic functionality
   - [ ] Run `go test ./internal/storage -race` to check for race conditions
   - [ ] Run `go test ./internal/storage -cover` to verify code coverage

## Status Update Format

After completing the implementation, I should provide a status update including:

- Files created/modified
- Structs and methods implemented
- Test results (tests passed, code coverage percentage)
- Which success criteria were met
