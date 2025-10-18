package agent

import (
	"testing"

	"github.com/paxren/metrics/internal/agent"
	"github.com/paxren/metrics/internal/repository"
)

func TestAgent_Send(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		r    repository.Repository
		want []error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := agent.NewAgent(tt.r)
			got := a.Send()
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Send() = %v, want %v", got, tt.want)
			}
		})
	}
}
