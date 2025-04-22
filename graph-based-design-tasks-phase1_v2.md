# Knowledge Graph-Based Flashcard System Implementation Plan - Phase 1 (Core Infrastructure)

This document outlines the implementation plan for the Knowledge Graph-Based Flashcard System, focusing on Phase 1: Core Infrastructure. Each task is designed to be implemented as a standalone unit with comprehensive specifications.

**IMPORTANT:** Tests must be executed between tasks, not just at the end of the phase. Each task should include explicit test steps throughout its implementation to ensure incremental validation. Do not proceed to the next task until all tests for the current task pass.

## Implementation Plan for Core Infrastructure

### Phase 1.1: Knowledge Graph Foundation

#### Task 1.1.1: Implement Core Knowledge Graph Data Structures

```
# Task 1.1.1: Implement Core Knowledge Graph Data Structures

## Background and Context
The existing flashcard system uses a simple model where cards are treated as independent entities with no relationships between them. To implement a prerequisite-first approach to learning, we need to create a knowledge graph structure that represents concepts and their dependencies. This will allow us to ensure students master prerequisites before encountering dependent concepts.

## My Task
Implement the core data structures needed for the knowledge graph, including concepts, dependencies, and bloom level classifications. These structures will form the foundation of our knowledge graph approach to spaced repetition.

## Files to Modify
1. `internal/graph/models.go`: Create new file for knowledge graph data structures
2. `internal/graph/graph.go`: Create new file for graph operations

## Implementation Details
In `internal/graph/models.go`:
1. Define key data structures for the knowledge graph:
   ```go
   package graph

   import (
      "time"
      "github.com/google/uuid"
      "github.com/open-spaced-repetition/go-fsrs"
   )

   // BloomLevel represents cognitive complexity levels based on Bloom's Taxonomy
   type BloomLevel string

   // Bloom's Taxonomy cognitive levels
   const (
      BloomRemember   BloomLevel = "remember"
      BloomUnderstand BloomLevel = "understand"
      BloomApply      BloomLevel = "apply"
      BloomAnalyze    BloomLevel = "analyze"
      BloomEvaluate   BloomLevel = "evaluate"
      BloomCreate     BloomLevel = "create"
   )

   // Concept represents a discrete learning topic in the knowledge graph
   type Concept struct {
      ID              string                      `json:"id"`
      Name            string                      `json:"name"`
      Description     string                      `json:"description"`
      BloomCards      map[BloomLevel][]string     `json:"bloom_cards"` // Card IDs by Bloom level
      PrerequisiteIDs []string                    `json:"prerequisite_ids"`
      DependentIDs    []string                    `json:"dependent_ids"`
      Created         time.Time                   `json:"created"`
      LastModified    time.Time                   `json:"last_modified"`
   }

   // Dependency represents a prerequisite relationship between concepts
   type Dependency struct {
      FromID          string     `json:"from_id"`  // Prerequisite concept
      ToID            string     `json:"to_id"`    // Dependent concept
      Strength        float64    `json:"strength"` // 0-1 importance of relationship
      Confidence      float64    `json:"confidence"` // 0-1 confidence in relationship
      Verified        bool       `json:"verified"` // Confirmed through interactions
   }

   // MasteryCriteria defines when a concept is considered mastered
   type MasteryCriteria struct {
      MinimumRating   float64    `json:"minimum_rating"`    // Minimum average rating (default 3.0)
      MinimumReviews  int        `json:"minimum_reviews"`   // Minimum number of reviews
      RequiredLevels  []BloomLevel `json:"required_levels"` // Which Bloom levels must be mastered
   }

   // StudentConceptState tracks a student's progress with a concept
   type StudentConceptState struct {
      StudentID       string                      `json:"student_id"`
      ConceptID       string                      `json:"concept_id"`
      MasteryLevel    float64                     `json:"mastery_level"`
      ReviewCount     int                         `json:"review_count"`
      LastReviewed    time.Time                   `json:"last_reviewed"`
      LevelMastery    map[BloomLevel]float64      `json:"level_mastery"`
   }
   ```

2. Extend the existing Card model to incorporate knowledge graph elements:
   ```go
   // FlashCard extends the existing card model with knowledge graph elements
   type FlashCard struct {
      ID              string     `json:"id"`
      ConceptID       string     `json:"concept_id"`
      BloomLevel      BloomLevel `json:"bloom_level"`
      Question        string     `json:"question"`
      Answer          string     `json:"answer"`
      
      // FSRS parameters
      Difficulty      float64    `json:"difficulty"`
      Stability       float64    `json:"stability"`
      Retrievability  float64    `json:"retrievability"`
      LastReviewed    time.Time  `json:"last_reviewed"`
      DueDate         time.Time  `json:"due_date"`
      State           fsrs.State `json:"state"`
   }
   ```

In `internal/graph/graph.go`:
1. Define the KnowledgeGraph interface and basic implementation:
   ```go
   package graph

   import (
      "errors"
      "time"
   )

   var (
      ErrConceptNotFound     = errors.New("concept not found")
      ErrDependencyNotFound  = errors.New("dependency not found")
      ErrInvalidDependency   = errors.New("invalid dependency")
      ErrCircularDependency  = errors.New("circular dependency detected")
   )

   // KnowledgeGraph defines operations for the concept knowledge graph
   type KnowledgeGraph interface {
      // Concept operations
      AddConcept(name, description string) (*Concept, error)
      GetConcept(id string) (*Concept, error)
      UpdateConcept(concept *Concept) error
      DeleteConcept(id string) error
      ListConcepts() ([]*Concept, error)
      
      // Dependency operations
      AddDependency(fromID, toID string, strength float64) (*Dependency, error)
      GetDependency(fromID, toID string) (*Dependency, error)
      UpdateDependency(dependency *Dependency) error
      DeleteDependency(fromID, toID string) error
      ListDependencies() ([]*Dependency, error)
      
      // Graph traversal
      GetPrerequisites(conceptID string, recursive bool) ([]*Concept, error)
      GetDependents(conceptID string, recursive bool) ([]*Concept, error)
      VerifyNoCycles() error
      
      // Card operations
      AddCardToConcept(card *FlashCard, conceptID string, level BloomLevel) error
      GetConceptCards(conceptID string, level *BloomLevel) ([]*FlashCard, error)
   }

   // MemoryKnowledgeGraph provides an in-memory implementation of the knowledge graph
   type MemoryKnowledgeGraph struct {
      Concepts    map[string]*Concept
      Dependencies map[string]map[string]*Dependency // fromID -> toID -> Dependency
      Cards       map[string]*FlashCard
   }

   // NewMemoryKnowledgeGraph creates a new in-memory knowledge graph
   func NewMemoryKnowledgeGraph() *MemoryKnowledgeGraph {
      return &MemoryKnowledgeGraph{
         Concepts:     make(map[string]*Concept),
         Dependencies: make(map[string]map[string]*Dependency),
         Cards:        make(map[string]*FlashCard),
      }
   }

   // AddConcept adds a new concept to the graph
   func (g *MemoryKnowledgeGraph) AddConcept(name, description string) (*Concept, error) {
      // Implementation required
      return nil, nil
   }

   // Additional method implementations would follow
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Concept creation and retrieval - Ensures concepts can be added to the graph and retrieved by ID
2. Dependency creation and validation - Verifies prerequisite relationships can be established and validated
3. Circular dependency detection - Checks that circular dependencies are detected and prevented
4. Bloom level card association - Tests adding cards to concepts at specific Bloom levels
5. Graph traversal - Validates that prerequisites and dependents can be correctly traversed

## Success Criteria
- [ ] [Concept operations] Creating, retrieving, updating, and deleting concepts works as expected [TestConceptOperations]
- [ ] [Dependency operations] Prerequisites can be added, retrieved, and validated [TestDependencyOperations]
- [ ] [Cycle detection] Circular dependencies are detected and prevented [TestCycleDetection]
- [ ] [Card management] Cards can be associated with concepts at specific Bloom levels [TestCardAssociation]
- [ ] [Graph traversal] Recursive and non-recursive traversal of prerequisites and dependents works correctly [TestGraphTraversal]

## Step-by-Step Implementation
1. [ ] Create the `internal/graph` directory
2. [ ] Create `models.go` with the core data structures
3. [ ] Write tests for data structure validation
4. [ ] Run tests to verify data structures (`go test ./internal/graph -run=TestDataStructures`)
5. [ ] Create `graph.go` with the interface and basic implementation
6. [ ] Implement core concept operations (add, get, update, delete)
7. [ ] Write tests for concept operations
8. [ ] Run tests to verify concept operations (`go test ./internal/graph -run=TestConceptOperations`)
9. [ ] Implement dependency operations (add, get, update, delete)
10. [ ] Write tests for dependency operations including cycle detection
11. [ ] Run tests to verify dependency operations (`go test ./internal/graph -run=TestDependencyOperations`)
12. [ ] Implement graph traversal methods
13. [ ] Write tests for graph traversal
14. [ ] Run tests to verify graph traversal (`go test ./internal/graph -run=TestGraphTraversal`)
15. [ ] Implement card association methods
16. [ ] Write tests for card association methods
17. [ ] Run tests to verify card association (`go test ./internal/graph -run=TestCardAssociation`)
18. [ ] Ensure all tests pass for the complete task (`go test ./internal/graph`)
19. [ ] Run linters and static code analysis
```

#### Task 1.1.2: Implement Graph Persistence Layer

```
# Task 1.1.2: Implement Graph Persistence Layer

## Background and Context
The knowledge graph data structures need to be persisted so that the learning graph can be maintained between sessions. The existing flashcard system uses JSON file storage for cards and review history. We need to extend this approach to store and retrieve the knowledge graph structures.

## My Task
Create a persistence layer for the knowledge graph that integrates with the existing storage system. This will allow the graph structure to be saved to and loaded from disk, ensuring continuity between user sessions.

## Files to Modify
1. `internal/graph/storage.go`: Create new file for knowledge graph persistence
2. `internal/storage/storage.go`: Modify existing storage to accommodate knowledge graph

## Implementation Details
In `internal/graph/storage.go`:
1. Implement a storage interface for the knowledge graph:
   ```go
   package graph

   import (
      "encoding/json"
      "errors"
      "os"
      "path/filepath"
      "time"
   )

   // GraphStore defines persistence operations for the knowledge graph
   type GraphStore interface {
      // Save the entire graph to persistence
      SaveGraph(graph KnowledgeGraph) error
      
      // Load the graph from persistence
      LoadGraph() (KnowledgeGraph, error)
      
      // Update specific elements without rewriting the entire graph
      SaveConcept(concept *Concept) error
      SaveDependency(dependency *Dependency) error
      
      // Path management
      GetStoragePath() string
      SetStoragePath(path string) error
   }

   // FileGraphStore provides file-based persistence for the knowledge graph
   type FileGraphStore struct {
      FilePath string
   }

   // NewFileGraphStore creates a new file-based graph store
   func NewFileGraphStore(path string) *FileGraphStore {
      return &FileGraphStore{
         FilePath: path,
      }
   }

   // GraphData represents the serializable form of the knowledge graph
   type GraphData struct {
      Concepts     []*Concept    `json:"concepts"`
      Dependencies []*Dependency `json:"dependencies"`
      LastModified time.Time     `json:"last_modified"`
   }
   ```

2. Implement core persistence methods:
   ```go
   // SaveGraph persists the entire graph to a JSON file
   func (s *FileGraphStore) SaveGraph(graph KnowledgeGraph) error {
      // Implementation required
      return nil
   }

   // LoadGraph loads the graph from a JSON file
   func (s *FileGraphStore) LoadGraph() (KnowledgeGraph, error) {
      // Implementation required
      return nil, nil
   }

   // SaveConcept updates a single concept in the persisted graph
   func (s *FileGraphStore) SaveConcept(concept *Concept) error {
      // Implementation required
      return nil
   }

   // SaveDependency updates a single dependency in the persisted graph
   func (s *FileGraphStore) SaveDependency(dependency *Dependency) error {
      // Implementation required
      return nil
   }
   ```

Modify `internal/storage/storage.go`:
1. Extend the existing storage model to include knowledge graph data:
   ```go
   // ExtendedFlashcardStore represents the data persisted to file including graph
   type ExtendedFlashcardStore struct {
      Cards        map[string]Card     `json:"cards"`
      Reviews      []ReviewRecord      `json:"reviews"`
      Concepts     []*graph.Concept    `json:"concepts"`
      Dependencies []*graph.Dependency `json:"dependencies"`
      LastModified time.Time           `json:"last_modified"`
   }
   ```

2. Update the load and save methods to handle the extended data model:
   ```go
   // loadFromFile loads flashcard and graph data from a JSON file
   func (s *ExtendedFlashcardService) LoadFromFile() error {
      // Implementation to load both cards and graph data
      return nil
   }

   // saveToFile saves flashcard and graph data to a JSON file
   func (s *ExtendedFlashcardService) SaveToFile() error {
      // Implementation to save both cards and graph data
      return nil
   }
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Graph serialization - Tests that the entire graph can be correctly serialized to JSON
2. Graph deserialization - Verifies that a graph can be correctly loaded from JSON
3. Incremental updates - Checks that individual concepts and dependencies can be updated
4. Error handling - Tests graceful handling of I/O errors and invalid data
5. Integration with existing storage - Verifies compatibility with the existing flashcard storage
6. Demonstrate end-to-end save/retrieve still working via MCP

## Success Criteria
- [ ] [Graph serialization] Complete graph correctly saves to JSON [TestGraphSerialization]
- [ ] [Graph deserialization] Complete graph correctly loads from JSON [TestGraphDeserialization]
- [ ] [Incremental updates] Individual concepts/dependencies can be updated [TestIncrementalUpdates]
- [ ] [Error handling] Storage gracefully handles file errors and corrupted data [TestStorageErrorHandling]
- [ ] [Integration] Graph storage works alongside existing flashcard storage [TestStorageIntegration]

## Step-by-Step Implementation
1. [ ] Create `internal/graph/storage.go` with the GraphStore interface
2. [ ] Implement FileGraphStore with basic serialization methods
3. [ ] Write tests for graph serialization
4. [ ] Run tests to verify serialization (`go test ./internal/graph -run=TestGraphSerialization`)
5. [ ] Implement graph loading functionality
6. [ ] Write tests for graph deserialization
7. [ ] Run tests to verify deserialization (`go test ./internal/graph -run=TestGraphDeserialization`)
8. [ ] Implement incremental update methods
9. [ ] Write tests for incremental updates
10. [ ] Run tests to verify incremental updates (`go test ./internal/graph -run=TestIncrementalUpdates`)
11. [ ] Implement error handling for storage operations
12. [ ] Write tests for error handling scenarios
13. [ ] Run tests to verify error handling (`go test ./internal/graph -run=TestStorageErrorHandling`)
14. [ ] Modify `internal/storage/storage.go` to integrate with graph storage
15. [ ] Write tests for storage integration
16. [ ] Run tests to verify storage integration (`go test ./internal/storage -run=TestStorageIntegration`)
17. [ ] Ensure all tests pass for the complete task (`go test ./internal/graph ./internal/storage`)
18. [ ] Run linters and static code analysis
```

### Phase 1.2: FSRS Integration

#### Task 1.2.1: Extend FSRS for Knowledge Graph Awareness

```
# Task 1.2.1: Extend FSRS for Knowledge Graph Awareness

## Background and Context
The Free Spaced Repetition Scheduler (FSRS) algorithm optimizes review timing based on memory characteristics but doesn't account for concept relationships. In our knowledge graph system, we need to extend FSRS to consider prerequisites and concept dependencies when scheduling cards for review.

## My Task
Enhance the FSRS integration to be knowledge graph-aware, ensuring that review scheduling respects prerequisite relationships and adjusts priorities based on a concept's position in the graph.

## Files to Modify
1. `internal/graph/scheduler.go`: Create new file for graph-aware scheduling logic
2. `internal/fsrs/fsrs.go`: Extend the existing FSRS implementation

## Implementation Details
In `internal/graph/scheduler.go`:
1. Define interfaces and structures for graph-aware scheduling:
   ```go
   package graph

   import (
      "time"
      "github.com/open-spaced-repetition/go-fsrs"
   )

   // GraphScheduler extends FSRS with knowledge graph awareness
   type GraphScheduler interface {
      // Schedule a card's next review considering its position in the graph
      ScheduleReview(card *FlashCard, rating fsrs.Rating, graph KnowledgeGraph) (fsrs.State, time.Time, error)
      
      // Get due cards that respect prerequisite relationships
      GetDueCards(graph KnowledgeGraph, studentID string, now time.Time) ([]*FlashCard, error)
      
      // Calculate priority for a card based on graph position and memory state
      CalculateCardPriority(card *FlashCard, graph KnowledgeGraph, studentID string) (float64, error)
      
      // Get the next card for review that respects prerequisites
      GetNextCardWithPrerequisites(graph KnowledgeGraph, studentID string, now time.Time) (*FlashCard, error)
   }

   // FSRSGraphScheduler implements graph-aware scheduling using FSRS
   type FSRSGraphScheduler struct {
      Params fsrs.Parameters
      PrerequisitesMastered map[string]bool // Cache for prerequisite mastery status
   }

   // MasteryChecker defines how concept mastery is determined
   type MasteryChecker interface {
      // Check if a concept is mastered by a student
      IsConceptMastered(conceptID string, studentID string, masteryThreshold float64) (bool, error)
      
      // Get student's mastery level for a concept
      GetConceptMasteryLevel(conceptID string, studentID string) (float64, error)
      
      // Update concept mastery level based on a review
      UpdateConceptMastery(conceptID string, studentID string, bloomLevel BloomLevel, rating fsrs.Rating) error
   }
   ```

2. Implement the scheduling methods with specific business logic:
   ```go
   // NewFSRSGraphScheduler creates a new graph-aware scheduler
   func NewFSRSGraphScheduler(params fsrs.Parameters) *FSRSGraphScheduler {
      return &FSRSGraphScheduler{
         Params: params,
         PrerequisitesMastered: make(map[string]bool),
      }
   }

   // ScheduleReview schedules the next review for a card
   // NOTE: Use standard FSRS scheduling without graph-based interval adjustments
   func (s *FSRSGraphScheduler) ScheduleReview(card *FlashCard, rating fsrs.Rating, graph KnowledgeGraph) (fsrs.State, time.Time, error) {
      // Implementation should use standard FSRS algorithm without interval modifications
      // Implementation required
      return fsrs.State{}, time.Time{}, nil
   }

   // GetDueCards gets cards due for review that respect prerequisites
   // NOTE: Filter out cards whose prerequisites are not mastered (threshold-based)
   func (s *FSRSGraphScheduler) GetDueCards(graph KnowledgeGraph, studentID string, now time.Time) ([]*FlashCard, error) {
      // Implementation should:
      // 1. Get all due cards based on standard FSRS scheduling
      // 2. Filter out cards whose prerequisites have not reached mastery threshold
      // Implementation required
      return nil, nil
   }

   // CalculateCardPriority calculates a priority score for a card
   // NOTE: Use depth-based priority, prioritizing deepest concepts first
   func (s *FSRSGraphScheduler) CalculateCardPriority(card *FlashCard, graph KnowledgeGraph, studentID string) (float64, error) {
      // Implementation should prioritize:
      // 1. Cards from concepts at deeper levels in the graph
      // 2. Cards that students have previously struggled with
      // 3. Cards that are prerequisites for multiple dependent concepts
      // Implementation required
      return 0, nil
   }
   ```

In `internal/fsrs/fsrs.go`:
1. Extend the FSRS manager to work with knowledge graph concepts:
   ```go
   // FSRSWithGraph extends basic FSRS with graph awareness
   type FSRSWithGraph struct {
      Params fsrs.Parameters
   }

   // CalculateNextStateWithGraph calculates next state using standard FSRS
   // NOTE: No graph-based interval adjustments needed
   func (f *FSRSWithGraph) CalculateNextStateWithGraph(
      difficulty float64,
      stability float64,
      rating fsrs.Rating,
   ) fsrs.State {
      // Implementation should use standard FSRS calculation
      // Implementation required
      return fsrs.State{}
   }
   ```

3. Implement decay-aware, bloom-weighted mastery tracking:
   ```go
   // MasteryTracker implements decay-aware, bloom-weighted mastery tracking
   type MasteryTracker struct {
      // Decay parameters
      MasteryDecayRate float64  // How quickly mastery decays over time
      LastReviewWindow time.Duration // Window for considering reviews "recent"
      
      // Bloom level weights (higher levels have higher weights)
      BloomLevelWeights map[BloomLevel]float64
   }
   
   // NewMasteryTracker creates a tracker with default settings
   func NewMasteryTracker() *MasteryTracker {
      return &MasteryTracker{
         MasteryDecayRate: 0.05, // 5% decay per day without review
         LastReviewWindow: 30 * 24 * time.Hour, // 30 days
         BloomLevelWeights: map[BloomLevel]float64{
            BloomRemember:   1.0,
            BloomUnderstand: 1.2,
            BloomApply:      1.5,
            BloomAnalyze:    1.8,
            BloomEvaluate:   2.0,
            BloomCreate:     2.5,
         },
      }
   }
   
   // CalculateMastery calculates concept mastery with time decay and bloom weighting
   func (m *MasteryTracker) CalculateMastery(
      reviews []ReviewRecord,
      bloomLevels []BloomLevel,
      now time.Time,
   ) float64 {
      // Implementation should:
      // 1. Apply time-based decay to older reviews
      // 2. Weight reviews by Bloom's taxonomy level
      // 3. Calculate weighted average across all relevant reviews
      // Implementation required
      return 0.0
   }
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Prerequisite enforcement - Verifies that cards whose prerequisites aren't mastered are excluded from due cards (using threshold-based enforcement)
2. Priority calculation - Tests that cards are prioritized based on depth in the graph, with deeper concepts prioritized first
3. Mastery tracking - Validates that concept mastery is correctly tracked with time-based decay and Bloom level weighting
4. Next card selection - Tests that the next card selection respects prerequisites and uses standard FSRS for timing
5. Mastery threshold validation - Verifies that the mastery threshold correctly identifies mastered concepts

## Success Criteria
- [ ] [Prerequisite enforcement] Cards with unmastered prerequisites (below threshold) are excluded from due cards [TestPrerequisiteEnforcement]
- [ ] [Priority calculation] Cards are prioritized correctly based on graph depth, with deeper concepts prioritized first [TestPriorityCalculation]
- [ ] [Mastery tracking] Concept mastery decays over time and is weighted by Bloom level [TestMasteryTracking]
- [ ] [Mastery decay] Mastery levels decrease appropriately when concepts aren't reviewed [TestMasteryDecay]
- [ ] [Next card selection] GetNextCardWithPrerequisites returns the highest priority valid card that respects prerequisites [TestNextCardSelection]

## Step-by-Step Implementation
1. [ ] Create `internal/graph/scheduler.go` with GraphScheduler interface
2. [ ] Implement basic FSRSGraphScheduler structure
3. [ ] Implement MasteryChecker interface with decay-aware, bloom-weighted logic
4. [ ] Write tests for mastery checking and decay functions
5. [ ] Run tests to verify mastery checking and decay (`go test ./internal/graph -run=TestMasteryChecking`)
6. [ ] Implement ScheduleReview using standard FSRS scheduling
7. [ ] Write tests for review scheduling
8. [ ] Run tests to verify review scheduling (`go test ./internal/graph -run=TestReviewScheduling`)
9. [ ] Implement GetDueCards with threshold-based prerequisite filtering
10. [ ] Write tests for due card filtering
11. [ ] Run tests to verify due card filtering (`go test ./internal/graph -run=TestDueCardFiltering`)
12. [ ] Implement CalculateCardPriority with depth-based prioritization
13. [ ] Write tests for priority calculation
14. [ ] Run tests to verify priority calculation (`go test ./internal/graph -run=TestPriorityCalculation`)
15. [ ] Implement GetNextCardWithPrerequisites for selecting cards
16. [ ] Write tests for next card selection
17. [ ] Run tests to verify next card selection (`go test ./internal/graph -run=TestNextCardSelection`)
18. [ ] Extend fsrs.go for compatibility with graph scheduler
19. [ ] Write tests for extended FSRS functions
20. [ ] Run tests to verify extended FSRS functions (`go test ./internal/fsrs -run=TestFSRSWithGraph`)
21. [ ] Ensure all tests pass for the complete task (`go test ./internal/graph ./internal/fsrs`)
22. [ ] Run linters and static code analysis
```

#### Task 1.2.2: Implement Concept Mastery Tracking

```
# Task 1.2.2: Implement Concept Mastery Tracking

## Background and Context
In a knowledge graph system, we need to track not just individual card memory states but also overall concept mastery. This allows us to properly enforce prerequisite relationships and determine when students are ready to progress to dependent concepts.

## My Task
Implement a concept mastery tracking system that aggregates individual card reviews to determine overall concept mastery levels across different Bloom's Taxonomy cognitive levels.

## Files to Modify
1. `internal/graph/mastery.go`: Create new file for mastery tracking logic
2. `internal/graph/student.go`: Create new file for student-specific state

## Implementation Details
In `internal/graph/mastery.go`:
1. Define the mastery tracking interface and implementation:
   ```go
   package graph

   import (
      "errors"
      "time"
      "github.com/open-spaced-repetition/go-fsrs"
   )

   // DefaultMasteryThreshold defines the rating threshold for mastery (3.0 or better)
   const DefaultMasteryThreshold = 3.0

   // MasteryTracker manages concept mastery across students
   type MasteryTracker interface {
      // Check if a concept is mastered by a student
      IsConceptMastered(conceptID, studentID string) (bool, error)
      
      // Get student's mastery level for a concept
      GetMasteryLevel(conceptID, studentID string) (float64, error)
      
      // Update concept mastery based on a card review
      UpdateMastery(conceptID, cardID, studentID string, bloomLevel BloomLevel, rating fsrs.Rating) error
      
      // Get all mastered concepts for a student
      GetMasteredConcepts(studentID string) ([]string, error)
      
      // Get mastery status by Bloom level for a concept
      GetBloomLevelMastery(conceptID, studentID string) (map[BloomLevel]float64, error)
   }

   // MemoryMasteryTracker tracks mastery in memory
   type MemoryMasteryTracker struct {
      // Map of studentID -> conceptID -> StudentConceptState
      StudentStates map[string]map[string]*StudentConceptState
      
      // Reference to the knowledge graph
      Graph KnowledgeGraph
      
      // Mastery criteria configuration
      Criteria MasteryCriteria
   }

   // NewMemoryMasteryTracker creates a new mastery tracker
   func NewMemoryMasteryTracker(graph KnowledgeGraph) *MemoryMasteryTracker {
      return &MemoryMasteryTracker{
         StudentStates: make(map[string]map[string]*StudentConceptState),
         Graph: graph,
         Criteria: MasteryCriteria{
            MinimumRating: DefaultMasteryThreshold,
            MinimumReviews: 2,
            RequiredLevels: []BloomLevel{BloomRemember, BloomUnderstand},
         },
      }
   }
   ```

2. Implement core mastery tracking methods:
   ```go
   // IsConceptMastered checks if a concept is mastered by a student
   func (m *MemoryMasteryTracker) IsConceptMastered(conceptID, studentID string) (bool, error) {
      // Implementation required
      return false, nil
   }

   // GetMasteryLevel gets the overall mastery level for a concept
   func (m *MemoryMasteryTracker) GetMasteryLevel(conceptID, studentID string) (float64, error) {
      // Implementation required
      return 0, nil
   }

   // UpdateMastery updates mastery based on a card review
   func (m *MemoryMasteryTracker) UpdateMastery(
      conceptID, cardID, studentID string, 
      bloomLevel BloomLevel, 
      rating fsrs.Rating,
   ) error {
      // Implementation required
      return nil
   }

   // CalculateConceptMastery calculates overall mastery from card reviews
   func (m *MemoryMasteryTracker) CalculateConceptMastery(
      conceptID, studentID string,
   ) (float64, error) {
      // Implementation required
      return 0, nil
   }

   // GetBloomLevelMastery gets mastery by Bloom level
   func (m *MemoryMasteryTracker) GetBloomLevelMastery(
      conceptID, studentID string,
   ) (map[BloomLevel]float64, error) {
      // Implementation required
      return nil, nil
   }
   ```

In `internal/graph/student.go`:
1. Implement student-specific state management:
   ```go
   package graph

   import (
      "time"
   )

   // StudentManager tracks student interactions with the knowledge graph
   type StudentManager interface {
      // Get or create a student's state for a concept
      GetOrCreateStudentConceptState(studentID, conceptID string) (*StudentConceptState, error)
      
      // Get all concept states for a student
      GetStudentConceptStates(studentID string) ([]*StudentConceptState, error)
      
      // Update a student's concept state
      UpdateStudentConceptState(state *StudentConceptState) error
      
      // Get student review history for a concept
      GetStudentConceptReviews(studentID, conceptID string) ([]ReviewRecord, error)
   }

   // MemoryStudentManager manages student state in memory
   type MemoryStudentManager struct {
      // Map of studentID -> conceptID -> StudentConceptState
      States map[string]map[string]*StudentConceptState
      
      // Map of studentID -> conceptID -> cardID -> []ReviewRecord
      ReviewHistory map[string]map[string]map[string][]ReviewRecord
   }

   // ReviewRecord stores information about a concept review
   type ReviewRecord struct {
      StudentID   string       `json:"student_id"`
      ConceptID   string       `json:"concept_id"`
      CardID      string       `json:"card_id"`
      BloomLevel  BloomLevel   `json:"bloom_level"`
      Rating      float64      `json:"rating"`
      Timestamp   time.Time    `json:"timestamp"`
   }

   // NewMemoryStudentManager creates a new student manager
   func NewMemoryStudentManager() *MemoryStudentManager {
      return &MemoryStudentManager{
         States: make(map[string]map[string]*StudentConceptState),
         ReviewHistory: make(map[string]map[string]map[string][]ReviewRecord),
      }
   }
   ```

2. Implement student manager methods:
   ```go
   // GetOrCreateStudentConceptState gets or creates a student's state for a concept
   func (m *MemoryStudentManager) GetOrCreateStudentConceptState(
      studentID, conceptID string,
   ) (*StudentConceptState, error) {
      // Implementation required
      return nil, nil
   }

   // GetStudentConceptStates gets all concept states for a student
   func (m *MemoryStudentManager) GetStudentConceptStates(
      studentID string,
   ) ([]*StudentConceptState, error) {
      // Implementation required
      return nil, nil
   }

   // AddReviewRecord adds a review record to history
   func (m *MemoryStudentManager) AddReviewRecord(
      studentID, conceptID, cardID string,
      bloomLevel BloomLevel,
      rating float64,
   ) error {
      // Implementation required
      return nil
   }

   // GetStudentConceptReviews gets review history for a concept
   func (m *MemoryStudentManager) GetStudentConceptReviews(
      studentID, conceptID string,
   ) ([]ReviewRecord, error) {
      // Implementation required
      return nil, nil
   }
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Mastery calculation - Tests that concept mastery is correctly calculated from card reviews
2. Bloom level tracking - Verifies that mastery is tracked separately for different Bloom levels
3. Prerequisite validation - Checks that concept mastery status correctly reflects prerequisite requirements
4. Student state management - Tests creation and retrieval of student concept states
5. Review history - Verifies that review history is correctly recorded and retrieved

## Success Criteria
- [ ] [Mastery calculation] Concept mastery is calculated correctly from card reviews [TestMasteryCalculation]
- [ ] [Bloom level tracking] Mastery is tracked separately for each Bloom level [TestBloomLevelMastery]
- [ ] [Prerequisite validation] IsConceptMastered correctly identifies mastered concepts [TestPrerequisiteValidation]
- [ ] [Student state management] Student concept states are correctly managed [TestStudentStateManagement]
- [ ] [Review history] Review history is correctly recorded and retrieved [TestReviewHistoryManagement]

## Step-by-Step Implementation
1. [ ] Create `internal/graph/mastery.go` with MasteryTracker interface
2. [ ] Implement MemoryMasteryTracker with basic structure
3. [ ] Implement IsConceptMastered and GetMasteryLevel functions
4. [ ] Write tests for mastery checking functions
5. [ ] Run tests to verify mastery checking (`go test ./internal/graph -run=TestMasteryChecking`)
6. [ ] Create `internal/graph/student.go` with StudentManager interface
7. [ ] Implement MemoryStudentManager with basic structure
8. [ ] Implement GetOrCreateStudentConceptState and GetStudentConceptStates functions
9. [ ] Write tests for student state management
10. [ ] Run tests to verify student state management (`go test ./internal/graph -run=TestStudentStateManagement`)
11. [ ] Implement AddReviewRecord and GetStudentConceptReviews functions
12. [ ] Write tests for review history management
13. [ ] Run tests to verify review history management (`go test ./internal/graph -run=TestReviewHistoryManagement`)
14. [ ] Implement UpdateMastery function in MasteryTracker
15. [ ] Implement CalculateConceptMastery for mastery calculation
16. [ ] Write tests for mastery update and calculation
17. [ ] Run tests to verify mastery update and calculation (`go test ./internal/graph -run=TestMasteryUpdate`)
18. [ ] Implement GetBloomLevelMastery for Bloom level tracking
19. [ ] Write tests for Bloom level mastery
20. [ ] Run tests to verify Bloom level mastery (`go test ./internal/graph -run=TestBloomLevelMastery`)
21. [ ] Ensure all tests pass for the complete task (`go test ./internal/graph`)
22. [ ] Run linters and static code analysis
```

### Phase 1.3: MCP Tool Interface

#### Task 1.3.1: Create Core MCP Tools for Knowledge Graph

```
# Task 1.3.1: Create Core MCP Tools for Knowledge Graph

## Background and Context
To enable LLM interaction with our knowledge graph, we need to expose MCP tools that allow LLMs to analyze study materials, create concepts and cards, establish dependencies, and navigate the graph during student interactions.

## My Task
Implement the core MCP tools required for LLM interaction with the knowledge graph, particularly focusing on graph construction, navigation, and review management.

## Files to Modify
1. `cmd/flashcards/graph_tools.go`: Create new file for knowledge graph MCP tools
2. `cmd/flashcards/graph_handlers.go`: Create new file for knowledge graph tool handlers
3. `cmd/flashcards/main.go`: Modify to register the new tools

## Implementation Details
In `cmd/flashcards/graph_tools.go`:
1. Define the core knowledge graph tools:
   ```go
   package main

   import (
      "github.com/mark3labs/mcp-go/mcp"
   )

   // CreateAnalyzeStudyMaterialTool creates a tool for analyzing study materials
   func CreateAnalyzeStudyMaterialTool() *mcp.Tool {
      return mcp.NewTool("analyze_study_material",
         mcp.WithDescription(
            "Analyze learning materials to extract concepts, relationships, and structure. " +
            "USE: When student shares study materials, homework, or learning goals. " +
            "PROCESS: " +
            "1. Extract key concepts and their relationships " +
            "2. Identify learning objectives at appropriate levels " +
            "3. Create a structured knowledge representation " +
            "4. Generate appropriate study questions at different levels"
         ),
         mcp.WithString("content", 
            mcp.Required(),
            mcp.Description("Content of study materials to analyze"),
         ),
         mcp.WithString("subject_area", 
            mcp.Description("Subject area if known"),
         ),
         mcp.WithString("grade_level", 
            mcp.Description("Student's grade level if known"),
         ),
      )
   }

   // CreateConceptTool creates a tool for adding concepts to the graph
   func CreateConceptTool() *mcp.Tool {
      return mcp.NewTool("create_concept",
         mcp.WithDescription(
            "Create a new concept in the knowledge graph. " +
            "USE: When identifying a discrete topic or idea that should be learned. " +
            "PROCESS: " +
            "1. Create a new concept node in the knowledge graph " +
            "2. Set its basic properties (name, description) " +
            "3. Prepare it for flashcard association"
         ),
         mcp.WithString("name", 
            mcp.Required(),
            mcp.Description("Name of the concept"),
         ),
         mcp.WithString("description", 
            mcp.Required(),
            mcp.Description("Description of the concept"),
         ),
      )
   }

   // CreateCreateFlashcardTool creates a tool for creating a new flashcard
   func CreateCreateFlashcardTool() *mcp.Tool {
      return mcp.NewTool("create_flashcard",
         mcp.WithDescription(
            "Create a new flashcard for a concept at a specific Bloom's level. " +
            "INTERNAL PROCESS: " +
            "1. Generate appropriate question for the target Bloom's level " +
            "2. Create answer and scaffolding resources " +
            "3. Add to knowledge graph " +
            "4. Associate with concept node"
         ),
         mcp.WithString("concept_id", 
            mcp.Required(),
            mcp.Description("ID of the concept this card belongs to"),
         ),
         mcp.WithString("bloom_level", 
            mcp.Required(),
            mcp.Description("Bloom's Taxonomy level (remember, understand, apply, analyze, evaluate, create)"),
         ),
         mcp.WithString("question", 
            mcp.Required(),
            mcp.Description("Question text for the flashcard"),
         ),
         mcp.WithString("answer", 
            mcp.Required(),
            mcp.Description("Answer text for the flashcard"),
         ),
      )
   }

   // CreateDependencyTool creates a tool for establishing a dependency
   func CreateDependencyTool() *mcp.Tool {
      return mcp.NewTool("create_dependency",
         mcp.WithDescription(
            "Create a prerequisite relationship between concepts. " +
            "PROCESS: " +
            "1. Establish that one concept is prerequisite to another " +
            "2. Update the knowledge graph structure " +
            "3. Adjust study scheduling to respect the new dependency"
         ),
         mcp.WithString("prerequisite_concept_id", 
            mcp.Required(),
            mcp.Description("ID of the prerequisite concept"),
         ),
         mcp.WithString("dependent_concept_id", 
            mcp.Required(),
            mcp.Description("ID of the dependent concept"),
         ),
         mcp.WithFloat("relationship_strength", 
            mcp.Description("How strong the dependency is (0-1)"),
            mcp.MinValue(0),
            mcp.MaxValue(1),
            mcp.DefaultValue(0.8),
         ),
      )
   }

   // CreateGetNextCardTool creates a tool for getting the next flashcard
   func CreateGetNextCardTool() *mcp.Tool {
      return mcp.NewTool("get_next_card",
         mcp.WithDescription(
            "Get the next optimal flashcard ensuring prerequisites are mastered. " +
            "PROCESS: " +
            "1. Identify cards due for review based on spaced repetition " +
            "2. Filter out cards whose prerequisites aren't mastered (score < 3) " +
            "3. If no valid cards exist, select the most critical prerequisite to review " +
            "4. Otherwise, select the highest priority card among valid candidates"
         ),
         mcp.WithString("student_id", 
            mcp.Required(),
            mcp.Description("ID of the student"),
         ),
         mcp.WithString("session_id", 
            mcp.Required(),
            mcp.Description("ID of the current study session"),
         ),
      )
   }

   // CreateRecordStudentResponseTool creates a tool for recording responses
   func CreateRecordStudentResponseTool() *mcp.Tool {
      return mcp.NewTool("record_student_response",
         mcp.WithDescription(
            "Record and evaluate a student's response to a question. " +
            "USE: After asking a question and receiving the student's answer. " +
            "INTERNAL PROCESS: " +
            "1. Evaluate the response against expected answer " +
            "2. Assess understanding level (internally using Bloom's) " +
            "3. Update knowledge model " +
            "4. Determine appropriate feedback"
         ),
         mcp.WithString("session_id", 
            mcp.Required(),
            mcp.Description("ID of the current study session"),
         ),
         mcp.WithString("card_id", 
            mcp.Required(),
            mcp.Description("ID of the flashcard being reviewed"),
         ),
         mcp.WithString("student_response", 
            mcp.Required(),
            mcp.Description("Student's response to the question"),
         ),
      )
   }
   ```

In `cmd/flashcards/graph_handlers.go`:
1. Implement handler functions for the knowledge graph tools:
   ```go
   package main

   import (
      "context"
      "encoding/json"
      "fmt"
      "github.com/mark3labs/mcp-go/mcp"
   )

   // AnalyzeStudyMaterialHandler handles analyze_study_material requests
   func AnalyzeStudyMaterialHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // CreateConceptHandler handles create_concept requests
   func CreateConceptHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // CreateFlashcardHandler handles create_flashcard requests
   func CreateFlashcardHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // CreateDependencyHandler handles create_dependency requests
   func CreateDependencyHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // GetNextCardHandler handles get_next_card requests
   func GetNextCardHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // RecordStudentResponseHandler handles record_student_response requests
   func RecordStudentResponseHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }
   ```

In `cmd/flashcards/main.go`:
1. Modify the main function to register the new tools:
   ```go
   // Add graph tools to the MCP server
   analyzeStudyMaterialTool := CreateAnalyzeStudyMaterialTool()
   s.AddTool(analyzeStudyMaterialTool, AnalyzeStudyMaterialHandler)

   createConceptTool := CreateConceptTool()
   s.AddTool(createConceptTool, CreateConceptHandler)

   createFlashcardTool := CreateCreateFlashcardTool()
   s.AddTool(createFlashcardTool, CreateFlashcardHandler)

   createDependencyTool := CreateDependencyTool()
   s.AddTool(createDependencyTool, CreateDependencyHandler)

   getNextCardTool := CreateGetNextCardTool()
   s.AddTool(getNextCardTool, GetNextCardHandler)

   recordStudentResponseTool := CreateRecordStudentResponseTool()
   s.AddTool(recordStudentResponseTool, RecordStudentResponseHandler)
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Material analysis - Tests analysis of study materials to extract concepts and relationships
2. Concept creation - Verifies that concepts can be created and added to the graph
3. Flashcard creation - Checks that flashcards can be created and associated with concepts at specific Bloom levels
4. Dependency creation - Tests that prerequisite relationships can be established between concepts
5. Next card selection - Verifies that the next card selection respects prerequisites and priorities
6. Response processing - Tests that student responses are correctly evaluated and used to update the knowledge model

## Success Criteria
- [ ] [Material analysis] Study materials are analyzed to extract concepts and relationships [TestAnalyzeStudyMaterial]
- [ ] [Concept creation] Concepts can be created and added to the graph [TestCreateConcept]
- [ ] [Flashcard creation] Flashcards can be created and associated with concepts [TestCreateFlashcard]
- [ ] [Dependency creation] Prerequisite relationships can be established between concepts [TestCreateDependency]
- [ ] [Next card selection] The next card selection respects prerequisites and priorities [TestGetNextCard]
- [ ] [Response processing] Student responses are correctly evaluated and used to update the knowledge model [TestRecordStudentResponse]

## Step-by-Step Implementation
1. [ ] Create `cmd/flashcards/graph_tools.go` with tool definitions
2. [ ] Create `cmd/flashcards/graph_handlers.go` with handler function signatures
3. [ ] Implement handler method for analyze_study_material
4. [ ] Write tests for analyze_study_material
5. [ ] Run tests to verify analyze_study_material (`go test ./cmd/flashcards -run=TestAnalyzeStudyMaterial`)
6. [ ] Implement handler method for create_concept
7. [ ] Write tests for create_concept
8. [ ] Run tests to verify create_concept (`go test ./cmd/flashcards -run=TestCreateConcept`)
9. [ ] Implement handler method for create_flashcard
10. [ ] Write tests for create_flashcard
11. [ ] Run tests to verify create_flashcard (`go test ./cmd/flashcards -run=TestCreateFlashcard`)
12. [ ] Implement handler method for create_dependency
13. [ ] Write tests for create_dependency
14. [ ] Run tests to verify create_dependency (`go test ./cmd/flashcards -run=TestCreateDependency`)
15. [ ] Implement handler method for get_next_card
16. [ ] Write tests for get_next_card
17. [ ] Run tests to verify get_next_card (`go test ./cmd/flashcards -run=TestGetNextCard`)
18. [ ] Implement handler method for record_student_response
19. [ ] Write tests for record_student_response
20. [ ] Run tests to verify record_student_response (`go test ./cmd/flashcards -run=TestRecordStudentResponse`)
21. [ ] Modify main.go to register the new tools
22. [ ] Ensure all tests pass for the complete task (`go test ./cmd/flashcards`)
23. [ ] Run linters and static code analysis
```

#### Task 1.3.2: Implement Knowledge Gap Analysis Tools

```
# Task 1.3.2: Implement Knowledge Gap Analysis Tools

## Background and Context
A key feature of our knowledge graph-based system is the ability to detect and remediate knowledge gaps when students struggle with concepts. We need tools that analyze student performance, identify potential missing prerequisites, and expand the graph dynamically.

## My Task
Implement MCP tools and handlers for knowledge gap analysis and dynamic graph expansion based on student performance.

## Files to Modify
1. `cmd/flashcards/gap_tools.go`: Create new file for knowledge gap analysis tools
2. `cmd/flashcards/gap_handlers.go`: Create new file for gap analysis tool handlers
3. `cmd/flashcards/main.go`: Modify to register the new tools

## Implementation Details
In `cmd/flashcards/gap_tools.go`:
1. Define the knowledge gap analysis tools:
   ```go
   package main

   import (
      "github.com/mark3labs/mcp-go/mcp"
   )

   // CreateAnalyzeKnowledgeGapsTool creates a tool for analyzing knowledge gaps
   func CreateAnalyzeKnowledgeGapsTool() *mcp.Tool {
      return mcp.NewTool("analyze_knowledge_gaps",
         mcp.WithDescription(
            "Analyze a student's knowledge gaps when they struggle with a concept. " +
            "INTERNAL PROCESS: " +
            "1. Identify the struggling concept and Bloom's level " +
            "2. Check mastery of prerequisites using graph relationships " +
            "3. Recursively test prerequisite concepts as needed " +
            "4. Generate a knowledge gap report with recommended focus areas"
         ),
         mcp.WithString("student_id", 
            mcp.Required(),
            mcp.Description("ID of the student"),
         ),
         mcp.WithString("concept_id", 
            mcp.Required(),
            mcp.Description("ID of the concept the student is struggling with"),
         ),
         mcp.WithString("bloom_level", 
            mcp.Required(),
            mcp.Description("Bloom's Taxonomy level where the struggle occurred"),
         ),
         mcp.WithString("error_pattern", 
            mcp.Description("Observed error pattern if any"),
         ),
      )
   }

   // CreateSuggestGraphExtensionTool creates a tool for suggesting graph extensions
   func CreateSuggestGraphExtensionTool() *mcp.Tool {
      return mcp.NewTool("suggest_graph_extension",
         mcp.WithDescription(
            "Analyze student's current understanding and suggest potential missing prerequisite concepts. " +
            "USE: When student struggles with a concept and existing cards don't address the gap. " +
            "PROCESS: " +
            "1. Analyze the nature of the student's misunderstanding " +
            "2. Identify potential missing prerequisite concepts " +
            "3. Suggest relationships to existing concepts " +
            "4. Recommend Bloom's levels for new flashcards"
         ),
         mcp.WithString("student_response", 
            mcp.Required(),
            mcp.Description("Student's response that indicates misunderstanding"),
         ),
         mcp.WithString("concept_id", 
            mcp.Required(),
            mcp.Description("ID of the concept the student is struggling with"),
         ),
         mcp.WithString("bloom_level", 
            mcp.Required(),
            mcp.Description("Bloom's Taxonomy level where the struggle occurred"),
         ),
         mcp.WithArray("existing_concepts", 
            mcp.Required(),
            mcp.Description("List of existing concept IDs in the graph"),
            mcp.Items(mcp.String()),
         ),
      )
   }

   // CreateSelectScaffoldingTool creates a tool for selecting scaffolding
   func CreateSelectScaffoldingTool() *mcp.Tool {
      return mcp.NewTool("select_scaffolding",
         mcp.WithDescription(
            "Select appropriate scaffolding strategy based on the concept, Bloom's level, and student state. " +
            "INTERNAL PROCESS: " +
            "1. Analyze current mastery level and error patterns " +
            "2. Determine optimal scaffolding intensity " +
            "3. Select strategies from the scaffolding matrix " +
            "4. Generate scaffolding resources for LLM to use"
         ),
         mcp.WithString("student_id", 
            mcp.Required(),
            mcp.Description("ID of the student"),
         ),
         mcp.WithString("concept_id", 
            mcp.Required(),
            mcp.Description("ID of the concept"),
         ),
         mcp.WithString("bloom_level", 
            mcp.Required(),
            mcp.Description("Bloom's Taxonomy level"),
         ),
         mcp.WithArray("previous_scaffolds", 
            mcp.Description("Previously used scaffolding strategies"),
            mcp.Items(mcp.String()),
         ),
      )
   }
   ```

In `cmd/flashcards/gap_handlers.go`:
1. Implement handler functions for the knowledge gap analysis tools:
   ```go
   package main

   import (
      "context"
      "encoding/json"
      "fmt"
      "github.com/mark3labs/mcp-go/mcp"
   )

   // AnalyzeKnowledgeGapsHandler handles analyze_knowledge_gaps requests
   func AnalyzeKnowledgeGapsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // SuggestGraphExtensionHandler handles suggest_graph_extension requests
   func SuggestGraphExtensionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // SelectScaffoldingHandler handles select_scaffolding requests
   func SelectScaffoldingHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }
   ```

In `internal/graph/gaps.go`:
1. Implement the core knowledge gap analysis functionality:
   ```go
   package graph

   import (
      "sort"
   )

   // GapAnalyzer defines operations for analyzing knowledge gaps
   type GapAnalyzer interface {
      // Analyze knowledge gaps for a struggling concept
      AnalyzeKnowledgeGaps(studentID, conceptID string, bloomLevel BloomLevel) (*GapAnalysisResult, error)
      
      // Get prerequisite concepts that aren't mastered
      GetUnmasteredPrerequisites(studentID, conceptID string) ([]*Concept, error)
      
      // Find the most critical prerequisite to focus on
      FindMostCriticalPrerequisite(studentID, conceptID string) (*Concept, error)
      
      // Analyze patterns to suggest potential missing prerequisites
      SuggestMissingPrerequisites(studentID, conceptID string, response string) ([]*PrerequisiteSuggestion, error)
   }

   // GapAnalysisResult represents the result of a knowledge gap analysis
   type GapAnalysisResult struct {
      ConceptID              string                  `json:"concept_id"`
      BloomLevel             BloomLevel              `json:"bloom_level"`
      UnmasteredPrerequisites []*Concept             `json:"unmastered_prerequisites"`
      SuggestedPrerequisites  []*PrerequisiteSuggestion `json:"suggested_prerequisites"`
      RecommendedFocus        *Concept               `json:"recommended_focus"`
      ScaffoldingStrategies   []string               `json:"scaffolding_strategies"`
   }

   // PrerequisiteSuggestion represents a suggested missing prerequisite
   type PrerequisiteSuggestion struct {
      Name            string      `json:"name"`
      Description     string      `json:"description"`
      Confidence      float64     `json:"confidence"`
      RelatedConceptID string     `json:"related_concept_id,omitempty"`
   }

   // BasicGapAnalyzer provides a basic implementation of gap analysis
   type BasicGapAnalyzer struct {
      Graph            KnowledgeGraph
      MasteryTracker   MasteryTracker
      // Other dependencies as needed
   }

   // NewBasicGapAnalyzer creates a new gap analyzer
   func NewBasicGapAnalyzer(graph KnowledgeGraph, masteryTracker MasteryTracker) *BasicGapAnalyzer {
      return &BasicGapAnalyzer{
         Graph:          graph,
         MasteryTracker: masteryTracker,
      }
   }
   ```

2. Implement the core gap analysis methods:
   ```go
   // AnalyzeKnowledgeGaps analyzes knowledge gaps for a concept
   func (a *BasicGapAnalyzer) AnalyzeKnowledgeGaps(
      studentID, conceptID string, 
      bloomLevel BloomLevel,
   ) (*GapAnalysisResult, error) {
      // Implementation required
      return nil, nil
   }

   // GetUnmasteredPrerequisites gets prerequisites that aren't mastered
   func (a *BasicGapAnalyzer) GetUnmasteredPrerequisites(
      studentID, conceptID string,
   ) ([]*Concept, error) {
      // Implementation required
      return nil, nil
   }

   // FindMostCriticalPrerequisite finds the most critical prerequisite
   func (a *BasicGapAnalyzer) FindMostCriticalPrerequisite(
      studentID, conceptID string,
   ) (*Concept, error) {
      // Implementation required
      return nil, nil
   }

   // SuggestMissingPrerequisites suggests potential missing prerequisites
   func (a *BasicGapAnalyzer) SuggestMissingPrerequisites(
      studentID, conceptID string, 
      response string,
   ) ([]*PrerequisiteSuggestion, error) {
      // Implementation required
      return nil, nil
   }
   ```

In `cmd/flashcards/main.go`:
1. Modify the main function to register the new tools:
   ```go
   // Add gap analysis tools to the MCP server
   analyzeKnowledgeGapsTool := CreateAnalyzeKnowledgeGapsTool()
   s.AddTool(analyzeKnowledgeGapsTool, AnalyzeKnowledgeGapsHandler)

   suggestGraphExtensionTool := CreateSuggestGraphExtensionTool()
   s.AddTool(suggestGraphExtensionTool, SuggestGraphExtensionHandler)

   selectScaffoldingTool := CreateSelectScaffoldingTool()
   s.AddTool(selectScaffoldingTool, SelectScaffoldingHandler)
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Knowledge gap analysis - Tests that the system correctly identifies concepts where prerequisites aren't mastered
2. Prerequisite identification - Verifies that unmastered prerequisites are correctly identified and prioritized
3. Missing prerequisite suggestion - Checks that potential missing prerequisites are suggested based on student responses
4. Scaffolding strategy selection - Tests that appropriate scaffolding strategies are selected based on concept and Bloom level
5. Graph extension suggestion - Verifies that graph extensions are suggested when a knowledge gap is identified

## Success Criteria
- [ ] [Knowledge gap analysis] Knowledge gaps are correctly identified for struggling concepts [TestKnowledgeGapAnalysis]
- [ ] [Prerequisite identification] Unmastered prerequisites are correctly identified and prioritized [TestUnmasteredPrerequisites]
- [ ] [Missing prerequisite suggestion] Potential missing prerequisites are suggested based on student responses [TestMissingPrerequisiteSuggestion]
- [ ] [Scaffolding strategy selection] Appropriate scaffolding strategies are selected based on concept and Bloom level [TestScaffoldingSelection]
- [ ] [Graph extension suggestion] Graph extensions are suggested when a knowledge gap is identified [TestGraphExtensionSuggestion]

## Step-by-Step Implementation
1. [ ] Create `internal/graph/gaps.go` with GapAnalyzer interface
2. [ ] Implement BasicGapAnalyzer with core structures and methods
3. [ ] Implement GetUnmasteredPrerequisites method
4. [ ] Write tests for unmastered prerequisite identification
5. [ ] Run tests to verify unmastered prerequisite identification (`go test ./internal/graph -run=TestUnmasteredPrerequisites`)
6. [ ] Implement FindMostCriticalPrerequisite method
7. [ ] Write tests for critical prerequisite identification
8. [ ] Run tests to verify critical prerequisite identification (`go test ./internal/graph -run=TestCriticalPrerequisite`)
9. [ ] Create `cmd/flashcards/gap_tools.go` with tool definitions
10. [ ] Create `cmd/flashcards/gap_handlers.go` with handler function signatures
11. [ ] Implement SuggestMissingPrerequisites method
12. [ ] Write tests for missing prerequisite suggestion
13. [ ] Run tests to verify missing prerequisite suggestion (`go test ./internal/graph -run=TestMissingPrerequisiteSuggestion`)
14. [ ] Implement AnalyzeKnowledgeGaps method that combines all analysis capabilities
15. [ ] Write tests for complete knowledge gap analysis
16. [ ] Run tests to verify knowledge gap analysis (`go test ./internal/graph -run=TestKnowledgeGapAnalysis`)
17. [ ] Implement AnalyzeKnowledgeGapsHandler in gap_handlers.go
18. [ ] Write tests for analyze_knowledge_gaps tool
19. [ ] Run tests to verify analyze_knowledge_gaps tool (`go test ./cmd/flashcards -run=TestAnalyzeKnowledgeGapsHandler`)
20. [ ] Implement SuggestGraphExtensionHandler and SelectScaffoldingHandler
21. [ ] Write tests for these handlers
22. [ ] Run tests to verify these handlers (`go test ./cmd/flashcards -run=TestGapHandlers`)
23. [ ] Modify main.go to register the new tools
24. [ ] Ensure all tests pass for the complete task (`go test ./internal/graph ./cmd/flashcards`)
25. [ ] Run linters and static code analysis
```

### Phase 1.4: Student Interaction Flowcharts

#### Task 1.4.1: Design Student Interaction Flows

```
# Task 1.4.1: Design Student Interaction Flows

## Background and Context
For effective learning experiences, we need to define clear interaction flows for students using the knowledge graph-based flashcard system. These flows will guide the LLM in providing a smooth, conversational experience that hides the technical complexity of the underlying knowledge graph.

## My Task
Create detailed flowcharts and specifications for the main student interaction patterns with the system, covering initial setup, study sessions, and remediation workflows.

## Files to Modify
1. `cmd/flashcards/flows/init_study.md`: Create design doc for initial study setup flow
2. `cmd/flashcards/flows/review_flow.md`: Create design doc for review session flow
3. `cmd/flashcards/flows/remediation_flow.md`: Create design doc for knowledge gap remediation flow
4. `cmd/flashcards/flows/flowcharts.md`: Create flowchart visualizations in Mermaid format

## Implementation Details
In `cmd/flashcards/flows/init_study.md`:
1. Document the initial study setup flow:
   ```markdown
   # Initial Study Setup Flow

   This document defines the flow for setting up a study session with the knowledge graph-based flashcard system.

   ## Primary Goals
   - Analyze student's study materials to build initial knowledge graph
   - Establish baseline for concept relationships
   - Create appropriate flashcards at multiple Bloom's levels
   - Set up a personalized initial study sequence

   ## Flow Steps

   1. **Material Collection**
      - LLM asks student for study materials, homework, or learning goals
      - Student shares materials via text, images, or descriptions
      - LLM acknowledges receipt and explains it will analyze the content

   2. **Material Analysis**
      - LLM calls `analyze_study_material` MCP tool
      - System extracts key concepts and relationships
      - System identifies learning objectives
      - LLM explains the identified topics to the student

   3. **Knowledge Graph Construction**
      - For each identified concept:
         - LLM calls `create_concept` to add it to the graph
         - LLM calls `create_flashcard` to create cards at appropriate Bloom levels
      - For each identified relationship:
         - LLM calls `create_dependency` to establish prerequisites

   4. **Initial Study Plan**
      - System identifies root concepts (those with no prerequisites)
      - LLM explains the study approach to the student:
         - Start with foundational concepts
         - Master prerequisites before moving to dependents
         - Regular reviews based on memory science
      - LLM sets expectations about the conversational learning process

   5. **First Card Selection**
      - LLM calls `get_next_card` to retrieve the first card
      - System selects an appropriate starting point (usually a root concept)
      - LLM presents the first question to the student

   ## Example Conversation

   [Include example conversation demonstrating the flow]

   ## Edge Cases and Handling

   1. **Limited Materials**
      - If student provides minimal materials, LLM should:
         - Ask follow-up questions to gather more information
         - Generate a minimal initial graph
         - Expand the graph dynamically based on student interactions

   2. **Complex Materials**
      - If materials contain many concepts, LLM should:
         - Focus on core concepts first
         - Explain that additional concepts will be added progressively
         - Prioritize foundational concepts

   3. **Unclear Prerequisites**
      - If material doesn't clearly indicate relationships, LLM should:
         - Make best guesses based on domain knowledge
         - Mark relationships as "unverified"
         - Adjust relationships based on student performance
   ```

In `cmd/flashcards/flows/review_flow.md`:
1. Document the review session flow:
   ```markdown
   # Review Session Flow

   This document defines the flow for a standard review session with the knowledge graph-based flashcard system.

   ## Primary Goals
   - Present due cards in optimal order based on prerequisites
   - Evaluate student responses and update memory models
   - Detect knowledge gaps and trigger remediation when needed
   - Progress through Bloom's levels as mastery increases

   ## Flow Steps

   1. **Session Initialization**
      - LLM greets student and establishes session context
      - System creates a new session ID
      - LLM asks if student wants to continue previous focus or start fresh

   2. **Card Selection Loop**
      - System calls `get_next_card` to select optimal next card
      - Card selection respects:
         - Prerequisite relationships (only show cards whose prerequisites are mastered)
         - FSRS due timing (prioritize cards that are due for review)
         - Bloom's level progression (gradually increase cognitive complexity)
      - LLM presents the question in a conversational way

   3. **Response Evaluation**
      - Student provides answer
      - LLM calls `record_student_response` to evaluate the response
      - System updates:
         - Card memory model (difficulty, stability, retrievability)
         - Concept mastery level
         - Bloom level mastery

   4. **Feedback Provision**
      - Based on evaluation, LLM provides feedback:
         - For correct answers: Confirmation and elaboration
         - For partially correct: Clarification and refinement
         - For incorrect: Gentle correction with explanation

   5. **Knowledge Gap Detection**
      - If student struggles (rating ≤ 2):
         - LLM calls `analyze_knowledge_gaps` to identify issues
         - If unmastered prerequisites exist:
            - Transition to remediation flow
         - If no clear prerequisites but still struggling:
            - LLM calls `select_scaffolding` to provide support
            - Apply scaffolding strategy conversationally

   6. **Bloom's Level Progression**
      - If student masters a concept at current Bloom level:
         - System identifies if higher levels exist
         - If yes, gradually introduce higher-level cards
         - If not, focus on dependent concepts

   7. **Session Conclusion**
      - After time limit or card count reached:
         - LLM summarizes progress and achievements
         - Provides preview of next session focus
         - Offers encouragement and persistence motivation

   ## Example Conversation

   [Include example conversation demonstrating the flow]

   ## Edge Cases and Handling

   1. **No Valid Cards Available**
      - If no cards pass prerequisite filter:
         - System identifies most critical prerequisite to focus on
         - LLM explains shift in focus to build necessary foundation
         - Present cards for the prerequisite concept

   2. **Persistent Struggling**
      - If student struggles with same concept repeatedly:
         - Increase scaffolding intensity
         - Consider suggesting graph extension
         - Offer alternative explanations and approaches

   3. **Session Interruption**
      - If session ends unexpectedly:
         - Save current state and progress
         - On return, offer to resume from last position
   ```

In `cmd/flashcards/flows/remediation_flow.md`:
1. Document the knowledge gap remediation flow:
   ```markdown
   # Knowledge Gap Remediation Flow

   This document defines the flow for remediating knowledge gaps when students struggle with concepts.

   ## Primary Goals
   - Identify specific knowledge gaps when students struggle
   - Recursively address prerequisite concepts that aren't mastered
   - Dynamically expand the knowledge graph when missing prerequisites are detected
   - Provide appropriate scaffolding to bridge gaps

   ## Flow Steps

   1. **Gap Detection Trigger**
      - Student struggles with concept (rating ≤ 2)
      - LLM calls `analyze_knowledge_gaps` to identify issues
      - System determines if:
         - Known prerequisites exist but aren't mastered
         - No clear prerequisites exist but student is still struggling

   2. **Known Prerequisite Remediation**
      - If unmastered prerequisites exist:
         - LLM explains the prerequisite relationship conversationally
         - Example: "Before understanding multiplication, we need to make sure you're comfortable with addition."
         - System shifts focus to the prerequisite concept
         - LLM calls `get_next_card` targeting the prerequisite

   3. **Missing Prerequisite Detection**
      - If all known prerequisites mastered but student still struggles:
         - LLM calls `suggest_graph_extension` with student response
         - System analyzes response to suggest potential missing prerequisites
         - LLM conversationally discusses the potential gap

   4. **Graph Extension**
      - If missing prerequisite identified:
         - LLM calls `create_concept` to create the new prerequisite
         - LLM calls `create_flashcard` to create appropriate cards
         - LLM calls `create_dependency` to link to the original concept
         - System marks the relationship as "unverified"

   5. **Scaffolding Application**
      - LLM calls `select_scaffolding` to identify appropriate support
      - Based on scaffolding type, LLM provides:
         - Hints and prompts
         - Alternative explanations
         - Step-by-step guidance
         - Visual representations or analogies
         - Worked examples

   6. **Recursive Remediation**
      - If student struggles with a prerequisite:
         - Apply the same remediation process recursively
         - System tracks the remediation path to prevent excessive depth
         - LLM maintains conversational context across the recursion

   7. **Return to Original Concept**
      - After prerequisite mastery improves:
         - LLM acknowledges progress
         - System returns focus to the original concept
         - LLM frames the return as building on the foundation

   ## Example Conversation

   [Include example conversation demonstrating the flow]

   ## Edge Cases and Handling

   1. **Excessive Recursion**
      - If remediation goes too deep (more than 3 levels):
         - System establishes a "return point"
         - Focus on mastering the deepest prerequisite first
         - Work backward systematically to original concept

   2. **No Clear Gap Identified**
      - If system cannot identify specific gaps:
         - LLM uses general scaffolding techniques
         - Present simplified versions of the concept
         - Try different presentation approaches

   3. **Resistance to Remediation**
      - If student wants to skip prerequisites:
         - LLM explains importance conversationally
         - Offer brief "preview" of original concept
         - Return to prerequisites with clearer connection to goal
   ```

In `cmd/flashcards/flows/flowcharts.md`:
1. Create flowchart visualizations using Mermaid:
   ```markdown
   # Student Interaction Flowcharts

   This document provides visual flowcharts for the main student interaction patterns.

   ## Initial Study Setup Flow

   ```mermaid
   flowchart TD
       A[Student Shares Materials] --> B[Analyze Materials]
       B --> C[Extract Concepts]
       B --> D[Identify Relationships]
       C --> E[Create Concepts]
       E --> F[Create Flashcards]
       D --> G[Establish Dependencies]
       F --> H[Identify Starting Points]
       G --> H
       H --> I[Begin First Review]
   ```

   ## Standard Review Flow

   ```mermaid
   flowchart TD
       A[Start Session] --> B[Get Next Card]
       B --> C{Prerequisites Mastered?}
       C -->|Yes| D[Present Question]
       C -->|No| E[Focus on Prerequisite]
       E --> B
       D --> F[Student Answers]
       F --> G[Evaluate Response]
       G --> H{Correct?}
       H -->|Yes| I[Positive Feedback]
       H -->|Partially| J[Clarification]
       H -->|No| K[Check for Knowledge Gap]
       I --> L{More Cards?}
       J --> L
       K --> M{Gap Detected?}
       M -->|Yes| N[Remediation Flow]
       M -->|No| O[Provide Scaffolding]
       O --> L
       N --> L
       L -->|Yes| B
       L -->|No| P[End Session]
   ```

   ## Knowledge Gap Remediation Flow

   ```mermaid
   flowchart TD
       A[Student Struggles] --> B[Analyze Knowledge Gaps]
       B --> C{Unmastered Prerequisites?}
       C -->|Yes| D[Focus on Prerequisite]
       C -->|No| E{Missing Prerequisites?}
       E -->|Yes| F[Suggest Graph Extension]
       E -->|No| G[Select Scaffolding]
       F --> H[Create New Prerequisite]
       H --> I[Create Dependency]
       I --> D
       D --> J[Review Prerequisite]
       J --> K{Mastered?}
       K -->|Yes| L[Return to Original]
       K -->|No| M[Apply Scaffolding]
       M --> J
       G --> N[Apply Scaffolding]
       N --> O[Review Original Concept]
   ```
   ```

## Behaviors to Test
This task involves designing and documenting interaction flows rather than implementing code, so "testing" refers to reviewing the flows for completeness and usability:

1. Flow completeness - Ensure flows cover all major interaction patterns
2. Edge case handling - Verify that flows address common edge cases and problems
3. Conversational quality - Ensure flows maintain natural conversation
4. Technical accuracy - Verify that flows correctly use the underlying knowledge graph capabilities
5. User experience - Ensure flows provide a positive, encouraging learning experience

## Success Criteria
- [ ] [Flow completeness] All major interaction patterns are documented [Design Review]
- [ ] [Edge case handling] Common edge cases are addressed in each flow [Design Review]
- [ ] [Conversational quality] Flows maintain natural conversation while hiding technical complexity [Design Review]
- [ ] [Technical accuracy] Flows correctly use the underlying knowledge graph capabilities [Design Review]
- [ ] [User experience] Flows provide a positive, encouraging learning experience [Design Review]

## Step-by-Step Implementation
1. [ ] Create the `cmd/flashcards/flows` directory
2. [ ] Draft the initial study setup flow document
3. [ ] Review for completeness and clarity
4. [ ] Draft the standard review flow document
5. [ ] Review for completeness and clarity
6. [ ] Draft the knowledge gap remediation flow document
7. [ ] Review for completeness and clarity
8. [ ] Create flowchart visualizations in Mermaid format
9. [ ] Review flowcharts for accuracy and clarity
10. [ ] Finalize all flow documents
11. [ ] Share with team for review and feedback
```

#### Task 1.4.2: Implement Session Management

```
# Task 1.4.2: Implement Session Management

## Background and Context
To support the student interaction flows, we need a session management system that maintains context across interactions, tracks the current study focus, and manages state transitions between different flows (review, remediation, etc.).

## My Task
Implement a session management system that supports the defined student interaction flows and maintains study context across interactions.

## Files to Modify
1. `internal/graph/session.go`: Create new file for session management 
2. `cmd/flashcards/session_handlers.go`: Create new file for session-related MCP tools and handlers

## Implementation Details
In `internal/graph/session.go`:
1. Define the session management interfaces and structures:
   ```go
   package graph

   import (
      "time"
      "github.com/google/uuid"
      "github.com/open-spaced-repetition/go-fsrs"
   )

   // SessionMode defines different modes for a study session
   type SessionMode string

   const (
      // Standard review mode
      SessionModeReview     SessionMode = "review"
      // Remediating a knowledge gap
      SessionModeRemediation SessionMode = "remediation"
      // Introducing a new concept
      SessionModeIntroduction SessionMode = "introduction"
   )

   // SessionState represents the current state of a study session
   type SessionState struct {
      ID                string          `json:"id"`
      StudentID         string          `json:"student_id"`
      CurrentMode       SessionMode     `json:"current_mode"`
      CurrentConceptID  string          `json:"current_concept_id,omitempty"`
      CurrentCardID     string          `json:"current_card_id,omitempty"`
      RemediationPath   []string        `json:"remediation_path,omitempty"`
      RemediationTarget string          `json:"remediation_target,omitempty"`
      CardsReviewed     int             `json:"cards_reviewed"`
      CorrectResponses  int             `json:"correct_responses"`
      StartTime         time.Time       `json:"start_time"`
      LastActivity      time.Time       `json:"last_activity"`
      Completed         bool            `json:"completed"`
   }

   // SessionEvent represents an event in a study session
   type SessionEvent struct {
      SessionID         string          `json:"session_id"`
      Timestamp         time.Time       `json:"timestamp"`
      EventType         string          `json:"event_type"`
      CardID            string          `json:"card_id,omitempty"`
      ConceptID         string          `json:"concept_id,omitempty"`
      Rating            fsrs.Rating     `json:"rating,omitempty"`
      ModeTransition    *ModeTransition `json:"mode_transition,omitempty"`
      Note              string          `json:"note,omitempty"`
   }

   // ModeTransition represents a transition between session modes
   type ModeTransition struct {
      FromMode          SessionMode     `json:"from_mode"`
      ToMode            SessionMode     `json:"to_mode"`
      Reason            string          `json:"reason"`
   }

   // SessionManager defines operations for managing study sessions
   type SessionManager interface {
      // Create a new session for a student
      CreateSession(studentID string) (*SessionState, error)
      
      // Get an existing session by ID
      GetSession(sessionID string) (*SessionState, error)
      
      // Update a session's state
      UpdateSession(session *SessionState) error
      
      // Complete a session
      CompleteSession(sessionID string) error
      
      // Record an event in the session
      RecordEvent(event *SessionEvent) error
      
      // Get events for a session
      GetSessionEvents(sessionID string) ([]*SessionEvent, error)
      
      // Change session mode
      ChangeMode(sessionID string, newMode SessionMode, reason string) error
      
      // Start remediation for a concept
      StartRemediation(sessionID string, targetConceptID string) error
      
      // Return from remediation to previous concept
      ReturnFromRemediation(sessionID string) error
   }

   // MemorySessionManager provides an in-memory implementation
   type MemorySessionManager struct {
      Sessions     map[string]*SessionState
      Events       map[string][]*SessionEvent
   }

   // NewMemorySessionManager creates a new session manager
   func NewMemorySessionManager() *MemorySessionManager {
      return &MemorySessionManager{
         Sessions: make(map[string]*SessionState),
         Events:   make(map[string][]*SessionEvent),
      }
   }
   ```

2. Implement the session management methods:
   ```go
   // CreateSession creates a new study session
   func (m *MemorySessionManager) CreateSession(studentID string) (*SessionState, error) {
      // Implementation required
      return nil, nil
   }

   // GetSession gets an existing session by ID
   func (m *MemorySessionManager) GetSession(sessionID string) (*SessionState, error) {
      // Implementation required
      return nil, nil
   }

   // UpdateSession updates a session's state
   func (m *MemorySessionManager) UpdateSession(session *SessionState) error {
      // Implementation required
      return nil
   }

   // RecordEvent records an event in the session
   func (m *MemorySessionManager) RecordEvent(event *SessionEvent) error {
      // Implementation required
      return nil
   }

   // ChangeMode changes the session mode
   func (m *MemorySessionManager) ChangeMode(
      sessionID string, 
      newMode SessionMode, 
      reason string,
   ) error {
      // Implementation required
      return nil
   }

   // StartRemediation starts a remediation flow
   func (m *MemorySessionManager) StartRemediation(
      sessionID string, 
      targetConceptID string,
   ) error {
      // Implementation required
      return nil
   }

   // ReturnFromRemediation returns from remediation to the original concept
   func (m *MemorySessionManager) ReturnFromRemediation(sessionID string) error {
      // Implementation required
      return nil
   }
   ```

In `cmd/flashcards/session_handlers.go`:
1. Implement MCP tools and handlers for session management:
   ```go
   package main

   import (
      "context"
      "github.com/mark3labs/mcp-go/mcp"
   )

   // CreateSessionTool creates a tool for managing sessions
   func CreateSessionTool() *mcp.Tool {
      return mcp.NewTool("create_session",
         mcp.WithDescription(
            "Create a new study session for a student. " +
            "USE: At the start of a new learning interaction. " +
            "PROCESS: " +
            "1. Initialize session tracking " +
            "2. Record session start time " +
            "3. Return session ID for further reference"
         ),
         mcp.WithString("student_id", 
            mcp.Required(),
            mcp.Description("ID of the student"),
         ),
      )
   }

   // CreateUpdateSessionTool creates a tool for updating session state
   func CreateUpdateSessionTool() *mcp.Tool {
      return mcp.NewTool("update_session",
         mcp.WithDescription(
            "Update the state of a study session. " +
            "USE: When session mode or focus changes. " +
            "PROCESS: " +
            "1. Record the current state " +
            "2. Update the session properties " +
            "3. Return updated session state"
         ),
         mcp.WithString("session_id", 
            mcp.Required(),
            mcp.Description("ID of the session"),
         ),
         mcp.WithString("mode", 
            mcp.Description("New session mode"),
         ),
         mcp.WithString("concept_id", 
            mcp.Description("Current concept focus"),
         ),
      )
   }

   // CreateStartRemediationTool creates a tool for starting remediation
   func CreateStartRemediationTool() *mcp.Tool {
      return mcp.NewTool("start_remediation",
         mcp.WithDescription(
            "Start a remediation flow for a struggling concept. " +
            "USE: When student struggles with a concept and needs to address prerequisites. " +
            "PROCESS: " +
            "1. Record original concept as remediation target " +
            "2. Change session mode to remediation " +
            "3. Focus on prerequisite concept"
         ),
         mcp.WithString("session_id", 
            mcp.Required(),
            mcp.Description("ID of the session"),
         ),
         mcp.WithString("target_concept_id", 
            mcp.Required(),
            mcp.Description("Concept that needs remediation"),
         ),
         mcp.WithString("prerequisite_concept_id", 
            mcp.Required(),
            mcp.Description("Prerequisite to focus on"),
         ),
      )
   }

   // CreateSessionStatsToolHandler handles session_stats requests
   func CreateSessionStatsTool() *mcp.Tool {
      return mcp.NewTool("session_stats",
         mcp.WithDescription(
            "Get statistics about the current study session. " +
            "USE: When summarizing progress or providing feedback. " +
            "PROCESS: " +
            "1. Calculate session statistics " +
            "2. Prepare summary of activities " +
            "3. Return formatted statistics"
         ),
         mcp.WithString("session_id", 
            mcp.Required(),
            mcp.Description("ID of the session"),
         ),
      )
   }

   // CreateSessionHandler handles create_session requests
   func CreateSessionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // UpdateSessionHandler handles update_session requests
   func UpdateSessionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // StartRemediationHandler handles start_remediation requests
   func StartRemediationHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }

   // SessionStatsHandler handles session_stats requests
   func SessionStatsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation required
      return nil, nil
   }
   ```

In `cmd/flashcards/main.go`:
1. Modify the main function to register the new tools:
   ```go
   // Add session management tools to the MCP server
   createSessionTool := CreateSessionTool()
   s.AddTool(createSessionTool, CreateSessionHandler)

   updateSessionTool := CreateUpdateSessionTool()
   s.AddTool(updateSessionTool, UpdateSessionHandler)

   startRemediationTool := CreateStartRemediationTool()
   s.AddTool(startRemediationTool, StartRemediationHandler)

   sessionStatsTool := CreateSessionStatsTool()
   s.AddTool(sessionStatsTool, SessionStatsHandler)
   ```

## Behaviors to Test
The following functional behaviors should be tested:
1. Session creation and retrieval - Verifies that sessions can be created and retrieved
2. Event recording - Tests that session events are correctly recorded and retrieved
3. Mode transitions - Checks that session modes can be changed with proper tracking
4. Remediation flow - Tests the remediation path management (start, navigate, return)
5. Session statistics - Verifies that session statistics are correctly calculated

## Success Criteria
- [ ] [Session management] Sessions can be created, retrieved, and updated [TestSessionManagement]
- [ ] [Event recording] Session events are correctly recorded and retrieved [TestSessionEvents]
- [ ] [Mode transitions] Session modes can be changed with proper tracking [TestModeTransitions]
- [ ] [Remediation flow] Remediation paths are correctly managed [TestRemediationFlow]
- [ ] [Session statistics] Session statistics are correctly calculated [TestSessionStatistics]

## Step-by-Step Implementation
1. [ ] Create `internal/graph/session.go` with SessionManager interface
2. [ ] Implement basic MemorySessionManager structure
3. [ ] Implement CreateSession and GetSession methods
4. [ ] Write tests for session creation and retrieval
5. [ ] Run tests to verify session creation and retrieval (`go test ./internal/graph -run=TestSessionCreationRetrieval`)
6. [ ] Implement RecordEvent and GetSessionEvents methods
7. [ ] Write tests for event recording and retrieval
8. [ ] Run tests to verify event recording and retrieval (`go test ./internal/graph -run=TestSessionEvents`)
9. [ ] Implement ChangeMode and related mode transition methods
10. [ ] Write tests for mode transitions
11. [ ] Run tests to verify mode transitions (`go test ./internal/graph -run=TestModeTransitions`)
12. [ ] Implement StartRemediation and ReturnFromRemediation methods
13. [ ] Write tests for remediation flow management
14. [ ] Run tests to verify remediation flow management (`go test ./internal/graph -run=TestRemediationFlow`)
15. [ ] Create `cmd/flashcards/session_handlers.go` with tool definitions
16. [ ] Implement CreateSessionHandler and UpdateSessionHandler
17. [ ] Write tests for these handlers
18. [ ] Run tests to verify these handlers (`go test ./cmd/flashcards -run=TestSessionHandlers`)
19. [ ] Implement StartRemediationHandler and SessionStatsHandler
20. [ ] Write tests for these handlers
21. [ ] Run tests to verify these handlers (`go test ./cmd/flashcards -run=TestRemediationHandlers`)
22. [ ] Modify main.go to register the new tools
23. [ ] Ensure all tests pass for the complete task (`go test ./internal/graph ./cmd/flashcards`)
24. [ ] Run linters and static code analysis
```

## Execution Order

1. **First, Knowledge Graph Foundation**:
   - Task 1.1.1: Implement Core Knowledge Graph Data Structures
   - Task 1.1.2: Implement Graph Persistence Layer

2. **Then, FSRS Integration**:
   - Task 1.2.1: Extend FSRS for Knowledge Graph Awareness
   - Task 1.2.2: Implement Concept Mastery Tracking

3. **Next, MCP Tool Interface**:
   - Task 1.3.1: Create Core MCP Tools for Knowledge Graph
   - Task 1.3.2: Implement Knowledge Gap Analysis Tools

4. **Finally, Student Interaction Flowcharts**:
   - Task 1.4.1: Design Student Interaction Flows
   - Task 1.4.2: Implement Session Management

This sequence ensures that we build the system in a logical order, starting with the core data structures and persistence layer, then adding the intelligence layer (FSRS integration), followed by the interface layer (MCP tools), and finally defining the student interaction patterns.

## Phase 1 Completion Criteria

At the end of this phase, we should have:

1. A fully functional knowledge graph structure that represents concepts, dependencies, and Bloom's Taxonomy levels
2. FSRS integration that respects concept prerequisites and adjusts scheduling based on graph position
3. MCP tools that enable LLMs to interact with the knowledge graph for study sessions
4. Clearly defined student interaction flows and session management capabilities
5. Working implementation of knowledge gap detection and remediation

The implementation should correctly handle:
- Creating and navigating a knowledge graph of learning concepts
- Enforcing prerequisite relationships between concepts
- Scheduling reviews based on both memory state and graph position
- Detecting knowledge gaps when students struggle with concepts
- Supporting interactive study sessions with coherent flows

All tests should pass and the code should adhere to the project's style guide.
