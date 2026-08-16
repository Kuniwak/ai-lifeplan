package panictest_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/panictest"
)

func TestRecoveredShouldReturnWhatTheCallPanickedWith(t *testing.T) {
	refused := panictest.Recovered(func() { panic("契約違反") })

	if refused != "契約違反" {
		t.Errorf("Recovered() = %v, want %q", refused, "契約違反")
	}
}

func TestRecoveredShouldReturnNilWhenTheCallReturns(t *testing.T) {
	refused := panictest.Recovered(func() {})

	if refused != nil {
		t.Errorf("Recovered() = %v, want nil", refused)
	}
}

func TestRecoveredShouldReturnNonNilWhenTheCallPanickedWithNil(t *testing.T) {
	refused := panictest.Recovered(func() { panic(nil) })

	if refused == nil {
		t.Error("panic(nil) が「拒まれなかった」と同じに見えている")
	}
}

func TestMessageShouldReturnThePanicValueAsText(t *testing.T) {
	msg, refused := panictest.Message(func() { panic(42) })

	if !refused {
		t.Error("Message() said the call returned, but it panicked")
	}
	if msg != "42" {
		t.Errorf("Message() = %q, want %q", msg, "42")
	}
}

func TestMessageShouldSayTheCallReturnedWhenItReturns(t *testing.T) {
	msg, refused := panictest.Message(func() {})

	if refused {
		t.Error("Message() said the call panicked, but it returned")
	}
	if msg != "" {
		t.Errorf("Message() = %q, want %q", msg, "")
	}
}

func TestMessageShouldSeparateAnEmptyMessageFromNoPanicAtAll(t *testing.T) {
	msg, refused := panictest.Message(func() { panic("") })

	if !refused {
		t.Error(`panic("") が「拒まれなかった」と同じに見えている`)
	}
	if msg != "" {
		t.Errorf("Message() = %q, want %q", msg, "")
	}
}
