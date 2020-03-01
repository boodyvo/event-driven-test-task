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


const maxIteration = 100

var (
	printAction events.Action = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		fmt.Printf("arguments passed: %#v\n", input)
		return input, nil
	}

	addRandomValueAction events.Action = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		input["value"] = rand.Intn(1000)

		return input, nil
	}

	failRandomAction = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		// always fail if flag random is set
		if input["fail"] != nil {
			return nil, fmt.Errorf("permanent fail")
		}
		// randomly fail if flag random is set
		if input["random"] != nil {
			val := rand.Intn(100)

			if val < 50 {
				return nil, fmt.Errorf("random fail")
			}
		}

		return input, nil
	}

	actions = []events.Action{
		printAction,
		addRandomValueAction,
		failRandomAction,
	}
)

func TestEvents(t *testing.T) {
	store := statemng.NewStore()
	eventManager := events.NewManager(actions, store)
	eventOk := map[string]interface{}{
		"0": map[string]interface{}{
			"name": "first",
		},
		"1": map[string]interface{}{
			"name": "second",
		},
		"2": map[string]interface{}{
			"name": "third",
		},
		"other": "some other info",
	}
	_, err := eventManager.ExecuteEvent(context.Background(), eventOk)
	require.Nil(t, err)

	eventAlwaysFail := map[string]interface{}{
		"0": map[string]interface{}{
			"name": "first",
		},
		"1": map[string]interface{}{
			"name": "second",
		},
		"2": map[string]interface{}{
			"always": true,
		},
		"other": "some other info",
	}
	_, err = eventManager.ExecuteEvent(context.Background(), eventAlwaysFail)
	require.Nil(t, err)

	eventRandomFail := map[string]interface{}{
		"0": map[string]interface{}{
			"name": "first",
		},
		"1": map[string]interface{}{
			"name": "second",
		},
		"2": map[string]interface{}{
			"random": true,
		},
		"other": "some other info",
	}
	id, err := eventManager.ExecuteEvent(context.Background(), eventRandomFail)
	// checking that error occurs
	for err == nil {
		id, err = eventManager.ExecuteEvent(context.Background(), eventRandomFail)
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