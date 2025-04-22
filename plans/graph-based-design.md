# Knowledge Graph-Based Flashcard System with FSRS Integration

## 1. Vision and Goals

### 1.1 Overall Vision

The Knowledge Graph-Based Flashcard System aims to revolutionize how students learn by combining three powerful educational approaches:

1. **Knowledge Graph Structure**: Representing concepts and their dependencies as a graph to ensure logical learning progression
2. **Bloom's Taxonomy**: Categorizing learning tasks by cognitive complexity to promote deeper understanding
3. **Spaced Repetition (FSRS)**: Optimizing review timing for maximum memory retention with minimal study time

Unlike traditional flashcard systems that treat cards as isolated units, our system understands the relationships between concepts, ensuring students master prerequisites before encountering advanced topics. When knowledge gaps are detected, the system dynamically expands the graph to address these gaps, creating a personalized learning experience.

### 1.2 Key Goals

1. **Ensure Structured Learning**: Guide students through material in a logical order based on concept dependencies
2. **Adaptive Remediation**: When students struggle, recursively identify and address missing prerequisite knowledge
3. **Optimize Memory Formation**: Schedule reviews at optimal intervals based on the FSRS algorithm
4. **Progressive Cognitive Development**: Target increasingly complex thinking skills using Bloom's Taxonomy
5. **Handle Incomplete Knowledge**: Function effectively even with initially sparse knowledge graphs
6. **Seamless Student Experience**: Hide technical complexity beneath a friendly, conversational interface
7. **Dynamic Expansion**: Continuously improve the knowledge graph based on student performance

### 1.3 Target Users

The primary users are middle school students interacting through a conversational LLM interface. The system is designed to hide the technical complexity of the knowledge graph, Bloom's Taxonomy, and spaced repetition algorithms while providing their benefits through a natural study experience.

## 2. Prerequisite-First Approach

### 2.1 Core Principles

The foundation of our system is the "prerequisite-first" approach to learning, which ensures students master fundamental concepts before encountering dependent ones:

1. **Dependency Enforcement**: Students only see flashcards whose prerequisites have been mastered
2. **Mastery Definition**: Prerequisite concepts must achieve a minimum rating of 3+ on a 4-point scale
3. **Recursive Remediation**: When students struggle, the system identifies and addresses missing prerequisite knowledge
4. **Strategic Sequencing**: The system optimizes the learning path to efficiently cover prerequisites

### 2.2 Theoretical Basis

The prerequisite-first approach is grounded in Knowledge Space Theory (KST), which models knowledge domains as partially ordered sets representing prerequisite relationships. This approach is well-established in adaptive learning systems and has been empirically validated through implementations like ALEKS.

By combining KST with Bloom's Taxonomy and the FSRS algorithm, we create a system that addresses not just *what* to learn and in what order (KST), but also *how deeply* to understand it (Bloom's) and *when* to review it (FSRS).

## 3. Knowledge Graph Structure

### 3.1 Graph Components

The knowledge graph consists of:

```go
// Concept represents a discrete learning topic
type Concept struct {
    ID              string                      `json:"id"`
    Name            string                      `json:"name"`
    Description     string                      `json:"description"`
    BloomCards      map[BloomLevel][]FlashCard `json:"bloom_cards"`  // Cards at different cognitive levels
    PrerequisiteIDs []string                    `json:"prerequisite_ids"`
    DependentIDs    []string                    `json:"dependent_ids"`
    MasteryLevels   map[string]float64          `json:"mastery_levels"`  // By student
    Created         time.Time                   `json:"created"`
    LastModified    time.Time                   `json:"last_modified"`
}

// Dependency represents a prerequisite relationship
type Dependency struct {
    FromID          string     `json:"from_id"`  // Prerequisite
    ToID            string     `json:"to_id"`    // Dependent
    Strength        float64    `json:"strength"` // 0-1 importance of relationship
    Confidence      float64    `json:"confidence"` // 0-1 confidence in this relationship
    Evidence        []string   `json:"evidence"` // Why we believe this relationship exists
    Verified        bool       `json:"verified"` // Confirmed through interactions
}

// FlashCard represents a single study item
type FlashCard struct {
    ID              string     `json:"id"`
    ConceptID       string     `json:"concept_id"`
    BloomLevel      BloomLevel `json:"bloom_level"`
    Question        string     `json:"question"`
    Answer          string     `json:"answer"`
    Scaffolding     []ScaffoldingResource `json:"scaffolding"`
    
    // FSRS parameters
    Difficulty      float64    `json:"difficulty"`
    Stability       float64    `json:"stability"`
    Retrievability  float64    `json:"retrievability"`
    LastReviewed    time.Time  `json:"last_reviewed"`
    DueDate         time.Time  `json:"due_date"`
    ReviewHistory   []ReviewRecord `json:"review_history"`
}
```

### 3.2 Dual-Layer Structure

The knowledge graph has two integrated dimensions:

1. **Horizontal Dimension**: Prerequisite relationships between concepts
   - Concepts are connected in a directed acyclic graph (DAG)
   - Edges represent "is prerequisite for" relationships

2. **Vertical Dimension**: Cognitive complexity levels within each concept
   - Each concept has flashcards at different Bloom's Taxonomy levels
   - Students progress upward through cognitive levels as they master each concept

```
┌─────────────────────────────────────────────────────────────┐
│                    CONCEPT DEPENDENCY LAYER                 │
│                                                             │
│    ┌─────┐       ┌─────┐       ┌─────┐       ┌─────┐       │
│    │ C1  ├──────►│ C2  ├──────►│ C3  ├──────►│ C4  │       │
│    └──┬──┘       └──┬──┘       └──┬──┘       └──┬──┘       │
│       │             │             │             │          │
│       │             │             │             │          │
│       ▼             ▼             ▼             ▼          │
│    ┌─────┐       ┌─────┐       ┌─────┐       ┌─────┐       │
│    │  B  │       │  B  │       │  B  │       │  B  │       │
│    │  L  │       │  L  │       │  L  │       │  L  │       │
│    │  O  │       │  O  │       │  O  │       │  O  │       │
│    │  O  │       │  O  │       │  O  │       │  O  │       │
│    │  M  │       │  M  │       │  M  │       │  M  │       │
│    │  '  │       │  '  │       │  '  │       │  '  │       │
│    │  S  │       │  S  │       │  S  │       │  S  │       │
│    │     │       │     │       │     │       │     │       │
│    └─────┘       └─────┘       └─────┘       └─────┘       │
│                                                             │
│                   COGNITIVE COMPLEXITY LAYER                │
└─────────────────────────────────────────────────────────────┘
```

## 4. Handling Incomplete Graphs

### 4.1 Dynamic Graph Construction

The default state of our system is an incomplete knowledge graph, since students typically begin with a limited set of flashcards. We address this through dynamic graph construction:

1. **Seed Graph**: Initial flashcards create a sparse "seed graph" of disconnected concepts
2. **Just-In-Time Expansion**: The graph expands only when needed, based on student performance
3. **Prerequisite Inference**: The LLM suggests likely prerequisites when students struggle
4. **Verification**: Relationships are initially marked as "unverified" and gain confidence through student interactions

### 4.2 Parallel Graph Structures

To manage uncertainty in the knowledge graph, we maintain two parallel structures:

1. **Explicit Graph**: Contains only concepts and relationships explicitly created or verified
2. **Shadow Graph**: Contains LLM-suggested relationships and concepts that haven't been fully verified

```
┌──────────────────────────────────────────────────┐
│  EXPLICIT GRAPH (CONFIRMED THROUGH INTERACTION)  │
│                                                  │
│    [Card3] ────► [Card7] ────► [Card12]          │
│       ▲                           ▲              │
│       │                           │              │
│    [Card5]                     [Card18]          │
└──────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────┐
│  SHADOW GRAPH (LLM-SUGGESTED, UNCONFIRMED)       │
│                                                  │
│    [Card3] ────► [Card7] ────► [Card12]          │
│       ▲             ▲             ▲              │
│       │             │             │              │
│    [Card5]       [Missing]     [Card18]          │
│       ▲          Concept         ▲               │
│       │                          │               │
│    [Missing] ───────────────► [Missing]          │
│    Concept                     Concept           │
└──────────────────────────────────────────────────┘
```

### 4.3 Probabilistic Relationship Modeling

To handle uncertainty in the knowledge graph, relationships include confidence scores:

```go
type Dependency struct {
    // ... other fields ...
    Confidence     float64    `json:"confidence"`    // 0-1 confidence score
    Evidence       []string   `json:"evidence"`      // Records of why we believe this relationship exists
    Verified       bool       `json:"verified"`      // Has this been confirmed through student interaction?
}
```

### 4.4 Gap Detection Logic

When a student struggles with a concept, the system uses this algorithm to detect potential knowledge gaps:

```go
// When a student struggles with a concept
func HandleConceptStruggle(conceptID string, studentResponse string) {
    // 1. Check if known prerequisites exist and are mastered
    concept := graph.GetConcept(conceptID)
    unmasteredPrereqs := GetUnmasteredPrerequisites(concept, studentID)
    
    if len(unmasteredPrereqs) > 0 {
        // 2a. Known prerequisites exist but aren't mastered
        // Focus on these prerequisites first
        SchedulePrerequisiteReview(unmasteredPrereqs, studentID)
        return
    }
    
    // 2b. All known prerequisites mastered or none exist
    // Ask LLM to suggest potential missing prerequisites
    potentialMissingPrereqs := LLM.AnalyzePotentialPrerequisites(
        conceptID, 
        studentResponse,
        graph.GetExistingConcepts()
    )
    
    if len(potentialMissingPrereqs) > 0 {
        // 3. Create these prerequisites and add to graph
        newPrereqs := CreatePrerequisiteNodes(potentialMissingPrereqs, conceptID)
        AddToGraph(newPrereqs)
        
        // 4. Start with these prerequisites
        SchedulePrerequisiteReview(newPrereqs, studentID)
    } else {
        // 5. No clear prerequisites, try different scaffolding approach
        ApplyAlternativeScaffolding(conceptID, studentID)
    }
}
```

## 5. Graph Navigation with Prerequisite Enforcement

### 5.1 Navigation Principles

The system navigates the knowledge graph according to these principles:

1. **Prerequisite First**: Always ensure prerequisites are mastered before showing dependent concepts
2. **Level Locking**: Concepts are "locked" until their prerequisites reach a rating of 3+ (out of 4)
3. **Breadth vs. Depth**: The system balances breadth-first (covering prerequisites) with depth-first (cognitive levels) navigation
4. **Priority Weighting**: Cards are prioritized based on a combination of:
   - FSRS scheduling (memory optimization)
   - Graph position (prerequisite importance)
   - Current learning focus
   - Bloom's level progression

### 5.2 Navigation Algorithm

The graph navigation follows this logic:

```
┌───────────────────────────────────────────┐
│       CONCEPT DEPENDENCY NAVIGATION       │
│                                           │
│  Level 1      Level 2       Level 3       │
│                                           │
│ [Concept A] ──► [Concept C] ──► [Concept E]│
│     ▲              ▲               ▲      │
│     │              │               │      │
│     │              │               │      │
│ [Concept B] ──► [Concept D] ──► [Concept F]│
│                                           │
│  Only unlock      Only unlock    Only     │
│  when Level 1     when Level 2   unlock   │
│  mastered (3+)    mastered (3+)  when... │
└───────────────────────────────────────────┘
```

## 6. Handling New Flashcard Creation

### 6.1 Initial Processing Algorithm

When a student creates new flashcards, the system process them as follows:

```go
// Process newly created flashcards 
func ProcessNewFlashcards(cards []FlashCard, studentID string) {
    // 1. Have LLM analyze cards to suggest dependencies
    suggestedDependencies := LLM.AnalyzeCardDependencies(cards)
    
    // 2. Build initial knowledge graph structure
    graph := BuildInitialGraph(cards, suggestedDependencies)
    
    // 3. Identify root concepts (those with no prerequisites)
    rootConcepts := FindRootConcepts(graph)
    
    // 4. Set initial study sequence starting with roots
    SetInitialReviewSchedule(rootConcepts, studentID)
    
    // 5. Schedule dependent cards with appropriate delays
    ScheduleDependentCards(graph, studentID)
    
    // 6. Create additional cards at different Bloom's levels
    for _, concept := range graph.Concepts {
        EnsureBloomLevelCoverage(concept)
    }
}
```

### 6.2 Bloom's Level Card Generation

For each concept, the system ensures coverage across Bloom's levels:

```go
// Ensure cards exist at each Bloom's level
func EnsureBloomLevelCoverage(concept *Concept) {
    // Check which Bloom's levels are missing cards
    missingLevels := FindMissingBloomLevels(concept)
    
    // Generate cards for missing levels
    for _, level := range missingLevels {
        cards := LLM.GenerateBloomLevelCards(concept.Name, concept.Description, level)
        concept.BloomCards[level] = append(concept.BloomCards[level], cards...)
    }
}
```

### 6.3 Creating Logical Study Sequence

The system creates a logical study sequence by:

1. Identifying root concepts (those with no prerequisites)
2. Scheduling these for immediate study
3. Scheduling dependent concepts with appropriate delays
4. Beginning with lower Bloom's levels before advancing to higher ones

## 7. Prerequisite-Enforced Card Selection Algorithm

### 7.1 Card Selection Logic

```go
// Get next card ensuring prerequisites are mastered
func GetNextCardWithPrerequisiteCheck(studentID string, graph *KnowledgeGraph) (*FlashCard, error) {
    // 1. Get all due cards using FSRS
    dueCards := GetDueCards(studentID)
    
    // 2. Filter cards based on prerequisite mastery
    validCards := []FlashCard{}
    
    for _, card := range dueCards {
        // Get concept for this card
        concept := graph.GetConcept(card.ConceptID)
        
        // Check if all prerequisites are mastered (3+ rating)
        prerequisitesMastered := true
        for _, prereqID := range concept.PrerequisiteIDs {
            masteryLevel := GetConceptMasteryLevel(studentID, prereqID)
            if masteryLevel < 3.0 {
                prerequisitesMastered = false
                break
            }
        }
        
        // Only include cards with mastered prerequisites
        if prerequisitesMastered {
            validCards = append(validCards, card)
        }
    }
    
    // 3. If no valid cards exist, find prerequisite to work on
    if len(validCards) == 0 {
        // Find the most important unmastared prerequisite
        return FindMostCriticalPrerequisite(studentID, graph)
    }
    
    // 4. From valid cards, use FSRS to select highest priority
    return SelectHighestPriorityCard(validCards)
}
```

### 7.2 Priority Calculation

Cards are prioritized using a weighted formula:

```go
// Calculate priority score for a card
func CalculateCardPriority(card FlashCard, graph *KnowledgeGraph, studentID string) float64 {
    // Base priority from FSRS
    fsrsPriority := CalculateFSRSPriority(card)
    
    // Adjustment for position in knowledge graph
    graphPosition := CalculateGraphPositionFactor(card, graph)
    
    // Adjustment for Bloom's level progression
    bloomFactor := CalculateBloomProgressionFactor(card, studentID)
    
    // Combine factors with appropriate weights
    return (0.5 * fsrsPriority) + (0.3 * graphPosition) + (0.2 * bloomFactor)
}
```

## 8. Integration with FSRS Algorithm

### 8.1 FSRS Memory Model

The Free Spaced Repetition Scheduler (FSRS) is based on a three-component memory model:

1. **Difficulty (D)**: How challenging the card is for the student
2. **Stability (S)**: How resistant the memory is to forgetting
3. **Retrievability (R)**: The probability of successful recall at a given moment

These components determine optimal review timing to maximize retention while minimizing study time.

### 8.2 Enhanced FSRS Integration

Our system enhances FSRS by making it knowledge-graph aware:

```go
// Calculate review priority for a card considering graph position
func CalculateReviewPriority(card FlashCard, knowledgeGraph Graph) float64 {
    // Base priority from standard FSRS
    basePriority := CalculateStandardFSRSPriority(card)
    
    // Adjust based on graph position
    graphAdjustment := 1.0
    
    // Prerequisite concepts get priority boost
    if IsPrerequisiteForCurrentFocus(card.ConceptID, knowledgeGraph) {
        graphAdjustment *= 1.5
    }
    
    // Concepts in active remediation path get higher priority
    if IsInRemediationPath(card.ConceptID) {
        graphAdjustment *= 2.0
    }
    
    // Adjust priority based on Bloom's level relative to current focus
    bloomAdjustment := GetBloomLevelAdjustment(card.BloomLevel, currentFocusLevel)
    
    return basePriority * graphAdjustment * bloomAdjustment
}
```

### 8.3 Memory State Update Function

After each review, the memory state is updated:

```go
// Update card memory state after review
func UpdateCardMemoryState(card *FlashCard, response Response, knowledgeGraph *Graph) {
    rating := ConvertResponseToRating(response)
    
    // 1. Update FSRS parameters using algorithm
    newState := fsrs.CalculateNextState(card.Difficulty, card.Stability, rating)
    card.Difficulty = newState.Difficulty
    card.Stability = newState.Stability
    
    // 2. Calculate next due date
    card.DueDate = CalculateNextDueDate(newState.Stability)
    
    // 3. Update concept mastery level
    UpdateConceptMastery(card.ConceptID, card.BloomLevel, rating, knowledgeGraph)
    
    // 4. If struggling, flag for potential remediation
    if rating <= 2 {
        // Potential knowledge gap, mark for analysis
        MarkForGapAnalysis(card.ConceptID, card.BloomLevel, knowledgeGraph)
    }
    
    // 5. If mastered, consider promoting to higher Bloom's level
    if rating >= 3 && IsMasteryThresholdMet(card) {
        ConsiderBloomLevelPromotion(card.ConceptID, card.BloomLevel, knowledgeGraph)
    }
}
```

## 9. Progressive Graph Completion

### 9.1 Detecting Potential Missing Prerequisites

The system analyzes review patterns to detect potential missing prerequisites:

```go
// Analyze review patterns to detect potential missing prerequisites
func DetectMissingPrerequisites(conceptID string, reviewHistory []ReviewRecord) []PotentialPrerequisite {
    // Identify concepts with consistently poor performance despite review
    consistentlyDifficult := IdentifyConsistentlyDifficultCards(conceptID, reviewHistory)
    
    // Find patterns in error types that suggest missing knowledge
    errorPatterns := AnalyzeErrorPatterns(reviewHistory)
    
    // Have LLM analyze patterns to suggest potential missing prerequisites
    potentialMissing := LLM.AnalyzePotentialMissingPrerequisites(
        conceptID, 
        consistentlyDifficult, 
        errorPatterns
    )
    
    return potentialMissing
}
```

### 9.2 Prioritizing Graph Expansion

The system prioritizes graph expansion based on review performance:

```go
// Prioritize graph expansion based on review performance
func PrioritizeGraphExpansion(knowledgeGraph *Graph, reviewData map[string][]ReviewRecord) []ExpansionPriority {
    priorities := []ExpansionPriority{}
    
    // Identify concepts with poor review performance
    strugglingConcepts := GetStrugglingConcepts(reviewData)
    
    for _, concept := range strugglingConcepts {
        // Check if concept has complete prerequisite coverage
        if HasIncompletePrerequisites(concept.ID, knowledgeGraph) {
            // Add to expansion priorities
            priorities = append(priorities, ExpansionPriority{
                ConceptID: concept.ID,
                Priority: CalculateExpansionPriority(concept, reviewData),
                RecommendedExpansion: SuggestExpansionAreas(concept, knowledgeGraph),
            })
        }
    }
    
    return priorities
}
```

### 9.3 Progressive Card Generation

When a knowledge gap is identified, new cards are generated:

```go
// Generate cards for a missing prerequisite concept
function GeneratePrerequisiteCards(conceptName, originalCardID) {
    // 1. Have LLM identify the core facets of this prerequisite concept
    conceptFacets := LLM.AnalyzeConceptComponents(conceptName)
    
    // 2. Generate a minimal set of flashcards covering essential understanding
    cards := []
    
    // Always create a Remember-level card for basic recall
    rememberCard := CreateCard(
        conceptName, 
        BloomRemember,
        LLM.GenerateRememberQuestion(conceptName),
        LLM.GenerateRememberAnswer(conceptName)
    )
    cards = append(cards, rememberCard)
    
    // Create an Understanding-level card as it's often needed for prerequisites
    understandCard := CreateCard(
        conceptName, 
        BloomUnderstand,
        LLM.GenerateUnderstandQuestion(conceptName),
        LLM.GenerateUnderstandAnswer(conceptName)
    )
    cards = append(cards, understandCard)
    
    // 3. Add these cards to the knowledge graph
    AddCardsToGraph(cards, conceptName, originalCardID)
    
    return cards
}
```

## 10. MCP Tool Interface

### 10.1 Core MCP Tools

The system exposes these core tools to the LLM:

```go
// Tool for analyzing study materials
analyzeStudyMaterialTool := mcp.NewTool("analyze_study_material",
    mcp.WithDescription(
        "Analyze learning materials to extract concepts, relationships, and structure. " +
        "USE: When student shares study materials, homework, or learning goals. " +
        "PROCESS: " +
        "1. Extract key concepts and their relationships " +
        "2. Identify learning objectives at appropriate levels " +
        "3. Create a structured knowledge representation " +
        "4. Generate appropriate study questions at different levels"
    ),
    mcp.WithString("content", mcp.Required()),
    mcp.WithString("subject_area", mcp.Description("Subject area if known")),
    mcp.WithString("grade_level", mcp.Description("Student's grade level if known")),
)

// Tool for creating flashcards
createFlashcardTool := mcp.NewTool("create_flashcard",
    mcp.WithDescription(
        "Create a new flashcard for a concept at a specific Bloom's level. " +
        "INTERNAL PROCESS: " +
        "1. Generate appropriate question for the target Bloom's level " +
        "2. Create answer and scaffolding resources " +
        "3. Add to knowledge graph " +
        "4. Associate with concept node"
    ),
    mcp.WithString("concept_id", mcp.Required()),
    mcp.WithString("bloom_level", mcp.Required()),
    mcp.WithString("question", mcp.Required()),
    mcp.WithString("answer", mcp.Required()),
    mcp.WithArray("scaffolding", mcp.Description("Scaffolding resources")),
)

// Tool for getting next card with prerequisite enforcement
getNextCardTool := mcp.NewTool("get_next_card",
    mcp.WithDescription(
        "Get the next optimal flashcard ensuring prerequisites are mastered. " +
        "PROCESS: " +
        "1. Identify cards due for review based on spaced repetition " +
        "2. Filter out cards whose prerequisites aren't mastered (score < 3) " +
        "3. If no valid cards exist, select the most critical prerequisite to review " +
        "4. Otherwise, select the highest priority card among valid candidates"
    ),
    mcp.WithString("student_id", mcp.Required()),
    mcp.WithString("session_id", mcp.Required()),
)

// Tool for recording student response
recordStudentResponseTool := mcp.NewTool("record_student_response",
    mcp.WithDescription(
        "Record and evaluate a student's response to a question. " +
        "USE: After asking a question and receiving the student's answer. " +
        "INTERNAL PROCESS: " +
        "1. Evaluate the response against expected answer " +
        "2. Assess understanding level (internally using Bloom's) " +
        "3. Update knowledge model " +
        "4. Determine appropriate feedback"
    ),
    mcp.WithString("session_id", mcp.Required()),
    mcp.WithString("card_id", mcp.Required()),
    mcp.WithString("student_response", mcp.Required()),
)
```

### 10.2 Gap Analysis Tools

```go
// Tool for dynamically analyzing knowledge gaps
analyzeKnowledgeGapsTool := mcp.NewTool("analyze_knowledge_gaps",
    mcp.WithDescription(
        "Analyze a student's knowledge gaps when they struggle with a concept. " +
        "INTERNAL PROCESS: " +
        "1. Identify the struggling concept and Bloom's level " +
        "2. Check mastery of prerequisites using graph relationships " +
        "3. Recursively test prerequisite concepts as needed " +
        "4. Generate a knowledge gap report with recommended focus areas"
    ),
    mcp.WithString("student_id", mcp.Required()),
    mcp.WithString("concept_id", mcp.Required()),
    mcp.WithString("bloom_level", mcp.Required()),
    mcp.WithString("error_pattern", mcp.Description("Observed error pattern if any")),
)

// Tool for suggesting graph extensions
suggestGraphExtensionTool := mcp.NewTool("suggest_graph_extension",
    mcp.WithDescription(
        "Analyze student's current understanding and suggest potential missing prerequisite concepts. " +
        "USE: When student struggles with a concept and existing cards don't address the gap. " +
        "PROCESS: " +
        "1. Analyze the nature of the student's misunderstanding " +
        "2. Identify potential missing prerequisite concepts " +
        "3. Suggest relationships to existing concepts " +
        "4. Recommend Bloom's levels for new flashcards"
    ),
    mcp.WithString("student_response", mcp.Required()),
    mcp.WithString("concept_id", mcp.Required()),
    mcp.WithString("bloom_level", mcp.Required()),
    mcp.WithArray("existing_concepts", mcp.Required()),
)
```

### 10.3 Scaffolding and Learning Strategy Tools

```go
// Tool for selecting appropriate scaffolding strategy
selectScaffoldingTool := mcp.NewTool("select_scaffolding",
    mcp.WithDescription(
        "Select appropriate scaffolding strategy based on the concept, Bloom's level, and student state. " +
        "INTERNAL PROCESS: " +
        "1. Analyze current mastery level and error patterns " +
        "2. Determine optimal scaffolding intensity " +
        "3. Select strategies from the scaffolding matrix " +
        "4. Generate scaffolding resources for LLM to use"
    ),
    mcp.WithString("student_id", mcp.Required()),
    mcp.WithString("concept_id", mcp.Required()),
    mcp.WithString("bloom_level", mcp.Required()),
    mcp.WithArray("previous_scaffolds", mcp.Description("Previously used scaffolds")),
)

// Tool for creating dependencies between cards
createDependencyTool := mcp.NewTool("create_dependency",
    mcp.WithDescription(
        "Create a prerequisite relationship between concepts. " +
        "PROCESS: " +
        "1. Establish that one concept is prerequisite to another " +
        "2. Update the knowledge graph structure " +
        "3. Adjust study scheduling to respect the new dependency"
    ),
    mcp.WithString("prerequisite_concept_id", mcp.Required()),
    mcp.WithString("dependent_concept_id", mcp.Required()),
    mcp.WithFloat("relationship_strength", mcp.Description("How strong the dependency is (0-1)")),
)
```

## 11. Typical User Flow

### 11.1 Initial Interaction

1. **Student Uploads Materials**:
   - Student shares study guide, homework, or describes what they need to learn
   - LLM calls `analyze_study_material` to extract concepts and relationships
   - System creates initial knowledge graph with LLM-suggested dependencies

2. **Initial Graph Construction**:
   - LLM calls `create_flashcard` for each identified concept
   - System constructs seed graph with initial dependencies
   - LLM explains to student they'll be using flashcards to learn

### 11.2 Study Session Flow

1. **Card Selection**:
   - LLM calls `get_next_card` to retrieve optimal next flashcard
   - System filters cards based on prerequisite mastery
   - If no cards are valid, system identifies a prerequisite to work on

2. **Presenting the Card**:
   - LLM presents question in a conversational, age-appropriate way
   - LLM doesn't mention knowledge graph, Bloom's level, or FSRS

3. **Processing Response**:
   - Student responds to the question
   - LLM calls `record_student_response` to evaluate the answer
   - System updates memory state and concept mastery

4. **Handling Struggles**:
   - If student struggles (rating <= 2), LLM calls `analyze_knowledge_gaps`
   - System identifies if prerequisites are missing or unmastered
   - LLM calls `suggest_graph_extension` if no clear prerequisites exist
   - LLM guides student to remediate prerequisites before returning to original concept

5. **Applying Scaffolding**:
   - If student struggles with a concept, LLM calls `select_scaffolding`
   - LLM applies scaffolding strategies conversationally
   - Scaffolding adapted to concept and Bloom's level

### 11.3 Progressive Knowledge Building

1. **Graph Expansion**:
   - As student struggles, system identifies missing prerequisites
   - LLM creates new flashcards for these prerequisites
   - Knowledge graph gradually becomes more complete and accurate

2. **Cognitive Advancement**:
   - As student masters concepts at lower Bloom's levels, system introduces higher levels
   - LLM calls `create_flashcard` to generate cards at appropriate new Bloom's levels

3. **Spaced Repetition Optimization**:
   - System schedules reviews based on FSRS algorithm
   - Reviews are prioritized based on both memory state and graph position
   - Student receives timely reviews to maintain memory while respecting prerequisite relationships

### 11.4 Internal System State Changes

During this interaction, the system undergoes these internal changes:

1. **Knowledge Graph Growth**:
   - Initial sparse graph expands to include missing prerequisites
   - Confidence scores on relationships increase with student interactions
   - Shadow graph concepts become part of explicit graph when confirmed

2. **Memory State Tracking**:
   - Each card updates FSRS parameters based on review performance
   - System tracks concept mastery at different Bloom's levels
   - Review history grows, enabling more accurate pattern detection

3. **Learning Path Optimization**:
   - System continually refines optimal learning path based on performance
   - Prerequisites are sequenced to maximize learning efficiency
   - Bloom's levels progress appropriately for each concept

## 12. Revealing Knowledge Gaps Through Patterns

### 12.1 Pattern Recognition Approach

The system identifies knowledge gaps through these pattern recognition approaches:

1. **Consistent Difficulty Analysis**:
   - Tracking concepts with consistently poor performance despite multiple reviews
   - Analyzing error patterns within related concept clusters
   - Detecting concepts where performance doesn't improve with review

2. **Error Pattern Classification**:
   - Categorizing types of errors (conceptual misunderstandings, procedural errors, etc.)
   - Mapping error types to potential missing prerequisites
   - Identifying systematic error patterns across multiple concepts

3. **Temporal Analysis**:
   - Analyzing how performance changes over time
   - Detecting concepts where retrievability decays unusually quickly
   - Identifying plateaus in learning progress

### 12.2 Implementation Algorithm

```go
// Analyze review patterns to detect knowledge gaps
func AnalyzeReviewPatternsForGaps(studentID string, conceptID string) []KnowledgeGap {
    // Get review history for this concept and related concepts
    reviewHistory := GetRelevantReviewHistory(studentID, conceptID)
    
    // Identify consistently difficult cards
    consistentlyDifficultCards := FindConsistentlyDifficultCards(reviewHistory)
    
    // Analyze error patterns
    errorPatterns := AnalyzeErrorTypes(reviewHistory)
    
    // Calculate learning curves
    learningCurves := CalculateLearningCurves(reviewHistory)
    
    // Identify anomalies in curves
    anomalies := FindLearningCurveAnomalies(learningCurves)
    
    // Have LLM analyze all patterns to suggest potential knowledge gaps
    suggestedGaps := LLM.AnalyzePatterns(
        consistentlyDifficultCards,
        errorPatterns,
        anomalies
    )
    
    // Score and rank the suggested gaps
    scoredGaps := ScoreKnowledgeGaps(suggestedGaps, reviewHistory)
    
    return scoredGaps
}
```

### 12.3 Leveraging LLM for Gap Analysis

The LLM plays a crucial role in gap analysis by:

1. Analyzing student responses to identify misconceptions
2. Suggesting potential missing knowledge based on domain expertise
3. Generating targeted diagnostic questions to isolate specific gaps
4. Recommending appropriate remediation strategies

## 13. Implementation Roadmap

### 13.1 Phase 1: Core Infrastructure

1. Implement basic knowledge graph structure
2. Integrate FSRS algorithm for card scheduling
3. Build MCP tool interface for LLM interaction
4. Create basic flowcharts for student interaction

### 13.2 Phase 2: Prerequisite Enforcement

1. Implement prerequisite checking for card selection
2. Build navigation algorithms for the knowledge graph
3. Create initial dependency suggestion capabilities
4. Develop scaffolding selection framework

### 13.3 Phase 3: Dynamic Graph Expansion

1. Implement pattern detection for knowledge gaps
2. Build dynamic card generation capabilities
3. Create graph expansion prioritization system
4. Develop confidence scoring for relationships

### 13.4 Phase 4: Advanced Features

1. Implement Bloom's level progression
2. Build advanced scaffolding strategies
3. Create learning analytics dashboard
4. Develop personalization capabilities

## 14. Conclusion

The Knowledge Graph-Based Flashcard System with FSRS Integration represents a significant advancement in educational technology. By combining the structural benefits of knowledge graphs, the cognitive progression of Bloom's Taxonomy, and the memory optimization of FSRS, we create a system that ensures students learn concepts in a logical sequence, at appropriate cognitive levels, and with optimal review timing.

The prerequisite-first approach ensures students build a solid foundation before tackling advanced concepts, while the dynamic graph expansion capability allows the system to function effectively even with initially incomplete knowledge graphs. By hiding this technical complexity beneath a friendly, conversational interface, we provide middle school students with a sophisticated yet accessible learning tool.

This design document provides a blueprint for implementing this system, with detailed algorithms, data structures, and interaction patterns. The result will be a powerful educational tool that adapts to each student's unique learning needs while optimizing for long-term retention and understanding.
