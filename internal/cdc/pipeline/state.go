// Package pipeline provides CDC pipeline orchestration.
package pipeline

import (
	"fmt"
	"sync"
)

// State represents the pipeline state.
type State int

const (
	// StateStarting indicates the pipeline is starting up.
	StateStarting State = iota
	// StateRunning indicates the pipeline is actively processing events.
	StateRunning
	// StatePaused indicates the pipeline is paused (e.g., due to backpressure).
	StatePaused
	// StateStopping indicates the pipeline is shutting down gracefully.
	StateStopping
	// StateStopped indicates the pipeline has stopped.
	StateStopped
	// StateFailed indicates the pipeline has encountered a fatal error.
	StateFailed
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// validTransitions defines allowed state transitions.
var validTransitions = map[State][]State{
	StateStarting: {StateRunning, StateFailed, StateStopping},
	StateRunning:  {StatePaused, StateStopping, StateFailed},
	StatePaused:   {StateRunning, StateStopping, StateFailed},
	StateStopping: {StateStopped, StateFailed},
	StateStopped:  {StateStarting},
	StateFailed:   {StateStarting, StateStopped},
}

// StateMachine manages pipeline state transitions.
type StateMachine struct {
	mu        sync.RWMutex
	state     State
	listeners []StateChangeListener
}

// StateChangeListener is called when state changes.
type StateChangeListener func(from, to State)

// NewStateMachine creates a new state machine starting in StateStarting.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		state: StateStarting,
	}
}

// State returns the current state.
func (sm *StateMachine) State() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// Transition attempts to transition to the target state.
// Returns an error if the transition is not valid.
func (sm *StateMachine) Transition(target State) error {
	sm.mu.Lock()

	if !sm.canTransition(target) {
		sm.mu.Unlock()
		return fmt.Errorf("invalid state transition from %s to %s", sm.state, target)
	}

	from := sm.state
	sm.state = target

	// Notify listeners (copy to avoid holding lock)
	listeners := make([]StateChangeListener, len(sm.listeners))
	copy(listeners, sm.listeners)

	// Release lock before calling listeners
	sm.mu.Unlock()

	for _, listener := range listeners {
		listener(from, target)
	}

	return nil
}

// canTransition checks if a transition to target is valid.
// Must be called with lock held.
func (sm *StateMachine) canTransition(target State) bool {
	allowed, ok := validTransitions[sm.state]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}

// AddListener adds a state change listener.
func (sm *StateMachine) AddListener(listener StateChangeListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// IsRunning returns true if the pipeline is in a running state.
func (sm *StateMachine) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state == StateRunning
}

// IsPaused returns true if the pipeline is paused.
func (sm *StateMachine) IsPaused() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state == StatePaused
}

// CanProcess returns true if the pipeline can process events.
func (sm *StateMachine) CanProcess() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state == StateRunning
}

// IsTerminal returns true if the pipeline is in a terminal state.
func (sm *StateMachine) IsTerminal() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state == StateStopped || sm.state == StateFailed
}
