package events

import (
	"context"
	"fmt"
	sm "github.com/boodyvo/snapshot-backup/statemng"
	"github.com/google/uuid"
	"log"
)

type Action = func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

type Manager interface {
	// string for return ID of event is just for fast testing, could be tested in another way
	ExecuteEvent(ctx context.Context, event map[string]interface{}) (string, error)
	RestoreEvent(ctx context.Context, id string) (string, error)
}

type ManagerImp struct {
	events  map[string]interface{}
	// list of actions that should be executed
	actions []Action
	store  sm.Store
}

func NewManager(actions []Action, store sm.Store) Manager {
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
func (m *ManagerImp) ExecuteEvent(ctx context.Context, event map[string]interface{}) (string, error) {
	log.Println("event execution: ", event)
	// save initial request

	stateId := uuid.New().String()
	state := sm.NewState(stateId, event)
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
	for index, action := range m.actions {
		actionState := state.GetActionState(index)
		if actionState == nil {
			// ignore nil inputs for simplicity
			inputAction, _ := state.Event[fmt.Sprintf("%d", index)].(map[string]interface{})
			actionState = &sm.ActionState{
				Input:  inputAction,
				Output: make(map[string]interface{}),
				Status: sm.Started,
			}
		} else if actionState.Status == sm.Succeed {
			// continue if we already succeeded the action
			continue
		}
		log.Printf("event %s, step %d\n", state.Id, index)

		// if we failed the task or didn't finish/execute it we need to execute it
		state.UpdateAction(index, actionState)
		m.store.SaveState(state)
		output, err := action(ctx, actionState.Input)
		actionState.Output = output
		if err != nil {
			actionState.Status = sm.Failed
			state.UpdateAction(index, actionState)
			m.store.SaveState(state)
			log.Println("an error during action execution:", err)

			return state.Id, fmt.Errorf("an error during action execution: %v", err)
		}
		actionState.Status = sm.Succeed
		state.UpdateAction(index, actionState)
		m.store.SaveState(state)
	}
	// all actions completed
	state.Completed = true
	m.store.SaveState(state)

	return state.Id, nil
}