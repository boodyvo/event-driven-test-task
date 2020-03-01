package statemng

import (
	"fmt"
	"sync"
	"time"
)

const ActionNumber = 3

type ActionStatus int

const (
	NotExecuted ActionStatus = 0
	Started     ActionStatus = 1
	Succeed     ActionStatus = 2
	Failed      ActionStatus = 3
)

type ActionState struct {
	Input     map[string]interface{}
	Output    map[string]interface{}
	Status    ActionStatus
}

type State struct {
	Id           string
	Timestamp    time.Time
	Event        map[string]interface{}
	actionStates []*ActionState
	// already completed actions, timestamp should be saved here
	Completed    bool
}

func (s *State) UpdateAction(id int, action *ActionState) {
	if id >= ActionNumber {
		return
	}
	s.actionStates[id] = action
}

func (s *State) GetActionState(id int) (*ActionState) {
	return s.actionStates[id]
}

func NewState(id string, event map[string]interface{}) *State {
	return &State{
		Id:           id,
		Timestamp:    time.Now(),
		Event:        event,
		actionStates: make([]*ActionState, ActionNumber),
		Completed:    false,
	}
}

type Store interface {
	SaveState(state *State)
	RestoreState(id string) (*State, error)
}

type StoreImp struct {
	states map[string]*State

	// we don't have parallel execution but it is better to have
	// if we will have parallel actions
	sync.Mutex
}

func NewStore() Store {
	return &StoreImp{
		states: make(map[string]*State),
	}
}

func (s *StoreImp) SaveState(state *State) {
	s.Lock()
	defer s.Unlock()

	s.states[state.Id] = state
}

func (s *StoreImp) RestoreState(id string) (*State, error) {
	s.Lock()
	defer s.Unlock()

	state, ok := s.states[id]
	if !ok {
		return nil, fmt.Errorf("event not found: %s", id)
	}

	return state, nil
}