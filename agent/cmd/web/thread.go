// cmd/web/thread.go — Per-thread agent state management.
package main

import (
	"sync"

	rtools "vibex/agent/agents/runtime/tools"
	"vibex/agent/agents/skills"

	"github.com/openai/openai-go/v3/responses"
)

// Per-thread agent state.
type threadState struct {
	mu            sync.RWMutex
	messages      []responses.ResponseInputItemUnionParam
	todo          *rtools.TodoStore
	skillState    *skills.State
	activeSession string
	stepType      string // 当前 spec 步骤类型：routing/clarify/spec-goal/spec-feature/spec-bug/tdd-design/tdd-run/tdd-iterate/spec-apply/canvas-exhibit
}

var threadStates = sync.Map{} // threadID → *threadState

// getThreadState returns (creating if needed) the state for a thread.
func getThreadState(threadID string) *threadState {
	v, _ := threadStates.LoadOrStore(threadID, &threadState{
		todo:       rtools.NewTodoStore(),
		skillState: skills.NewState(),
	})
	return v.(*threadState)
}
