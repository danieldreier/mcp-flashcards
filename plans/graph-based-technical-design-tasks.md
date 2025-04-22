# Graph-Based Flashcards MCP - Implementation Tasks

## Overview

We are enhancing the Flashcards MCP with graph-based functionality in four phases:
1. Basic directed graph structure
2. Card linking and prerequisite filtering 
3. Consecutive failure detection
4. Problem decomposition

This document outlines the specific tasks for implementation.

## Phase 1: Basic Directed Graph Structure

### Task 1: Define Graph Data Structures

**Objective:** Create the core data structures needed for the directed graph functionality.

**Context:** The flashcard system needs to track relationships between cards and concepts. We'll implement a directed graph structure to represent these relationships, ensuring we can check prerequisite knowledge before presenting cards.

**Implementation:**
- Create/modify these structures in `internal/storage/storage.go`:
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

  // ConceptNode represents a knowledge concept in the graph
  type ConceptNode struct {
      ID            string    `json:"id"`
      Name          string    `json:"name"`
      Description   string    `json:"description,omitempty"`
      CreatedAt     time.Time `json:"created_at"`
  }

  // CardRelationship represents how cards relate to concepts
  type CardRelationship struct {
      // Map from concept ID to card IDs that teach this concept
      ConceptToCards map[string][]string `json:"concept_to_cards"`
      // Map from card ID to concept IDs it teaches
      CardToConcepts map[string][]string `json:"card_to_concepts"`
  }
  ```

- Update the FlashcardStore structure to include graph components:
  ```go
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
  ```

**Guidelines:**
- Ensure all fields have proper JSON tags
- Mark mutex fields with `json:"-"` to exclude from serialization
- Add appropriate comments documenting each field's purpose
- Ensure all new types are exported (begin with capital letter)

**Success Criteria:**
- All graph data structures are properly defined and documented
- FlashcardStore structure is extended with graph components
- JSON tags are correctly specified for all fields

### Task 2: Implement DirectedGraph Methods

**Objective:** Create core graph operations for adding/removing nodes and edges.

**Context:** The directed graph needs methods to add and remove nodes and edges, retrieve prerequisite and dependent nodes, and detect cycles to ensure the graph remains acyclic.

**Implementation:**
Implement these methods in `internal/storage/storage.go`:

```go
// NewDirectedGraph creates a new empty directed graph
func NewDirectedGraph() *DirectedGraph {
    return &DirectedGraph{
        Prerequisites: make(map[string][]string),
        Dependents:    make(map[string][]string),
    }
}

// AddNode adds a new node to the graph
func (g *DirectedGraph) AddNode(nodeID string) {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // Initialize empty arrays if node doesn't exist
    if _, exists := g.Prerequisites[nodeID]; !exists {
        g.Prerequisites[nodeID] = []string{}
    }
    if _, exists := g.Dependents[nodeID]; !exists {
        g.Dependents[nodeID] = []string{}
    }
}

// RemoveNode removes a node and all its edges from the graph
func (g *DirectedGraph) RemoveNode(nodeID string) {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // Remove all edges where this node is a prerequisite
    for dependent, _ := range g.Dependents[nodeID] {
        g.removePrerequisiteUnsafe(nodeID, dependent)
    }
    
    // Remove all edges where this node is dependent
    for prereq, _ := range g.Prerequisites[nodeID] {
        g.removePrerequisiteUnsafe(prereq, nodeID)
    }
    
    // Delete the node
    delete(g.Prerequisites, nodeID)
    delete(g.Dependents, nodeID)
}

// AddEdge creates a directed edge from prerequisite to dependent
func (g *DirectedGraph) AddEdge(prerequisiteID, dependentID string) error {
    // Implementation with proper checks and cycle detection
}

// RemoveEdge removes a directed edge from prerequisite to dependent
func (g *DirectedGraph) RemoveEdge(prerequisiteID, dependentID string) error {
    // Implementation with proper checks
}

// GetPrerequisites returns all prerequisites for a given node
func (g *DirectedGraph) GetPrerequisites(nodeID string) []string {
    // Implementation with thread safety and defensive copy
}

// GetDependents returns all nodes that depend on the given node
func (g *DirectedGraph) GetDependents(nodeID string) []string {
    // Implementation with thread safety and defensive copy
}

// HasCycle detects if the graph contains any cycles
func (g *DirectedGraph) HasCycle() bool {
    // Implementation with depth-first search
}
```

**Guidelines:**
- Ensure thread safety using RWMutex:
  - Use RLock() for read operations (GetPrerequisites, GetDependents, HasCycle)
  - Use Lock() for write operations (AddNode, RemoveNode, AddEdge, RemoveEdge)
- Validate inputs:
  - Check that nodes exist before adding an edge
  - Check for duplicate edges before adding
  - Return appropriate errors for invalid operations
- Maintain graph integrity:
  - When removing a node, clean up all references to it
  - Prevent cycles (the graph should be a directed acyclic graph)
- Make defensive copies of slices returned from public methods
- Use errors.New() for simple errors and fmt.Errorf() for formatted errors

**Success Criteria:**
- All DirectedGraph methods are correctly implemented
- Thread safety is ensured with proper mutex usage
- Cycle detection prevents creation of circular dependencies
- All operations maintain graph integrity
- Methods return appropriate errors for invalid operations

### Task 3: Implement Concept and Relationship Methods

**Objective:** Implement the concept management and card-concept relationship functionality.

**Context:** We need to manage knowledge concepts and their relationships to cards in the graph-based flashcard system.

**Implementation:**
Implement these methods in `internal/storage/storage.go`:

```go
// CreateConcept creates a new concept node
func (s *FileStorage) CreateConcept(name, description string) (ConceptNode, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Create concept with unique ID
    concept := ConceptNode{
        ID:          uuid.New().String(),
        Name:        name,
        Description: description,
        CreatedAt:   time.Now(),
    }
    
    // Add to storage
    s.store.Concepts[concept.ID] = concept
    
    // Add node to concept graph
    s.store.ConceptGraph.AddNode(concept.ID)
    
    s.store.LastUpdated = time.Now()
    return concept, nil
}

// GetConcept retrieves a concept by ID
func (s *FileStorage) GetConcept(id string) (ConceptNode, error) {
    // Implementation with proper error handling
}

// ListConcepts returns all concepts
func (s *FileStorage) ListConcepts() ([]ConceptNode, error) {
    // Implementation with thread safety
}

// AssociateCardWithConcept links a card to a concept
func (s *FileStorage) AssociateCardWithConcept(cardID, conceptID string) error {
    // Implementation with proper validation and error handling
}

// GetConceptCards returns all cards associated with a concept
func (s *FileStorage) GetConceptCards(conceptID string) ([]Card, error) {
    // Implementation with error handling
}

// GetCardConcepts returns all concepts associated with a card
func (s *FileStorage) GetCardConcepts(cardID string) ([]ConceptNode, error) {
    // Implementation with error handling
}
```

**Guidelines:**
- Ensure thread safety with proper mutex usage
- Validate that cards and concepts exist before associating them
- Return descriptive errors for invalid operations
- Make defensive copies of returned slices
- Update LastUpdated timestamp after modifications

**Success Criteria:**
- All concept methods are correctly implemented
- Card-concept relationships are properly tracked
- Thread safety is ensured
- All operations have proper error handling

### Task 4: Implement Prerequisite Relationship Methods

**Objective:** Implement methods to manage prerequisite relationships between cards and concepts.

**Context:** The graph-based flashcard system needs to track which cards/concepts are prerequisites for others.

**Implementation:**
Implement these methods in `internal/storage/storage.go`:

```go
// AddPrerequisiteCard creates a prerequisite relationship between cards
func (s *FileStorage) AddPrerequisiteCard(prerequisiteID, dependentID string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Validate card existence
    if _, exists := s.store.Cards[prerequisiteID]; !exists {
        return fmt.Errorf("prerequisite card %s does not exist", prerequisiteID)
    }
    if _, exists := s.store.Cards[dependentID]; !exists {
        return fmt.Errorf("dependent card %s does not exist", dependentID)
    }
    
    // Add edge to graph
    if err := s.store.CardGraph.AddEdge(prerequisiteID, dependentID); err != nil {
        return err
    }
    
    s.store.LastUpdated = time.Now()
    return nil
}

// GetCardPrerequisites returns all prerequisite cards for a given card
func (s *FileStorage) GetCardPrerequisites(cardID string) ([]Card, error) {
    // Implementation with error handling
}

// GetCardDependents returns all cards that depend on the given card
func (s *FileStorage) GetCardDependents(cardID string) ([]Card, error) {
    // Implementation with error handling
}

// AddPrerequisiteConcept creates a prerequisite relationship between concepts
func (s *FileStorage) AddPrerequisiteConcept(prerequisiteID, dependentID string) error {
    // Implementation with validation and error handling
}

// GetConceptPrerequisites returns all prerequisite concepts for a given concept
func (s *FileStorage) GetConceptPrerequisites(conceptID string) ([]ConceptNode, error) {
    // Implementation with error handling
}

// GetConceptDependents returns all concepts that depend on the given concept
func (s *FileStorage) GetConceptDependents(conceptID string) ([]ConceptNode, error) {
    // Implementation with error handling
}
```

**Guidelines:**
- Validate that cards/concepts exist before creating relationships
- Ensure operations don't create cycles in the graph
- Update LastUpdated timestamp after modifications
- Properly handle error conditions

**Success Criteria:**
- All prerequisite relationship methods are correctly implemented
- Graph integrity is maintained (no cycles)
- Thread safety is ensured
- All operations have proper error handling

### Task 5: Update Storage Persistence

**Objective:** Update Load and Save methods to handle the new graph-based data structures.

**Context:** The flashcard system's persistence layer needs to be enhanced to store and load graph-based structures.

**Implementation:**
Update these methods in `internal/storage/storage.go`:

```go
// Load loads flashcard data from a JSON file
func (s *FileStorage) Load() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Read file
    data, err := os.ReadFile(s.filePath)
    if err != nil {
        if os.IsNotExist(err) {
            // Initialize with empty structures if file doesn't exist
            s.store = &FlashcardStore{
                Cards:         make(map[string]Card),
                Reviews:       []Review{},
                Concepts:      make(map[string]ConceptNode),
                CardGraph:     *NewDirectedGraph(),
                ConceptGraph:  *NewDirectedGraph(),
                Relationships: CardRelationship{
                    ConceptToCards: make(map[string][]string),
                    CardToConcepts: make(map[string][]string),
                },
                LastUpdated:   time.Now(),
            }
            return nil
        }
        return err
    }
    
    // Unmarshal JSON
    var store FlashcardStore
    if err := json.Unmarshal(data, &store); err != nil {
        return err
    }
    
    // Handle backward compatibility for older files without graph structures
    if store.Concepts == nil {
        store.Concepts = make(map[string]ConceptNode)
    }
    if store.CardGraph.Prerequisites == nil || store.CardGraph.Dependents == nil {
        store.CardGraph = *NewDirectedGraph()
    }
    if store.ConceptGraph.Prerequisites == nil || store.ConceptGraph.Dependents == nil {
        store.ConceptGraph = *NewDirectedGraph()
    }
    if store.Relationships.ConceptToCards == nil || store.Relationships.CardToConcepts == nil {
        store.Relationships = CardRelationship{
            ConceptToCards: make(map[string][]string),
            CardToConcepts: make(map[string][]string),
        }
    }
    
    s.store = &store
    return nil
}

// Save saves flashcard data to a JSON file
func (s *FileStorage) Save() error {
    // Implementation with proper file writing and error handling
}
```

**Guidelines:**
- Ensure backward compatibility with older files
- Initialize empty structures for missing components
- Use atomic write pattern for Save() to prevent data corruption
- Properly handle file I/O errors

**Success Criteria:**
- Load() and Save() methods handle graph structures correctly
- Backward compatibility with older files is maintained
- Thread safety is ensured during I/O operations
- Data integrity is preserved with atomic writes

### Task 6: Implement Unit Tests

**Objective:** Create comprehensive tests for the graph functionality.

**Context:** To ensure the graph-based enhancements work correctly, we need thorough tests for all operations.

**Implementation:**
Create test files:
1. `internal/storage/graph_test.go` - Tests for DirectedGraph
2. `internal/storage/concept_test.go` - Tests for concept operations
3. `internal/storage/card_relationship_test.go` - Tests for card-concept relationships
4. Update `internal/storage/storage_test.go` - Tests for persistence

Example test for DirectedGraph:
```go
func TestDirectedGraph_AddEdge(t *testing.T) {
    g := NewDirectedGraph()
    
    // Add nodes
    g.AddNode("A")
    g.AddNode("B")
    
    // Add edge
    err := g.AddEdge("A", "B")
    if err != nil {
        t.Fatalf("Failed to add edge: %v", err)
    }
    
    // Verify prerequisite relationship
    prereqs := g.GetPrerequisites("B")
    if len(prereqs) != 1 || prereqs[0] != "A" {
        t.Errorf("Expected B to have A as prerequisite, got %v", prereqs)
    }
    
    // Verify dependent relationship
    deps := g.GetDependents("A")
    if len(deps) != 1 || deps[0] != "B" {
        t.Errorf("Expected A to have B as dependent, got %v", deps)
    }
}

func TestDirectedGraph_CycleDetection(t *testing.T) {
    g := NewDirectedGraph()
    
    // Add nodes
    g.AddNode("A")
    g.AddNode("B")
    g.AddNode("C")
    
    // Add edges
    _ = g.AddEdge("A", "B")
    _ = g.AddEdge("B", "C")
    
    // Attempt to create cycle
    err := g.AddEdge("C", "A")
    if err == nil {
        t.Fatal("Expected error when creating cycle, got nil")
    }
    
    // Verify no cycle was created
    if g.HasCycle() {
        t.Error("Graph should not have a cycle")
    }
}
```

**Guidelines:**
- Test both valid and invalid operations
- Verify graph integrity after operations
- Test cycle detection functionality
- Test thread safety with concurrent operations
- Test backward compatibility with older data formats

**Success Criteria:**
- Tests cover all graph operations
- Tests verify proper error handling
- Tests confirm thread safety
- Tests ensure backward compatibility
- All tests pass, with at least 85% code coverage
