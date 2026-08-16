package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/messagetest"
)

func messagesOf(findings []Finding) []string {
	messages := make([]string, 0, len(findings))
	for _, f := range findings {
		messages = append(messages, f.Message)
	}
	return messages
}

func messagesWithSlotOf(findings []Finding) []string {
	messages := make([]string, 0, len(findings))
	for _, f := range findings {
		messages = append(messages, string(f.Slot)+": "+f.Message)
	}
	return messages
}

func assertFindings(t *testing.T, got []string, want ...[]string) {
	t.Helper()
	messagetest.AssertAll(t, got, want...)
}

func assertFinding(t *testing.T, got []string, parts ...string) {
	t.Helper()
	messagetest.AssertOne(t, got, parts...)
}
