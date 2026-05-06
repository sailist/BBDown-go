package download

import (
	"testing"
)

func TestNopReporter(t *testing.T) {
	r := &NopReporter{}
	r.Report(10, 100) // should not panic
}

func TestConsoleReporter(t *testing.T) {
	r := NewConsoleReporter(100)
	r.Report(50, 100)
	if r.bar.GetMax64() != 100 {
		t.Fatalf("expected max 100, got %d", r.bar.GetMax64())
	}
}

func TestServerReporter(t *testing.T) {
	ch := make(chan ProgressMsg, 1)
	r := NewServerReporter(ch)
	r.Report(25, 100)

	select {
	case msg := <-ch:
		if msg.Current != 25 || msg.Total != 100 {
			t.Fatalf("unexpected msg: %+v", msg)
		}
	default:
		t.Fatal("expected message in channel")
	}
}

func TestServerReporter_NonBlocking(t *testing.T) {
	ch := make(chan ProgressMsg) // unbuffered
	r := NewServerReporter(ch)
	r.Report(25, 100) // should not block
}
