# Flashcards MCP - Prompting Improvements Implementation Plan

This document outlines the steps to implement the enhanced prompting strategy for the Flashcards MCP server, as detailed in `plans/prompting-improvements-design.md`. The goal is to embed detailed educational workflow instructions into the server information and tool descriptions to guide LLM behavior.

## Task 1: Update Server Info and Existing Tool Descriptions

### Background and Context
The current MCP server lacks explicit guidance for LLMs on the desired educational workflow. The design document specifies embedding this guidance directly into the server's initialization information (`serverInfo`) and the descriptions of existing tools. This task focuses on implementing these text-based changes in the server's main configuration file.

### My Task
1. Define the new `serverInfo` constant in `cmd/flashcards/main.go` with the exact text from the design document.
2. Update the `server.NewMCPServer` call to include `server.WithServerInfo()`.
3. Update the `mcp.WithDescription()` calls for the existing tools (`get_due_card`, `submit_review`, `create_card`, `update_card`, `list_cards`) in `cmd/flashcards/main.go` with the enhanced descriptions from the design document.

### Files to Create/Modify
1. `cmd/flashcards/main.go` - Modify server initialization and tool definitions.

### Implementation Details

#### 1. Define `serverInfo` Constant
Add a multi-line string constant named `flashcardsServerInfo` (or similar) at the top level of `cmd/flashcards/main.go` containing the exact text specified in section 3.1 of the design document.

```go
const flashcardsServerInfo = `
This is a spaced repetition flashcard system designed for middle school students. 
When using this server, always follow this precise educational workflow:

1. PRESENTATION PHASE:
   - Present only the front (question) side of the flashcard first
   - Never reveal the answer until after the student has attempted a response
   - Use an enthusiastic, encouraging tone with appropriate emojis
   - Make learning fun and exciting! 🤩 💪 🚀

2. RESPONSE PHASE:
   - Collect the student's answer attempt
   - Be supportive regardless of correctness
   - Use excited, age-appropriate language for middle schoolers

3. EVALUATION PHASE:
   - Show the correct answer only after student has responded
   - Compare the student's answer to the correct one with enthusiasm
   - For incorrect answers, explain the concept briefly in a friendly way
   - Ask a follow-up question to check understanding
   - Use many emojis and positive reinforcement! 🎯 ⭐ 🏆

4. RATING PHASE:
   - Automatically estimate difficulty using this criteria:
     * Rating 1: Answer was absent or completely wrong
     * Rating 2: Answer was partially correct or very vague
     * Rating 3: Answer was right but took >60 seconds or wasn't obvious from student's questions
     * Rating 4: Student answered correctly immediately
   - Only ask student how difficult it was if you can't confidently estimate
   - Ask informally: "How hard was that one for you?" rather than mentioning the 1-4 scale
   - Students who got answers wrong should ONLY receive ratings of 1 or 2
   - Use student's responses to gauge comprehension

5. TRANSITION PHASE:
   - Flow naturally to the next card to maintain engagement
   - Use transitional phrases like "Let's try another one!" or "Ready for the next challenge?"
   - Keep the energy high with enthusiastic language and emojis 🔥 ✨ 🎉

6. COMPLETION PHASE:
   - When out of cards, congratulate student on a great study session
   - Use extra enthusiastic celebration language and emojis 🎊 🎓 🥳
   - Propose brainstorming new cards together
   - When creating new cards, analyze what the student struggled with most
   - Identify prerequisite concepts they may be missing
   - Focus on fundamental knowledge common to multiple missed questions

Always maintain an excited, encouraging tone throughout the entire session using plenty of emojis!
`
```

#### 2. Update Server Initialization
Modify the `server.NewMCPServer` call to include the new server info:

```go
s := server.NewMCPServer(
    "Flashcards MCP",
    "1.0.0",
    server.WithServerInfo(flashcardsServerInfo), // Add this line
    server.WithResourceCapabilities(true, true),
    server.WithToolCapabilities(true),
    server.WithLogging(),
)
```

#### 3. Update Tool Descriptions
Locate the `mcp.NewTool` definitions for `get_due_card`, `submit_review`, `create_card`, `update_card`, and `list_cards`. Replace the existing string passed to `mcp.WithDescription()` with the corresponding enhanced description string provided in section 4.1 of the design document.

Example for `get_due_card`:
```go
getDueCardTool := mcp.NewTool("get_due_card",
    mcp.WithDescription(
        "Get the next flashcard due for review with statistics. " +
        "IMPORTANT EDUCATIONAL WORKFLOW: " +
        "1. Show ONLY the front (question) side of the card to the student 📝 " +
        "2. DO NOT reveal the back (answer) side at this stage ⚠️ " +
        "3. Ask the student to attempt to recall and provide their answer 🤔 " +
        "4. Use an enthusiastic, excited tone with plenty of emojis 🚀 " +
        "5. Make it fun and engaging for middle school students! 🎮 " +
        "6. NEVER show both sides of the card simultaneously at this phase ❌ " +
        "This follows proven spaced repetition methodology for effective learning."
    ),
    // Existing parameters...
)
```
Repeat this process for all five existing tools.

### Test Implementation Requirements
- The existing integration tests in `cmd/flashcards/main_test.go` primarily verify tool functionality and data structures, not the descriptive text itself.
- **Manual Verification**: Add a step to manually inspect the server's initialization message and tool definitions (e.g., by connecting with a simple MCP client or examining logs if available) to confirm the new `serverInfo` and descriptions are correctly set.
- **Optional**: Consider adding a simple test case in `main_test.go` that retrieves the server's capabilities or tool definitions (if the MCP client library supports this) and asserts that the description strings match the expected enhanced text. This provides some automated verification that the text is being applied.

### Success Criteria
- [ ] `serverInfo` constant is defined correctly in `cmd/flashcards/main.go`.
- [ ] `server.NewMCPServer` call includes `server.WithServerInfo()`.
- [ ] Descriptions for `get_due_card`, `submit_review`, `create_card`, `update_card`, `list_cards` are updated with the enhanced text from the design document.
- [ ] Existing integration tests in `main_test.go` still pass (ensuring no functional regressions).
- [ ] Manual verification confirms the new text is present in server info and tool descriptions.

## Task 2: Add `help_analyze_learning` Tool

### Background and Context
The design document suggests an optional new tool, `help_analyze_learning`, to further guide the LLM in analyzing student progress and suggesting improvements. This task involves defining this tool and registering it with a placeholder handler.

### My Task
1. Define the `help_analyze_learning` tool in `cmd/flashcards/main.go` using `mcp.NewTool`, including its description from the design document.
2. Register the new tool using `s.AddTool`, associating it with a placeholder handler function.

### Files to Create/Modify
1. `cmd/flashcards/main.go` - Add tool definition and registration.
2. `cmd/flashcards/handlers.go` - Add placeholder handler function (optional, could be an inline func).

### Implementation Details

#### 1. Define `help_analyze_learning` Tool
Add the following tool definition in `cmd/flashcards/main.go`:

```go
helpAnalyzeLearningTool := mcp.NewTool(
    "help_analyze_learning",
    mcp.WithDescription(
        "Analyze the student's learning progress and suggest improvements. " +
        "IMPORTANT EDUCATIONAL GUIDANCE: " +
        "1. Review the student's performance across all cards 📊 " +
        "2. Identify patterns in what concepts are challenging 🧩 " +
        "3. Suggest new cards that would help with prerequisite knowledge 💡 " +
        "4. Look for fundamental concepts that apply across multiple difficult cards 🔍 " +
        "5. Explain your analysis enthusiastically and supportively 🚀 " +
        "6. Use many emojis and exciting middle-school appropriate language 🤩 " +
        "7. Get the student excited about mastering these concepts! 💪 " +
        "8. Frame challenges as opportunities for growth, not as failures ✨ " +
        "9. Suggest specific strategies tailored to their learning patterns 🎯"
    ),
    // No parameters defined for this tool initially
)
```

#### 2. Register Tool with Placeholder Handler
Add the registration call within the `main` function:

```go
// Placeholder handler for help_analyze_learning
func handleHelpAnalyzeLearning(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Placeholder implementation - does nothing functional yet
    log.Printf("Placeholder handler called for help_analyze_learning")
    // Return an empty result or a message indicating it's not implemented
    return mcp.NewToolResultText(`{"message": "Tool 'help_analyze_learning' is defined but not yet implemented."}`), nil
}

// In main():
s.AddTool(helpAnalyzeLearningTool, handleHelpAnalyzeLearning) // Or use an inline func
```

### Test Implementation Requirements
- Add a new test case to `cmd/flashcards/main_test.go` that calls the `help_analyze_learning` tool.
- Verify that the tool call succeeds and returns the expected placeholder response (e.g., the "not yet implemented" message).
- **Manual Verification**: Confirm the tool appears in the server's list of available tools when connecting with an MCP client.

### Success Criteria
- [ ] `help_analyze_learning` tool is defined with the correct description in `cmd/flashcards/main.go`.
- [ ] The tool is registered using `s.AddTool` with a placeholder handler.
- [ ] A new test case verifies that the tool can be called and returns the placeholder response.
- [ ] Existing tests pass.

## Task 3: Define Manual Testing Strategy

### Background and Context
As identified during planning, the effectiveness of the embedded prompting instructions cannot be fully verified through automated tests within the Go project. A manual testing strategy is required to assess whether an LLM client correctly interprets and follows the new guidance.

### My Task
Outline a manual testing strategy to validate the prompting improvements. This strategy should be documented within this plan but executed separately after the code changes are deployed.

### Files to Create/Modify
- None (This task defines the strategy, documentation is part of this plan).

### Implementation Details

#### Manual Testing Steps:
1.  **Setup**:
    *   Build and run the modified Flashcards MCP server.
    *   Connect to the server using an MCP-compatible LLM client (e.g., a development environment or a testing tool).
2.  **Server Info Verification**:
    *   Observe the initial information provided by the client about the server. Confirm that the detailed educational workflow from `serverInfo` is presented to the LLM (or is accessible to it).
3.  **Tool Description Verification**:
    *   Initiate interactions that would require using each tool (`get_due_card`, `submit_answer` (via `submit_review`), `create_card`, etc.).
    *   Observe the tool descriptions presented to the LLM by the client. Confirm they match the enhanced descriptions containing the workflow instructions.
4.  **Workflow Execution Testing**:
    *   **`get_due_card`**: Instruct the LLM to start a review session. Verify it only presents the *front* of the card and asks for the student's answer, using an enthusiastic tone as instructed. Check that it avoids showing the back.
    *   **`submit_review` (Response & Evaluation)**: Provide an answer (correct or incorrect). Verify the LLM shows the correct answer *after* the attempt, compares enthusiastically, explains if incorrect, asks a follow-up, and attempts to estimate difficulty based on the criteria. Check for appropriate tone and emoji usage. Verify it only asks "How hard was that?" if estimation is difficult and avoids mentioning the 1-4 scale directly. Ensure incorrect answers get ratings 1 or 2.
    *   **`submit_review` (Transition)**: After rating, verify the LLM uses enthusiastic transitional phrases and emojis to move to the next card (if available).
    *   **`get_due_card` (Completion)**: When no more cards are due, verify the LLM congratulates the student enthusiastically and proposes creating new cards based on difficult topics.
    *   **`create_card`**: Instruct the LLM to create a new card. Verify it attempts to follow the guidance on analyzing past struggles and focusing on fundamentals, using an enthusiastic tone.
    *   **`update_card`**: Instruct the LLM to update a card. Verify it focuses on clarity/accuracy and uses an enthusiastic tone.
    *   **`list_cards`**: Instruct the LLM to list cards. Verify it defaults to showing only the front unless asked otherwise and uses an encouraging tone.
    *   **`help_analyze_learning`**: Instruct the LLM to analyze progress. Verify it attempts to follow the guidance on identifying patterns, suggesting prerequisites, and using a supportive, enthusiastic tone.
5.  **Tone and Emoji Usage**: Throughout the session, monitor the LLM's language, tone, and emoji usage. Verify it consistently maintains the enthusiastic, encouraging, middle-school-appropriate style specified.
6.  **Edge Cases**: Test scenarios like providing vague answers, taking a long time to answer, or having no cards in the system.

### Test Implementation Requirements
- This section describes the *manual* testing process. No automated tests are created for this task.

### Success Criteria
- [ ] A clear manual testing plan is documented.
- [ ] The plan covers verification of `serverInfo` and all enhanced tool descriptions.
- [ ] The plan includes steps to test the LLM's adherence to each phase of the specified educational workflow (Presentation, Response, Evaluation, Rating, Transition, Completion, Creation).
- [ ] The plan includes checks for the specified tone and emoji usage.

## Implementation Sequence Summary

1.  **Task 1**: Implement changes to `serverInfo` and existing tool descriptions in `cmd/flashcards/main.go`. Verify text application and ensure existing tests pass.
2.  **Task 2**: Define and register the new `help_analyze_learning` tool with a placeholder handler in `cmd/flashcards/main.go`. Add a test to verify the placeholder.
3.  **Task 3**: (Documentation Complete) The manual testing strategy is defined within this plan. Execution occurs post-deployment.