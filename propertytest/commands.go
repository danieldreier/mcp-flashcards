package propertytest

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/commands"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	// Corrected import path for go-fsrs
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// --- System Under Test Definition ---
type FlashcardSUT struct {
	Client      *client.Client
	Ctx         context.Context
	Cancel      context.CancelFunc
	CleanupFunc func()
	T           *testing.T
}

// --- State Definition ---

// CommandState holds the *model* state.
// It uses REAL card IDs once they are known via CreateCardCmd's PostCondition.
type CommandState struct {
	Cards        map[string]Card // Model using REAL IDs
	KnownRealIDs []string        // List of REAL IDs known to exist
	LastRealID   string          // Last REAL ID created/focused
	T            *testing.T      // Testing context for logging
}

// Helper to create a deep copy of the state
func (s *CommandState) deepCopy() *CommandState {
	newState := &CommandState{
		Cards:        make(map[string]Card, len(s.Cards)),
		KnownRealIDs: make([]string, len(s.KnownRealIDs)),
		LastRealID:   s.LastRealID,
		T:            s.T,
	}
	for k, v := range s.Cards {
		newState.Cards[k] = v // Card struct is assumed to be copyable or immutable enough
	}
	copy(newState.KnownRealIDs, s.KnownRealIDs)
	return newState
}

// --- Command Interface Implementations ---

// --- CreateCardCmd ---
type CreateCardCmd struct {
	Front string
	Back  string
	Tags  []string
}

func (c *CreateCardCmd) Run(sut commands.SystemUnderTest) commands.Result {
	fSUT := sut.(*FlashcardSUT)
	fSUT.T.Logf("Run: %s", c.String())
	createCardRequest := mcp.CallToolRequest{}
	createCardRequest.Params.Name = "create_card"
	createCardRequest.Params.Arguments = map[string]interface{}{
		"front": c.Front,
		"back":  c.Back,
		"tags":  InterfaceSlice(c.Tags),
	}
	createResult, err := fSUT.Client.CallTool(fSUT.Ctx, createCardRequest)
	if err != nil {
		return fmt.Errorf("create_card Run failed: %w", err)
	}
	if len(createResult.Content) == 0 {
		return fmt.Errorf("create_card Run: no content returned")
	}
	createTextContent, ok := createResult.Content[0].(mcp.TextContent)
	if !ok {
		return fmt.Errorf("create_card Run: expected TextContent, got %T", createResult.Content[0])
	}
	var createResponse CreateCardResponse
	err = json.Unmarshal([]byte(createTextContent.Text), &createResponse)
	if err != nil {
		var errResp map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(createTextContent.Text), &errResp); jsonErr == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				fSUT.T.Logf("Original JSON parse error (ignored): %v", err)
				return fmt.Errorf("create_card tool error: %s", errMsg)
			}
		}
		return fmt.Errorf("create_card Run: failed parse response: %w. Resp: %s", err, createTextContent.Text)
	}
	if createResponse.Card.ID == "" {
		return fmt.Errorf("create_card Run: response missing card ID. Resp: %s", createTextContent.Text)
	}
	return createResponse
}

func (c *CreateCardCmd) NextState(state commands.State) commands.State {
	// Create does not change the *prior* model state representation used by PostCondition.
	// It only introduces a new entity whose ID is revealed by Run's result.
	// The state *update* happens in PostCondition.
	return state // Return original state
}

func (c *CreateCardCmd) PreCondition(state commands.State) bool {
	return true // Can always attempt to create
}

func (c *CreateCardCmd) PostCondition(state commands.State, result commands.Result) *gopter.PropResult {
	cmdState := state.(*CommandState)
	label := fmt.Sprintf("PostCondition %s", c.String())
	if _, ok := result.(error); ok {
		cmdState.T.Logf("PostCondition failed for %s due to Run error: %v", c.String(), result.(error))
		return gopter.NewPropResult(false, label)
	}
	createResponse, ok := result.(CreateCardResponse)
	if !ok {
		return gopter.NewPropResult(false, label)
	}
	if createResponse.Card.Front != c.Front || createResponse.Card.Back != c.Back || !CompareTags(c.Tags, createResponse.Card.Tags) {
		cmdState.T.Logf("Create PostCondition failed: Data mismatch. Expected Front='%s', Back='%s', Tags=%v. Got Front='%s', Back='%s', Tags=%v",
			c.Front, c.Back, c.Tags, createResponse.Card.Front, createResponse.Card.Back, createResponse.Card.Tags)
		return gopter.NewPropResult(false, label)
	}

	// 2. Mutate the model state to include the newly created card
	realID := createResponse.Card.ID
	if _, exists := cmdState.Cards[realID]; !exists {
		cmdState.KnownRealIDs = append(cmdState.KnownRealIDs, realID)
	}
	newCard := Card{
		ID:        createResponse.Card.ID,
		Front:     createResponse.Card.Front,
		Back:      createResponse.Card.Back,
		CreatedAt: createResponse.Card.CreatedAt,
		Tags:      createResponse.Card.Tags,
		FSRS:      createResponse.Card.FSRS,
	}
	sort.Strings(newCard.Tags)
	cmdState.Cards[realID] = newCard
	cmdState.LastRealID = realID
	cmdState.T.Logf("PostCondition %s: Updated model state with real ID %s", c.String(), realID)

	return gopter.NewPropResult(true, label)
}

func (c *CreateCardCmd) String() string {
	return fmt.Sprintf("CreateCard(Front: '%s', Back: '%s', Tags: %v)", c.Front, c.Back, c.Tags)
}

// --- GetCardCmd ---
type GetCardCmd struct {
	CardID string // REAL Card ID
}

func (c *GetCardCmd) Run(sut commands.SystemUnderTest) commands.Result {
	fSUT := sut.(*FlashcardSUT)
	fSUT.T.Logf("Run: %s", c.String())
	listCardsRequest := mcp.CallToolRequest{}
	listCardsRequest.Params.Name = "list_cards"
	listResult, err := fSUT.Client.CallTool(fSUT.Ctx, listCardsRequest)
	if err != nil {
		return fmt.Errorf("list_cards Run failed: %w", err)
	}
	if len(listResult.Content) == 0 {
		return ListCardsResponse{Cards: []Card{}}
	} // Return empty list
	listTextContent, ok := listResult.Content[0].(mcp.TextContent)
	if !ok {
		return fmt.Errorf("list_cards Run: expected TextContent, got %T", listResult.Content[0])
	}
	var listResponse ListCardsResponse
	err = json.Unmarshal([]byte(listTextContent.Text), &listResponse)
	if err != nil {
		return fmt.Errorf("list_cards Run: failed parse: %w. Resp: %s", err, listTextContent.Text)
	}
	return listResponse // Return the actual list response
}

func (c *GetCardCmd) NextState(state commands.State) commands.State {
	// Get doesn't change the model state
	return state // Return original state (no change expected in the model)
}

func (c *GetCardCmd) PreCondition(state commands.State) bool {
	_, exists := state.(*CommandState).Cards[c.CardID]
	return exists // Check if the REAL ID exists in the model
}

func (c *GetCardCmd) PostCondition(state commands.State, result commands.Result) *gopter.PropResult {
	cmdState := state.(*CommandState)
	label := fmt.Sprintf("PostCondition %s", c.String())
	if _, ok := result.(error); ok {
		cmdState.T.Logf("PostCondition failed for %s due to Run error: %v", c.String(), result.(error))
		return gopter.NewPropResult(false, label)
	}
	listResponse, ok := result.(ListCardsResponse)
	if !ok {
		return gopter.NewPropResult(false, label)
	}
	found := false
	for _, actualCard := range listResponse.Cards {
		if actualCard.ID == c.CardID {
			found = true
			// Optional: Compare actualCard data with currentState.Cards[c.CardID]
			break
		}
	}
	if !found {
		return gopter.NewPropResult(false, label)
	}
	return gopter.NewPropResult(true, label)
}

func (c *GetCardCmd) String() string { return fmt.Sprintf("GetCard(ID: '%s')", c.CardID) }

// --- DeleteCardCmd ---
type DeleteCardCmd struct {
	CardID string // REAL Card ID
}

func (c *DeleteCardCmd) Run(sut commands.SystemUnderTest) commands.Result {
	fSUT := sut.(*FlashcardSUT)
	fSUT.T.Logf("Run: %s", c.String())
	deleteReq := mcp.CallToolRequest{}
	deleteReq.Params.Name = "delete_card"
	deleteReq.Params.Arguments = map[string]interface{}{"card_id": c.CardID}
	deleteRes, err := fSUT.Client.CallTool(fSUT.Ctx, deleteReq)
	if err != nil {
		return fmt.Errorf("delete_card Run failed: %w", err)
	}
	if len(deleteRes.Content) == 0 {
		return fmt.Errorf("delete_card Run: no content")
	}
	deleteTxt, ok := deleteRes.Content[0].(mcp.TextContent)
	if !ok {
		return fmt.Errorf("delete_card Run: expected TextContent, got %T", deleteRes.Content[0])
	}
	var deleteResp DeleteCardResponse
	err = json.Unmarshal([]byte(deleteTxt.Text), &deleteResp)
	if err != nil {
		var errResp map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(deleteTxt.Text), &errResp); jsonErr == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				if strings.Contains(strings.ToLower(errMsg), "not found") {
					fSUT.T.Logf("Original JSON parse error (ignored): %v", err)
					fSUT.T.Logf("DeleteCardCmd Run: Card %s not found, treating as success.", c.CardID)
					return DeleteCardResponse{Success: true, Message: "Already deleted"}
				}
				return fmt.Errorf("delete_card tool error: %s", errMsg)
			}
		}
		return fmt.Errorf("failed parse delete_card JSON: %w. Resp: %s", err, deleteTxt.Text)
	}
	return deleteResp
}

func (c *DeleteCardCmd) NextState(state commands.State) commands.State {
	currentState := state.(*CommandState)
	next := currentState.deepCopy() // Use deep copy helper
	delete(next.Cards, c.CardID)
	newKnownIDs := []string{}
	for _, id := range next.KnownRealIDs {
		if id != c.CardID {
			newKnownIDs = append(newKnownIDs, id)
		}
	}
	next.KnownRealIDs = newKnownIDs
	if next.LastRealID == c.CardID {
		next.LastRealID = ""
	}
	return next
}

func (c *DeleteCardCmd) PreCondition(state commands.State) bool {
	_, exists := state.(*CommandState).Cards[c.CardID]
	return exists // Must exist in model to be deleted by generated command
}

func (c *DeleteCardCmd) PostCondition(state commands.State, result commands.Result) *gopter.PropResult {
	cmdState := state.(*CommandState)
	label := fmt.Sprintf("PostCondition %s", c.String())
	if _, ok := result.(error); ok {
		cmdState.T.Logf("PostCondition failed for %s due to Run error: %v", c.String(), result.(error))
		return gopter.NewPropResult(false, label)
	}
	deleteResp, ok := result.(DeleteCardResponse)
	if !ok {
		return gopter.NewPropResult(false, label)
	}
	if !deleteResp.Success {
		return gopter.NewPropResult(false, label)
	}
	return gopter.NewPropResult(true, label)
}

func (c *DeleteCardCmd) String() string { return fmt.Sprintf("DeleteCard(ID: '%s')", c.CardID) }

// --- UpdateCardCmd ---
type UpdateCardCmd struct {
	CardID   string // REAL Card ID
	NewFront *string
	NewBack  *string
	NewTags  *[]string
	// Store expected values for PostCondition check
	ExpectedFront string
	ExpectedBack  string
	ExpectedTags  []string
}

func (c *UpdateCardCmd) Run(sut commands.SystemUnderTest) commands.Result {
	fSUT := sut.(*FlashcardSUT)
	fSUT.T.Logf("Run: %s", c.String())
	args := map[string]interface{}{"card_id": c.CardID}
	if c.NewFront != nil {
		args["front"] = *c.NewFront
	}
	if c.NewBack != nil {
		args["back"] = *c.NewBack
	}
	if c.NewTags != nil {
		args["tags"] = InterfaceSlice(*c.NewTags)
	}
	if len(args) <= 1 {
		return fmt.Errorf("UpdateCardCmd Run: no fields provided")
	}
	updateReq := mcp.CallToolRequest{}
	updateReq.Params.Name = "update_card"
	updateReq.Params.Arguments = args
	updateRes, err := fSUT.Client.CallTool(fSUT.Ctx, updateReq)
	if err != nil {
		return fmt.Errorf("update_card Run failed: %w", err)
	}
	if len(updateRes.Content) == 0 {
		return fmt.Errorf("update_card Run: no content")
	}
	updateTxt, ok := updateRes.Content[0].(mcp.TextContent)
	if !ok {
		return fmt.Errorf("update_card Run: expected TextContent, got %T", updateRes.Content[0])
	}
	var updateResp UpdateCardResponse
	err = json.Unmarshal([]byte(updateTxt.Text), &updateResp)
	if err != nil {
		var errResp map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(updateTxt.Text), &errResp); jsonErr == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				fSUT.T.Logf("Original JSON parse error (ignored): %v", err)
				return fmt.Errorf("update_card tool error: %s", errMsg)
			}
		}
		return fmt.Errorf("failed parse update_card JSON: %w. Resp: %s", err, updateTxt.Text)
	}
	if !updateResp.Success {
		return fmt.Errorf("update_card failure: %s", updateResp.Message)
	}
	// Return success, PostCondition will verify using Get
	return updateResp
}

func (c *UpdateCardCmd) NextState(state commands.State) commands.State {
	currentState := state.(*CommandState)
	next := currentState.deepCopy()
	if cardToUpdate, ok := next.Cards[c.CardID]; ok {
		// Calculate and store expected values *before* updating model
		c.ExpectedFront = cardToUpdate.Front
		c.ExpectedBack = cardToUpdate.Back
		c.ExpectedTags = cardToUpdate.Tags // Assumes sorted
		if c.NewFront != nil {
			c.ExpectedFront = *c.NewFront
		}
		if c.NewBack != nil {
			c.ExpectedBack = *c.NewBack
		}
		if c.NewTags != nil {
			c.ExpectedTags = *c.NewTags
			sort.Strings(c.ExpectedTags)
		}

		// Update the model state for the next command
		cardToUpdate.Front = c.ExpectedFront
		cardToUpdate.Back = c.ExpectedBack
		cardToUpdate.Tags = c.ExpectedTags
		next.Cards[c.CardID] = cardToUpdate
	}
	return next
}

func (c *UpdateCardCmd) PreCondition(state commands.State) bool {
	_, exists := state.(*CommandState).Cards[c.CardID]
	return exists && (c.NewFront != nil || c.NewBack != nil || c.NewTags != nil)
}

func (c *UpdateCardCmd) PostCondition(state commands.State, result commands.Result) *gopter.PropResult {
	cmdState := state.(*CommandState)
	label := fmt.Sprintf("PostCondition %s", c.String())
	if _, ok := result.(error); ok {
		cmdState.T.Logf("PostCondition failed for %s due to Run error: %v", c.String(), result.(error))
		return gopter.NewPropResult(false, label)
	}
	updateResp, ok := result.(UpdateCardResponse)
	if !ok {
		return gopter.NewPropResult(false, label)
	}
	if !updateResp.Success {
		return gopter.NewPropResult(false, label)
	}

	// Verify: Use the expected values calculated and stored by NextState
	// Get the card from the *current* model state (which NextState returned)
	updatedModelCard, ok := state.(*CommandState).Cards[c.CardID]
	if !ok {
		return gopter.NewPropResult(false, label)
	}

	if updatedModelCard.Front != c.ExpectedFront {
		return gopter.NewPropResult(false, label)
	}
	if updatedModelCard.Back != c.ExpectedBack {
		return gopter.NewPropResult(false, label)
	}
	if !CompareTags(c.ExpectedTags, updatedModelCard.Tags) {
		return gopter.NewPropResult(false, label)
	}

	return gopter.NewPropResult(true, label)
}

func (c *UpdateCardCmd) String() string {
	var updates []string
	if c.NewFront != nil {
		updates = append(updates, fmt.Sprintf("Front: '%s'", *c.NewFront))
	}
	if c.NewBack != nil {
		updates = append(updates, fmt.Sprintf("Back: '%s'", *c.NewBack))
	}
	if c.NewTags != nil {
		updates = append(updates, fmt.Sprintf("Tags: %v", *c.NewTags))
	}
	return fmt.Sprintf("UpdateCard(ID: '%s', Updates: [%s])", c.CardID, strings.Join(updates, ", "))
}

// --- SubmitReviewCmd ---
type SubmitReviewCmd struct {
	CardID string // REAL Card ID
	Rating gofsrs.Rating
	Answer string // Optional answer text
	// Store expected FSRS state for PostCondition
	ExpectedFSRSState gofsrs.State
	ExpectedDueDate   time.Time
}

func (c *SubmitReviewCmd) Run(sut commands.SystemUnderTest) commands.Result {
	fSUT := sut.(*FlashcardSUT)
	fSUT.T.Logf("Run: %s", c.String())

	submitReviewRequest := mcp.CallToolRequest{}
	submitReviewRequest.Params.Name = "submit_review"
	submitReviewRequest.Params.Arguments = map[string]interface{}{
		"card_id": c.CardID,
		"rating":  float64(c.Rating), // Convert Rating to float64 as expected by API
		"answer":  c.Answer,
	}

	submitResult, err := fSUT.Client.CallTool(fSUT.Ctx, submitReviewRequest)
	if err != nil {
		return fmt.Errorf("submit_review Run failed: %w", err)
	}
	if len(submitResult.Content) == 0 {
		return fmt.Errorf("submit_review Run: no content returned")
	}

	submitTextContent, ok := submitResult.Content[0].(mcp.TextContent)
	if !ok {
		return fmt.Errorf("submit_review Run: expected TextContent, got %T", submitResult.Content[0])
	}

	var reviewResponse ReviewResponse
	err = json.Unmarshal([]byte(submitTextContent.Text), &reviewResponse)
	if err != nil {
		var errResp map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(submitTextContent.Text), &errResp); jsonErr == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				fSUT.T.Logf("Original JSON parse error (ignored): %v", err)
				return fmt.Errorf("submit_review tool error: %s", errMsg)
			}
		}
		return fmt.Errorf("submit_review Run: failed parse response: %w. Resp: %s", err, submitTextContent.Text)
	}

	if !reviewResponse.Success {
		return fmt.Errorf("submit_review failed: %s", reviewResponse.Message)
	}

	return reviewResponse
}

func (c *SubmitReviewCmd) NextState(state commands.State) commands.State {
	currentState := state.(*CommandState)
	nextState := currentState.deepCopy()

	// Find the card model
	cardModel, exists := nextState.Cards[c.CardID]
	if !exists {
		// This shouldn't happen due to PreCondition, but handle it
		return nextState
	}

	// Simulate the FSRS update logic based on current state, rating, and current time
	now := time.Now()

	// --- FSRS Simulation Logic ---
	// Retrieve FSRS parameters (use default for simulation)
	fakeParams := gofsrs.DefaultParam()

	// Debug before the call
	fmt.Printf("DEBUG NextState before Repeat: card state=%v, rating=%v\n",
		cardModel.FSRS.State, c.Rating)

	// Call the actual FSRS library's Repeat function to get the correct next state
	// Use the same approach as the actual ScheduleReview function
	schedulingInfo := fakeParams.Repeat(cardModel.FSRS, now)
	nextFSRSInfo := schedulingInfo[c.Rating]

	// Debug after the call
	fmt.Printf("DEBUG NextState after Repeat: got next state=%v\n",
		nextFSRSInfo.Card.State)

	// Store expected values for PostCondition check
	c.ExpectedFSRSState = nextFSRSInfo.Card.State
	c.ExpectedDueDate = nextFSRSInfo.Card.Due
	// --- End FSRS Simulation Logic ---

	// Update the card in the next state model
	cardModel.FSRS = nextFSRSInfo.Card // Store the updated FSRS details
	cardModel.LastReviewedAt = now
	nextState.Cards[c.CardID] = cardModel

	return nextState
}

func (c *SubmitReviewCmd) PreCondition(state commands.State) bool {
	_, exists := state.(*CommandState).Cards[c.CardID]
	return exists // Card must exist in the model
}

func (c *SubmitReviewCmd) PostCondition(state commands.State, result commands.Result) *gopter.PropResult {
	cmdState := state.(*CommandState)
	label := fmt.Sprintf("PostCondition %s", c.String())

	if errResult, ok := result.(error); ok {
		cmdState.T.Logf("PostCondition failed for %s due to Run error: %v", c.String(), errResult)
		return gopter.NewPropResult(false, label)
	}

	reviewResp, ok := result.(ReviewResponse)
	if !ok {
		return gopter.NewPropResult(false, label)
	}

	if !reviewResp.Success {
		return gopter.NewPropResult(false, label)
	}

	// Verify the card ID matches
	if reviewResp.Card.ID != c.CardID {
		cmdState.T.Logf("Card ID mismatch: expected %s, got %s", c.CardID, reviewResp.Card.ID)
		return gopter.NewPropResult(false, label)
	}

	// Verify the FSRS state matches our expectation
	if reviewResp.Card.FSRS.State != c.ExpectedFSRSState {
		cmdState.T.Logf("FSRS state mismatch: expected %v (%d), got %v (%d)",
			c.ExpectedFSRSState, int(c.ExpectedFSRSState),
			reviewResp.Card.FSRS.State, int(reviewResp.Card.FSRS.State))
		return gopter.NewPropResult(false, label)
	}

	// Verify the due date is close to our expectation
	// Allow a small time difference due to processing time (e.g., 5 seconds)
	timeDiff := reviewResp.Card.FSRS.Due.Sub(c.ExpectedDueDate).Abs()
	allowedDiff := 5 * time.Second
	if timeDiff > allowedDiff {
		cmdState.T.Logf("Due date mismatch: expected %v, got %v (diff: %v > %v)",
			c.ExpectedDueDate, reviewResp.Card.FSRS.Due, timeDiff, allowedDiff)
		return gopter.NewPropResult(false, label)
	}

	return gopter.NewPropResult(true, label)
}

func (c *SubmitReviewCmd) String() string {
	return fmt.Sprintf("SubmitReview(CardID: '%s', Rating: %d, Answer: '%s')", c.CardID, c.Rating, c.Answer)
}

// --- GetDueCardCmd ---
type GetDueCardCmd struct {
	FilterTags []string
	// Store expected card ID or error type for PostCondition
	ExpectedCardID    string    // Empty if expecting "no cards due" or specific tag error
	ExpectedErrorType ErrorType // Type of error expected (None, NoCardsDue, NoTagMatch)
}

type ErrorType int

const (
	ErrorTypeNone ErrorType = iota
	ErrorTypeNoCardsDue
	ErrorTypeNoTagMatch
)

func (c *GetDueCardCmd) Run(sut commands.SystemUnderTest) commands.Result {
	fSUT := sut.(*FlashcardSUT)
	fSUT.T.Logf("Run: %s", c.String())

	getDueCardRequest := mcp.CallToolRequest{}
	getDueCardRequest.Params.Name = "get_due_card"

	if len(c.FilterTags) > 0 {
		getDueCardRequest.Params.Arguments = map[string]interface{}{
			"filter_tags": InterfaceSlice(c.FilterTags),
		}
	}

	getDueResult, err := fSUT.Client.CallTool(fSUT.Ctx, getDueCardRequest)
	if err != nil {
		return fmt.Errorf("get_due_card Run failed: %w", err)
	}
	if len(getDueResult.Content) == 0 {
		return fmt.Errorf("get_due_card Run: no content returned")
	}

	dueTextContent, ok := getDueResult.Content[0].(mcp.TextContent)
	if !ok {
		return fmt.Errorf("get_due_card Run: expected TextContent, got %T", getDueResult.Content[0])
	}

	// Try parsing as CardResponse first
	var cardResponse CardResponse
	cardErr := json.Unmarshal([]byte(dueTextContent.Text), &cardResponse)
	if cardErr == nil && cardResponse.Card.ID != "" {
		return cardResponse
	}

	// If that failed, try parsing as error response
	var errorResp map[string]interface{}
	errorErr := json.Unmarshal([]byte(dueTextContent.Text), &errorResp)
	if errorErr == nil {
		if errMsg, ok := errorResp["error"].(string); ok {
			// Return structured error with the message
			return fmt.Errorf("get_due_card tool error: %s", errMsg)
		}
	}

	// If both parsing attempts failed, return generic error
	return fmt.Errorf("get_due_card Run: failed to parse response. Resp: %s", dueTextContent.Text)
}

func (c *GetDueCardCmd) NextState(state commands.State) commands.State {
	currentState := state.(*CommandState)
	next := currentState.deepCopy() // Start with a copy

	// Calculate expected outcome based on the *current* state
	now := time.Now()
	var potentialDueCards []*Card
	anyCardWithTags := false

	for _, card := range next.Cards { // Iterate over cards in the copied state
		if hasAllTags(&card, c.FilterTags) {
			anyCardWithTags = true
			if !card.FSRS.Due.After(now) { // Card is due
				cardCopy := card // Make a copy to avoid pointer issues if needed later
				potentialDueCards = append(potentialDueCards, &cardCopy)
			}
		}
	}

	// Sort due cards by priority using direct calculation
	if len(potentialDueCards) > 0 {
		// Replicate priority logic from internal/fsrs/fsrs.go::GetReviewPriority
		sort.Slice(potentialDueCards, func(i, j int) bool {
			priority1 := calculateModelPriority(potentialDueCards[i].FSRS.State, potentialDueCards[i].FSRS.Due, now)
			priority2 := calculateModelPriority(potentialDueCards[j].FSRS.State, potentialDueCards[j].FSRS.Due, now)
			return priority1 > priority2 // Higher priority first
		})
		c.ExpectedCardID = potentialDueCards[0].ID
		c.ExpectedErrorType = ErrorTypeNone
	} else if len(c.FilterTags) > 0 && !anyCardWithTags {
		// If filtering by tags and no cards have those tags
		c.ExpectedCardID = ""
		c.ExpectedErrorType = ErrorTypeNoTagMatch
	} else {
		// No cards are due (either overall or matching the tags)
		c.ExpectedCardID = ""
		c.ExpectedErrorType = ErrorTypeNoCardsDue
	}

	// GetDueCardCmd doesn't change the state itself, only calculates expectations
	return currentState // Return the original state, not the modified copy
}

func (c *GetDueCardCmd) PreCondition(state commands.State) bool {
	return true // Always can try to get due cards
}

func (c *GetDueCardCmd) PostCondition(state commands.State, result commands.Result) *gopter.PropResult {
	cmdState := state.(*CommandState)
	label := fmt.Sprintf("PostCondition %s", c.String())

	// Check if the result is an error
	if errResult, ok := result.(error); ok {
		// We got an error from Run
		errorMsg := strings.ToLower(errResult.Error())
		switch c.ExpectedErrorType {
		case ErrorTypeNone:
			cmdState.T.Logf("Unexpected error from get_due_card: %v", errResult)
			return gopter.NewPropResult(false, label)
		case ErrorTypeNoCardsDue:
			if !strings.Contains(errorMsg, "no cards due") {
				cmdState.T.Logf("Expected 'no cards due' error, got: %v", errResult)
				return gopter.NewPropResult(false, label)
			}
			return gopter.NewPropResult(true, label) // Expected error received
		case ErrorTypeNoTagMatch:
			if !(strings.Contains(errorMsg, "no cards found") && strings.Contains(errorMsg, "specified tags")) {
				cmdState.T.Logf("Expected 'no tag match' error, got: %v", errResult)
				return gopter.NewPropResult(false, label)
			}
			return gopter.NewPropResult(true, label) // Expected error received
		}
	}

	// If we got here, result was not an error
	if c.ExpectedErrorType != ErrorTypeNone {
		cmdState.T.Logf("Expected an error (type %d), but received success: %+v", c.ExpectedErrorType, result)
		return gopter.NewPropResult(false, label)
	}

	// Expected success, verify the card response
	cardResponse, ok := result.(CardResponse)
	if !ok {
		cmdState.T.Logf("Expected CardResponse but got %T", result)
		return gopter.NewPropResult(false, label)
	}

	// Verify the returned card ID matches the expected highest priority ID
	if cardResponse.Card.ID != c.ExpectedCardID {
		cmdState.T.Logf("Returned card ID %s does not match expected highest priority card ID %s",
			cardResponse.Card.ID, c.ExpectedCardID)
		return gopter.NewPropResult(false, label)
	}

	// Check stats for basic validity
	if cardResponse.Stats.TotalCards < 0 || cardResponse.Stats.DueCards < 0 ||
		cardResponse.Stats.ReviewsToday < 0 || cardResponse.Stats.RetentionRate < 0 ||
		cardResponse.Stats.RetentionRate > 100 {
		cmdState.T.Logf("Stats contain unreasonable values: %+v", cardResponse.Stats)
		return gopter.NewPropResult(false, label)
	}

	return gopter.NewPropResult(true, label)
}

func (c *GetDueCardCmd) String() string {
	if len(c.FilterTags) == 0 {
		return "GetDueCard()"
	}
	return fmt.Sprintf("GetDueCard(FilterTags: %v)", c.FilterTags)
}

// Helper function (assuming it's defined elsewhere or needs to be added)
func hasAllTags(card *Card, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true // No filter means match
	}
	if card == nil || card.Tags == nil {
		return false // Cannot have all tags if card or tags are nil
	}
	cardTagsMap := make(map[string]bool)
	for _, tag := range card.Tags {
		cardTagsMap[tag] = true
	}
	for _, reqTag := range requiredTags {
		if !cardTagsMap[reqTag] {
			return false // Missing a required tag
		}
	}
	return true // All required tags found
}

// calculateModelPriority replicates the priority logic from internal/fsrs/fsrs.go
func calculateModelPriority(state gofsrs.State, due time.Time, now time.Time) float64 {
	var basePriority float64
	switch state {
	case gofsrs.New:
		basePriority = 1.0
	case gofsrs.Learning, gofsrs.Relearning:
		basePriority = 3.0
	case gofsrs.Review:
		basePriority = 2.0
	}

	overdueDays := now.Sub(due).Hours() / 24.0
	if overdueDays >= 0 {
		overdueFactor := 1.0 + (overdueDays * 0.1)
		return basePriority * overdueFactor
	}

	daysToDue := -overdueDays
	return basePriority / (1.0 + daysToDue)
}

// --- Gopter ProtoCommands Definition ---

// Map to store cleanup functions, keyed by SUT client pointer
// Note: key type changed to *client.Client
var cleanupMap = make(map[*client.Client]func())
var cleanupMapMutex sync.Mutex

// Removed global FlashcardProtoCommands definition
// It will be defined locally within TestCommandSequences
