package handlers

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestEventHubsDoNotCrossPublish(t *testing.T) {
	hubA := NewEventHub()
	hubB := NewEventHub()
	clientA, cancelA := hubA.Subscribe()
	defer cancelA()
	clientB, cancelB := hubB.Subscribe()
	defer cancelB()

	assert.True(t, hubA.Publish(`{"type":"reload"}`))
	assert.Eq(t, `{"type":"reload"}`, <-clientA)
	select {
	case <-clientB:
		t.Fatal("event leaked to another hub")
	default:
	}
	assert.NoErr(t, hubA.Close())
	assert.NoErr(t, hubA.Close())
}
