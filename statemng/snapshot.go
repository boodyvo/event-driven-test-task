package statemng

import (
	"fmt"
	"sync"
)

type State struct {
	Id        string
	Input     map[string]interface{}
	Output    map[string]interface{}
	// actions that needs to be executed
	Actions   []string
	// already completed actions, timestamp should be saved here
	Completed int
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

func NewState(id string, input map[string]interface{}, actions []string) *State {
	return &State{
		Id:        id,
		Input:     input,
		Output:    make(map[string]interface{}),
		Actions:   actions,
		Completed: -1,
	}
}