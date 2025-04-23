package propertytest

import (
	"reflect"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/commands"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// --- Custom Command Generators ---

// generateCreateCardCmd creates a generator that produces CreateCardCmd instances
// without problematic type conversions
func generateCreateCardCmd() gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		// Generate front text (limit to 100 chars)
		frontGen := GenNonEmptyString(100)
		frontResult := frontGen(genParams)
		if frontResult.Shrinker == nil {
			return gopter.NewEmptyResult(reflect.TypeOf(""))
		}
		front := frontResult.Result.(string)

		// Generate back text (limit to 200 chars)
		backGen := GenNonEmptyString(200)
		backResult := backGen(genParams)
		if backResult.Shrinker == nil {
			return gopter.NewEmptyResult(reflect.TypeOf(""))
		}
		back := backResult.Result.(string)

		// Generate tags
		tagsGen := GenTags(5, 20)
		tagsResult := tagsGen(genParams)
		if tagsResult.Shrinker == nil {
			return gopter.NewEmptyResult(reflect.TypeOf([]string{}))
		}
		tags := tagsResult.Result.([]string)

		// Create the command
		cmd := &CreateCardCmd{
			Front: front,
			Back:  back,
			Tags:  tags,
		}

		// Return as a properly typed Command
		return gopter.NewGenResult(cmd, gopter.NoShrinker)
	}
}

// generateGetDueCardCmd creates a generator that produces GetDueCardCmd instances
func generateGetDueCardCmd() gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		// Sometimes use empty tags, sometimes generate tags
		var tags []string
		if genParams.Rng.Intn(2) == 0 { // 50% chance of no filter tags
			tags = []string{}
		} else {
			// Generate tags
			tagsGen := GenTags(5, 20)
			tagsResult := tagsGen(genParams)
			if tagsResult.Shrinker == nil {
				return gopter.NewEmptyResult(reflect.TypeOf([]string{}))
			}
			tags = tagsResult.Result.([]string)
		}

		// Create the command
		cmd := &GetDueCardCmd{
			FilterTags: tags,
		}

		// Return as a properly typed Command
		return gopter.NewGenResult(cmd, gopter.NoShrinker)
	}
}

// generateIdBasedCmd creates a generator that produces command instances
// that require an existing card ID
func generateIdBasedCmd(cmdState *CommandState) gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		// Ensure we have at least one ID
		if len(cmdState.KnownRealIDs) == 0 {
			return gopter.NewEmptyResult(reflect.TypeOf(""))
		}

		// Pick a random card ID
		idx := genParams.Rng.Intn(len(cmdState.KnownRealIDs))
		cardID := cmdState.KnownRealIDs[idx]

		// Choose one of the ID-based commands with weighted distribution
		// to ensure we test all command types
		// GetCardCmd: 20%, DeleteCardCmd: 20%, SubmitReviewCmd: 60%
		randVal := genParams.Rng.Intn(10)
		var cmd commands.Command

		switch {
		case randVal < 2: // 20% chance for GetCardCmd
			// Generate GetCardCmd
			cmd = &GetCardCmd{CardID: cardID}
		case randVal < 4: // 20% chance for DeleteCardCmd
			// Generate DeleteCardCmd
			cmd = &DeleteCardCmd{CardID: cardID}
		default: // 60% chance for SubmitReviewCmd
			// Generate SubmitReviewCmd
			// Choose a random rating
			ratings := []gofsrs.Rating{gofsrs.Again, gofsrs.Hard, gofsrs.Good, gofsrs.Easy}
			rating := ratings[genParams.Rng.Intn(len(ratings))]

			// Generate answer text
			answerGen := GenNonEmptyString(100)
			answerResult := answerGen(genParams)
			if answerResult.Shrinker == nil {
				return gopter.NewEmptyResult(reflect.TypeOf(""))
			}
			answer := answerResult.Result.(string)

			cmd = &SubmitReviewCmd{
				CardID: cardID,
				Rating: rating,
				Answer: answer,
			}
		}

		return gopter.NewGenResult(cmd, gopter.NoShrinker)
	}
}

// generateUpdateCardCmd creates a generator that produces UpdateCardCmd instances
func generateUpdateCardCmd(cmdState *CommandState) gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		// Ensure we have at least one ID
		if len(cmdState.KnownRealIDs) == 0 {
			return gopter.NewEmptyResult(reflect.TypeOf(""))
		}

		// Pick a random card ID
		idx := genParams.Rng.Intn(len(cmdState.KnownRealIDs))
		cardID := cmdState.KnownRealIDs[idx]

		// Decide which fields to update (at least one must be updated)
		updateFront := genParams.Rng.Intn(2) == 1
		updateBack := genParams.Rng.Intn(2) == 1
		updateTags := genParams.Rng.Intn(2) == 1

		// Ensure at least one field is updated
		if !updateFront && !updateBack && !updateTags {
			// Force at least one update
			switch genParams.Rng.Intn(3) {
			case 0:
				updateFront = true
			case 1:
				updateBack = true
			case 2:
				updateTags = true
			}
		}

		// Generate the update fields
		var newFront, newBack *string
		var newTags *[]string

		if updateFront {
			frontGen := GenNonEmptyString(100)
			frontResult := frontGen(genParams)
			if frontResult.Shrinker == nil {
				return gopter.NewEmptyResult(reflect.TypeOf(""))
			}
			front := frontResult.Result.(string)
			newFront = &front
		}

		if updateBack {
			backGen := GenNonEmptyString(200)
			backResult := backGen(genParams)
			if backResult.Shrinker == nil {
				return gopter.NewEmptyResult(reflect.TypeOf(""))
			}
			back := backResult.Result.(string)
			newBack = &back
		}

		if updateTags {
			tagsGen := GenTags(5, 20)
			tagsResult := tagsGen(genParams)
			if tagsResult.Shrinker == nil {
				return gopter.NewEmptyResult(reflect.TypeOf([]string{}))
			}
			tags := tagsResult.Result.([]string)
			newTags = &tags
		}

		// Create the command
		cmd := &UpdateCardCmd{
			CardID:   cardID,
			NewFront: newFront,
			NewBack:  newBack,
			NewTags:  newTags,
		}

		return gopter.NewGenResult(cmd, gopter.NoShrinker)
	}
}

// generateSubmitReviewCmd creates a generator that produces only SubmitReviewCmd instances
func generateSubmitReviewCmd(cmdState *CommandState) gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		// Ensure we have at least one ID
		if len(cmdState.KnownRealIDs) == 0 {
			return gopter.NewEmptyResult(reflect.TypeOf(""))
		}

		// Pick a random card ID
		idx := genParams.Rng.Intn(len(cmdState.KnownRealIDs))
		cardID := cmdState.KnownRealIDs[idx]

		// Choose a random rating
		ratings := []gofsrs.Rating{gofsrs.Again, gofsrs.Hard, gofsrs.Good, gofsrs.Easy}
		rating := ratings[genParams.Rng.Intn(len(ratings))]

		// Generate answer text
		answerGen := GenNonEmptyString(100)
		answerResult := answerGen(genParams)
		if answerResult.Shrinker == nil {
			return gopter.NewEmptyResult(reflect.TypeOf(""))
		}
		answer := answerResult.Result.(string)

		cmd := &SubmitReviewCmd{
			CardID: cardID,
			Rating: rating,
			Answer: answer,
		}

		return gopter.NewGenResult(cmd, gopter.NoShrinker)
	}
}
