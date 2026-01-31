package pipeline

import (
	"testing"
)

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StatePaused, "paused"},
		{StateStopping, "stopping"},
		{StateStopped, "stopped"},
		{StateFailed, "failed"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("State.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStateMachine_InitialState(t *testing.T) {
	sm := NewStateMachine()
	if sm.State() != StateStarting {
		t.Errorf("expected initial state to be StateStarting, got %v", sm.State())
	}
}

func TestStateMachine_ValidTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    State
		to      State
		wantErr bool
	}{
		{"starting to running", StateStarting, StateRunning, false},
		{"starting to failed", StateStarting, StateFailed, false},
		{"starting to stopping", StateStarting, StateStopping, false},
		{"running to paused", StateRunning, StatePaused, false},
		{"running to stopping", StateRunning, StateStopping, false},
		{"running to failed", StateRunning, StateFailed, false},
		{"paused to running", StatePaused, StateRunning, false},
		{"paused to stopping", StatePaused, StateStopping, false},
		{"stopping to stopped", StateStopping, StateStopped, false},
		{"stopped to starting", StateStopped, StateStarting, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateMachine{state: tt.from}
			err := sm.Transition(tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transition() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && sm.State() != tt.to {
				t.Errorf("expected state %v, got %v", tt.to, sm.State())
			}
		})
	}
}

func TestStateMachine_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from State
		to   State
	}{
		{"starting to stopped", StateStarting, StateStopped},
		{"starting to paused", StateStarting, StatePaused},
		{"running to starting", StateRunning, StateStarting},
		{"running to stopped", StateRunning, StateStopped},
		{"paused to starting", StatePaused, StateStarting},
		{"paused to stopped", StatePaused, StateStopped},
		{"stopped to running", StateStopped, StateRunning},
		{"stopped to paused", StateStopped, StatePaused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StateMachine{state: tt.from}
			err := sm.Transition(tt.to)
			if err == nil {
				t.Errorf("expected error for invalid transition from %v to %v", tt.from, tt.to)
			}
		})
	}
}

func TestStateMachine_Listener(t *testing.T) {
	sm := NewStateMachine()

	var fromState, toState State
	sm.AddListener(func(from, to State) {
		fromState = from
		toState = to
	})

	err := sm.Transition(StateRunning)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fromState != StateStarting {
		t.Errorf("expected listener fromState = StateStarting, got %v", fromState)
	}
	if toState != StateRunning {
		t.Errorf("expected listener toState = StateRunning, got %v", toState)
	}
}

func TestStateMachine_IsRunning(t *testing.T) {
	sm := NewStateMachine()

	if sm.IsRunning() {
		t.Error("expected IsRunning() = false for StateStarting")
	}

	sm.Transition(StateRunning)
	if !sm.IsRunning() {
		t.Error("expected IsRunning() = true for StateRunning")
	}

	sm.Transition(StatePaused)
	if sm.IsRunning() {
		t.Error("expected IsRunning() = false for StatePaused")
	}
}

func TestStateMachine_IsPaused(t *testing.T) {
	sm := NewStateMachine()
	sm.Transition(StateRunning)

	if sm.IsPaused() {
		t.Error("expected IsPaused() = false for StateRunning")
	}

	sm.Transition(StatePaused)
	if !sm.IsPaused() {
		t.Error("expected IsPaused() = true for StatePaused")
	}
}

func TestStateMachine_CanProcess(t *testing.T) {
	tests := []struct {
		state      State
		canProcess bool
	}{
		{StateStarting, false},
		{StateRunning, true},
		{StatePaused, false},
		{StateStopping, false},
		{StateStopped, false},
		{StateFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			sm := &StateMachine{state: tt.state}
			if got := sm.CanProcess(); got != tt.canProcess {
				t.Errorf("CanProcess() = %v, want %v", got, tt.canProcess)
			}
		})
	}
}

func TestStateMachine_IsTerminal(t *testing.T) {
	tests := []struct {
		state      State
		isTerminal bool
	}{
		{StateStarting, false},
		{StateRunning, false},
		{StatePaused, false},
		{StateStopping, false},
		{StateStopped, true},
		{StateFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			sm := &StateMachine{state: tt.state}
			if got := sm.IsTerminal(); got != tt.isTerminal {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.isTerminal)
			}
		})
	}
}
