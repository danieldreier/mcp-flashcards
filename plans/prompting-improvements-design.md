# Flashcards MCP - Enhanced Prompting Design

## 1. Overview

This document outlines the technical specification for enhancing the Flashcards Model Context Protocol (MCP) server with improved prompting strategies. The current implementation has functional issues where the LLM sometimes:

- Shows both question and answer simultaneously
- Fails to wait for student response before showing the answer
- Incorrectly prompts for difficulty ratings at inappropriate times
- Uses inconsistent educational tone

These issues occur because the MCP protocol itself doesn't inherently provide the LLM with guidance on the educational workflow expected in a spaced repetition system. This design focuses solely on embedding effective prompting instructions through server information and tool descriptions, without relying on system prompts or sampling capabilities that aren't universally supported across MCP clients.

## 2. Design Approach

The approach will use the "Tool Description + Instructions Embedding" pattern to guide LLM behavior through:

1. **Server initialization information** - Provide behavioral guidance in the server's metadata
2. **Tool descriptions** - Include detailed workflow instructions in each tool description

This design takes advantage of the fact that MCP clients communicate these descriptions to the LLM before tool use, allowing us to influence behavior without direct system prompt access.

## 3. Server Initialization Enhancement

### 3.1 Server Information Field

The MCP specification includes a `serverInfo` field during initialization that clients may use to improve the LLM's understanding of available capabilities. We'll enhance this field with educational workflow instructions:

```go
// Define server info with explicit educational workflow guidance
serverInfo := `
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

// Create the MCP server with enhanced server info
server := mcp.NewServer(
    "Flashcards MCP",
    "1.0.0",
    mcp.WithServerInfo(serverInfo),
)
```

## 4. Tool Description Enhancements

### 4.1 Enhanced Tool Descriptions

Each tool will have its description enhanced with explicit workflow instructions:

```go
// Enhanced get_due_card tool
getDueCardTool := mcp.NewTool(
    "get_due_card", 
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
)

// Enhanced submit_answer tool
submitAnswerTool := mcp.NewTool(
    "submit_answer",
    mcp.WithDescription(
        "Submit the student's answer for evaluation. " +
        "IMPORTANT EDUCATIONAL WORKFLOW: " +
        "1. Now that the student has provided their answer, show the correct answer 📝 " +
        "2. Compare the student's answer with the correct one supportively and enthusiastically 🎯 " +
        "3. For incorrect answers, briefly explain the concept in a friendly way 🤗 " +
        "4. Ask a quick follow-up question to check understanding 🧩 " +
        "5. Use their response to gauge comprehension 📊 " +
        "6. Automatically estimate difficulty rating using these criteria: " +
        "   • Rating 1: Answer was absent or completely wrong ❌ " +
        "   • Rating 2: Answer was partially correct or very vague 🤏 " +
        "   • Rating 3: Answer was right but took >60 seconds or student indicated difficulty 🕒 " +
        "   • Rating 4: Student answered correctly immediately ⚡ " +
        "7. Only if you can't confidently estimate, ask informally: 'How hard was that one for you?' 🤔 " +
        "8. Students who got answers wrong should ONLY receive ratings of 1 or 2 ⚠️ " +
        "9. Use LOTS of emojis and an excited, middle school appropriate tone! 🎉"
    ),
    mcp.WithString("card_id", 
        mcp.Required(),
        mcp.Description("ID of the card being reviewed"),
    ),
    mcp.WithString("answer",
        mcp.Required(),
        mcp.Description("The student's answer attempt"),
    ),
)

// Enhanced submit_review tool
submitReviewTool := mcp.NewTool(
    "submit_review",
    mcp.WithDescription(
        "Submit a review rating for a flashcard. " +
        "IMPORTANT EDUCATIONAL WORKFLOW: " +
        "1. Process the rating to schedule the next appearance of this card ⏱️ " +
        "2. Flow naturally into the next card with an enthusiastic transition 🌊 " +
        "3. Use phrases like 'Let's try another one!' or 'Ready for the next challenge?' 💪 " +
        "4. Keep the energy high with positive language and many emojis! 🔥 ✨ 🎉 " +
        "5. If this was the last card, celebrate their study session enthusiastically 🎊 " +
        "6. When out of cards, congratulate with extra enthusiasm: 'Amazing job completing your review session!' 🏆 " +
        "7. Then propose brainstorming new cards: 'Let's create some new cards to help with concepts you found challenging!' 💡"
    ),
    mcp.WithString("card_id",
        mcp.Required(),
        mcp.Description("ID of the card being rated"),
    ),
    mcp.WithNumber("rating",
        mcp.Required(),
        mcp.Description("Rating from 1-4 (Again=1, Hard=2, Good=3, Easy=4)"),
        mcp.MinValue(1),
        mcp.MaxValue(4),
    ),
    mcp.WithString("answer",
        mcp.Description("Final record of the student's answer (optional)"),
    ),
)

// Enhanced create_card tool
createCardTool := mcp.NewTool(
    "create_card",
    mcp.WithDescription(
        "Create a new flashcard. " +
        "IMPORTANT CREATIVE GUIDANCE: " +
        "1. Analyze what topics the student struggled with most in previous cards 📊 " +
        "2. Identify prerequisite concepts they may be missing 🧩 " +
        "3. Focus on fundamental knowledge that applies to multiple missed questions 🔍 " +
        "4. Create cards that build scaffolding for harder concepts 🏗️ " +
        "5. Make questions clear, specific, and targeted 🎯 " +
        "6. Keep answers concise but complete 📝 " +
        "7. Each card should test a single concept 🧠 " +
        "8. Use an enthusiastic tone when discussing the new cards with the student! 🚀 " +
        "9. Get the student excited about learning these new concepts 🤩"
    ),
    mcp.WithString("front",
        mcp.Required(),
        mcp.Description("Question side of the flashcard"),
    ),
    mcp.WithString("back",
        mcp.Required(),
        mcp.Description("Answer side of the flashcard"),
    ),
    mcp.WithArray("tags",
        mcp.Description("Optional tags for categorizing the flashcard"),
        mcp.Items(mcp.String()),
    ),
)

// Enhanced update_card tool
updateCardTool := mcp.NewTool(
    "update_card",
    mcp.WithDescription(
        "Update an existing flashcard. " +
        "IMPORTANT EDUCATIONAL GUIDANCE: " +
        "1. Preserve the educational intent of the card 🎓 " +
        "2. Improve clarity or accuracy to aid learning 🔍 " +
        "3. Consider making the card more engaging for middle school students 🎮 " +
        "4. Use enthusiastic language when discussing the improvements 🚀 " +
        "5. Get the student excited about the enhanced card! 🤩"
    ),
    mcp.WithString("card_id",
        mcp.Required(),
        mcp.Description("ID of the card to update"),
    ),
    mcp.WithString("front",
        mcp.Description("New question side of the flashcard"),
    ),
    mcp.WithString("back",
        mcp.Description("New answer side of the flashcard"),
    ),
    mcp.WithArray("tags",
        mcp.Description("Updated tags for the flashcard"),
        mcp.Items(mcp.String()),
    ),
)

// Enhanced list_cards tool
listCardsTool := mcp.NewTool(
    "list_cards",
    mcp.WithDescription(
        "List all flashcards, optionally filtered by tags. " +
        "IMPORTANT EDUCATIONAL GUIDANCE: " +
        "1. When displaying cards to the student, prefer to show only the question side " +
        "   unless the student specifically requests to see both sides 📝 " +
        "2. Use this data to identify patterns in what the student finds challenging 🔍 " +
        "3. Look for gaps in prerequisite knowledge based on difficult cards 🧩 " +
        "4. Maintain an enthusiastic, encouraging tone when discussing the cards 🚀 " +
        "5. Use plenty of emojis and positive language! 🤩 ✨ 💪"
    ),
    mcp.WithArray("tags",
        mcp.Description("Optional list of tags to filter cards"),
        mcp.Items(mcp.String()),
    ),
    mcp.WithBoolean("include_stats",
        mcp.Description("Whether to include statistics in the response"),
    ),
)

// Optional: Add a specialized help_analyze_learning tool
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
)
```

## 5. Implementation Recommendations

For optimal results, implement both enhanced server information and tool descriptions. The redundancy ensures that the LLM receives multiple signals about the expected educational workflow and tone.

Consider adding specialized tools like `help_analyze_learning` to encourage better learning behavior beyond the core flashcard functionality.

## 6. Expected Improvements

This enhanced prompting approach should address the current issues by:

1. **Preventing premature answer reveals** through explicit instructions never to show both sides simultaneously in the presentation phase
2. **Ensuring proper rating assignment** by providing clear criteria for automatically estimating difficulty
3. **Improving educational value** by instructing the LLM to explain concepts and check comprehension with follow-up questions
4. **Creating a more engaging experience** through instructions to use an enthusiastic tone with emojis
5. **Building a cohesive review flow** by guiding the LLM on how to transition between cards
6. **Addressing knowledge gaps** by providing guidance on analyzing difficulties and creating appropriate new cards

## 7. Limitations

This approach relies solely on the LLM interpreting and following the embedded instructions. Without state enforcement, there's still a possibility of workflow violations, though the redundant prompting should minimize this risk.

Some LLM clients may truncate or modify tool descriptions, potentially reducing the effectiveness of the embedded instructions.

## 8. Future Enhancements

If these prompting enhancements alone are insufficient, consider implementing the state enforcement approach as a complementary strategy to further ensure correct educational workflow.
