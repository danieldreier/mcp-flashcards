package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/danieldreier/mcp-flashcards/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

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

func main() {
	// Parse command-line flags
	filePath := flag.String("file", "./flashcards.json", "Path to flashcard data file")
	flag.Parse()

	// Initialize storage
	fileStorage := storage.NewFileStorage(*filePath)
	if err := fileStorage.Load(); err != nil {
		fmt.Printf("Error loading storage: %v\n", err)
		os.Exit(1)
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"Flashcards MCP",
		"1.0.0",
		server.WithInstructions(flashcardsServerInfo), // Provide educational workflow guidance
		server.WithResourceCapabilities(true, true),   // Resource capabilities for subscribe and listChanged
		server.WithToolCapabilities(true),             // Enable tool capabilities
		server.WithLogging(),                          // Enable logging for the server
	)

	// Initialize the flashcard service
	flashcardService := NewFlashcardService(fileStorage)

	// Create context with the service for tool handlers
	ctx := context.WithValue(context.Background(), "service", flashcardService)

	// Define the get_due_card tool
	getDueCardTool := mcp.NewTool("get_due_card",
		mcp.WithDescription(
			"Get the next flashcard due for review with statistics. "+
				"IMPORTANT EDUCATIONAL WORKFLOW: "+
				"1. Show ONLY the front (question) side of the card to the student 📝 "+
				"2. DO NOT reveal the back (answer) side at this stage ⚠️ "+
				"3. Ask the student to attempt to recall and provide their answer 🤔 "+
				"4. Use an enthusiastic, excited tone with plenty of emojis 🚀 "+
				"5. Make it fun and engaging for middle school students! 🎮 "+
				"6. NEVER show both sides of the card simultaneously at this phase ❌ "+
				"This follows proven spaced repetition methodology for effective learning.",
		),
		// No parameters required for now
	)

	// Define the submit_review tool
	submitReviewTool := mcp.NewTool("submit_review",
		mcp.WithDescription(
			"Submit the student's answer for evaluation. "+
				"IMPORTANT EDUCATIONAL WORKFLOW: "+
				"1. Now that the student has provided their answer, show the correct answer 📝 "+
				"2. Compare the student's answer with the correct one supportively and enthusiastically 🎯 "+
				"3. For incorrect answers, briefly explain the concept in a friendly way 🤗 "+
				"4. Ask a quick follow-up question to check understanding 🧩 "+
				"5. Use their response to gauge comprehension 📊 "+
				"6. Automatically estimate difficulty rating using these criteria: "+
				"   • Rating 1: Answer was absent or completely wrong ❌ "+
				"   • Rating 2: Answer was partially correct or very vague 🤏 "+
				"   • Rating 3: Answer was right but took >60 seconds or student indicated difficulty 🕒 "+
				"   • Rating 4: Student answered correctly immediately ⚡ "+
				"7. Only if you can't confidently estimate, ask informally: 'How hard was that one for you?' 🤔 "+
				"8. Students who got answers wrong should ONLY receive ratings of 1 or 2 ⚠️ "+
				"9. Use LOTS of emojis and an excited, middle school appropriate tone! 🎉",
		),
		// Define parameters
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the card being reviewed"),
		),
		mcp.WithNumber("rating",
			mcp.Required(),
			mcp.Description("Rating from 1-4: Again=1, Hard=2, Good=3, Easy=4"),
		),
		mcp.WithString("answer",
			mcp.Description("The answer provided by the user"),
		),
	)

	// Define the create_card tool
	createCardTool := mcp.NewTool("create_card",
		mcp.WithDescription(
			"Propose a new flashcard to the student based on learning analysis. "+
				"IMPORTANT CONFIRMATION WORKFLOW: "+
				"1. Propose the card details (front, back, tags) to the user for review FIRST. 🤔 "+
				"2. Ask the user explicitly if they approve creating this card. 👍👎 "+
				"3. ONLY call this tool if the user confirms approval. ✅ "+
				"4. If the user suggests changes, incorporate them and ask for approval again. 🔄 "+
				"CREATIVE GUIDANCE (when proposing the card): "+
				"1. Analyze what topics the student struggled with most in previous cards 📊 "+
				"2. Identify prerequisite concepts they may be missing 🧩 "+
				"3. Focus on fundamental knowledge that applies to multiple missed questions 🔍 "+
				"4. Create cards that build scaffolding for harder concepts 🏗️ "+
				"5. Make questions clear, specific, and targeted 🎯 "+
				"6. Keep answers concise but complete 📝 "+
				"7. Each card should test a single concept 🧠 "+
				"8. Use an enthusiastic tone when discussing the new cards with the student! 🚀 "+
				"9. Get the student excited about learning these new concepts 🤩",
		),
		// Define parameters
		mcp.WithString("front",
			mcp.Required(),
			mcp.Description("The front text of the card"),
		),
		mcp.WithString("back",
			mcp.Required(),
			mcp.Description("The back text of the card"),
		),
		mcp.WithArray("tags",
			mcp.Description("Tags for categorizing the card"),
		),
	)

	// Define the update_card tool
	updateCardTool := mcp.NewTool("update_card",
		mcp.WithDescription(
			"Update an existing flashcard. "+
				"IMPORTANT EDUCATIONAL GUIDANCE: "+
				"1. Preserve the educational intent of the card 🎓 "+
				"2. Improve clarity or accuracy to aid learning 🔍 "+
				"3. Consider making the card more engaging for middle school students 🎮 "+
				"4. Use enthusiastic language when discussing the improvements 🚀 "+
				"5. Get the student excited about the enhanced card! 🤩",
		),
		// Define parameters
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the card to update"),
		),
		mcp.WithString("front",
			mcp.Description("The new front text of the card"),
		),
		mcp.WithString("back",
			mcp.Description("The new back text of the card"),
		),
		mcp.WithArray("tags",
			mcp.Description("New tags for the card"),
		),
	)

	// Define the delete_card tool
	deleteCardTool := mcp.NewTool("delete_card",
		mcp.WithDescription("Delete a flashcard"),
		// Define parameters
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the card to delete"),
		),
	)

	// Define the list_cards tool
	listCardsTool := mcp.NewTool("list_cards",
		mcp.WithDescription(
			"List all flashcards, optionally filtered by tags. "+
				"IMPORTANT EDUCATIONAL GUIDANCE: "+
				"1. When displaying cards to the student, prefer to show only the question side "+
				"   unless the student specifically requests to see both sides 📝 "+
				"2. Use this data to identify patterns in what the student finds challenging 🔍 "+
				"3. Look for gaps in prerequisite knowledge based on difficult cards 🧩 "+
				"4. Maintain an enthusiastic, encouraging tone when discussing the cards 🚀 "+
				"5. Use plenty of emojis and positive language! 🤩 ✨ 💪",
		),
		// Define parameters
		mcp.WithArray("tags",
			mcp.Description("Filter cards by tags"),
		),
		mcp.WithBoolean("include_stats",
			mcp.Description("Include statistics in the response"),
		),
	)

	// Register all tools with their handlers
	s.AddTool(getDueCardTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Pass the context with service to the handler
		return handleGetDueCard(ctx, request)
	})
	s.AddTool(submitReviewTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSubmitReview(ctx, request)
	})
	s.AddTool(createCardTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateCard(ctx, request)
	})
	s.AddTool(updateCardTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateCard(ctx, request)
	})
	s.AddTool(deleteCardTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteCard(ctx, request)
	})
	s.AddTool(listCardsTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListCards(ctx, request)
	})

	// Define the help_analyze_learning tool
	helpAnalyzeLearningTool := mcp.NewTool(
		"help_analyze_learning",
		mcp.WithDescription(
			"Analyze the student's learning progress and suggest improvements. "+
				"IMPORTANT EDUCATIONAL GUIDANCE: "+
				"1. Review the student's performance across all cards 📊 "+
				"2. Identify patterns in what concepts are challenging 🧩 "+
				"3. Suggest new cards that would help with prerequisite knowledge 💡 "+
				"4. Look for fundamental concepts that apply across multiple difficult cards 🔍 "+
				"5. Explain your analysis enthusiastically and supportively 🚀 "+
				"6. Use many emojis and exciting middle-school appropriate language 🤩 "+
				"7. Get the student excited about mastering these concepts! 💪 "+
				"8. Frame challenges as opportunities for growth, not as failures ✨ "+
				"9. Suggest specific strategies tailored to their learning patterns 🎯",
		),
		// No parameters defined for this tool initially
	)

	// Register the help_analyze_learning tool with the implemented handler
	s.AddTool(helpAnalyzeLearningTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Pass the context with service to the handler
		return handleHelpAnalyzeLearning(ctx, request)
	})

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Error serving MCP server: %v", err)
	}
}
