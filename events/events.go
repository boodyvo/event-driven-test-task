package events

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"log"

	sm "github.com/boodyvo/snapshot-backup/statemng"
)

type Action = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

type Manager interface {
	// string for return ID of event is just for fast testing, could be tested in another way
	ExecuteEvent(ctx context.Context, input map[string]interface{}, actions []string) (string, error)
	RestoreEvent(ctx context.Context, id string) (string, error)
}

type ManagerImp struct {
	events  map[string]interface{}
	// list of actions that could be executed
	actions map[string]Action
	store  sm.Store
}

func NewManager(actions map[string]Action, store sm.Store) Manager {
	return &ManagerImp{
		events:  make(map[string]interface{}),
		actions: actions,
		store:   store,
	}
}

// 1. Could be atomic with consensus if we have atomic actions
// (sending an email is not such example)
// that fully managed with our systems so saving to DB of state
// is atomic with executing an action, but it is more complicated.
// 2. Assume actions not causing panic, only returning errors during fail or
// we need recover() wrapper in this case
func (m *ManagerImp) ExecuteEvent(ctx context.Context, input map[string]interface{}, actions []string) (string, error) {
	log.Println("event execution: ", actions)
	// save initial request

	stateId := uuid.New().String()
	state := sm.NewState(stateId, input, actions)
	m.store.SaveState(state)

	return m.executeActions(ctx, state)
}

func (m *ManagerImp) RestoreEvent(ctx context.Context, id string) (string, error) {
	state, err := m.store.RestoreState(id)
	if err != nil {
		log.Println("unknown event", id)
		return "", fmt.Errorf("unknown event")
	}

	return m.executeActions(ctx, state)
}

func (m *ManagerImp) executeActions(ctx context.Context, state *sm.State) (string, error) {
	// In map we will call actions in random order but assume that it is ok
	for index, actionId := range state.Actions {
		log.Printf("step %d, action %s\n", index, actionId)
		action, ok := m.actions[actionId]
		if !ok {
			log.Println("an error ")
		}
		output, err := action(ctx, state.Input)
		if err != nil {
			log.Println("an error during action execution:", err)
			// close event
			return state.Id, fmt.Errorf("an error during action execution: %v", err)
		}
		state.Output = output
		state.Completed = index
		m.store.SaveState(state)
	}

	return state.Id, nil
}