package main

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/boodyvo/snapshot-backup/events"
	"github.com/boodyvo/snapshot-backup/statemng"
)

const (
	printActionId = "0"
	failActionId = "1"
	emptyActionId = "2"
	addRandomValueActionId = "3"
	failRandomActionId = "4"
)

var (
	printAction events.Action = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		fmt.Printf("arguments passed: %#v\n", input)
		return input, nil
	}

	failAction events.Action = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		fmt.Printf("will fail the action")
		return input, fmt.Errorf("failed")
	}

	emptyAction events.Action = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return input, nil
	}

	addRandomValueAction events.Action = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		input["value"] = rand.Intn(1000)
		return input, nil
	}

	failRandomAction = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		val := rand.Intn(100)
		if val < 50 {
			return nil, fmt.Errorf("random fail")
		}

		return input, nil
	}

	actions = map[string]events.Action{
		printActionId: printAction,
		failActionId: failAction,
		emptyActionId: emptyAction,
		addRandomValueActionId: addRandomValueAction,
		failRandomActionId: failRandomAction,
	}
)

const maxIteration = 100

func TestEvents(t *testing.T) {
	testInput := map[string]interface{}{
		"name": "Vasya",
		"action": "send_email",
	}
	store := statemng.NewStore()
	eventManager := events.NewManager(actions, store)
	eventOk := []string{printActionId, emptyActionId, printActionId}
	_, err := eventManager.ExecuteEvent(context.Background(), testInput, eventOk)
	require.Nil(t, err)

	eventFailAlways := []string{emptyActionId, failActionId}
	idFail, err := eventManager.ExecuteEvent(context.Background(), testInput, eventFailAlways)
	for i := 0; i < 5; i++ {
		_, err = eventManager.RestoreEvent(context.Background(), idFail)
		if err == nil {
			break
		}
	}


	eventRandomFail := []string{failRandomActionId, failRandomActionId}
	id, err := eventManager.ExecuteEvent(context.Background(), testInput, eventRandomFail)
	// checking that error occurs
	for err == nil {
		id, err = eventManager.ExecuteEvent(context.Background(), testInput, eventRandomFail)
	}

	i := 0
	// probability of failing for maxIteration time is less than 2^-maxIteration * (1 + (1-2^-maxIteration))
	// so it is almost impossible that it will not recover for maxIteration iteration
	for i = 0; i < maxIteration; i++ {
		_, err = eventManager.RestoreEvent(context.Background(), id)
		if err == nil {
			break
		}
	}

	if i == maxIteration {
		require.FailNow(
			t,
			fmt.Sprintf("random function didn't finished successfully for %d restores", maxIteration),
		)
	}
}