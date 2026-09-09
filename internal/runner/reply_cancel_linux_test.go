package runner

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSupervisorStopTransitions(t *testing.T) {
	for _, mode := range []string{"resume", "cancel-on-resume", "stop-error"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("stop failed")
			calls := 0
			reader := bufio.NewReader(strings.NewReader("{\"Stopped\":true}\n{\"Stopped\":true}\n{\"Complete\":true,\"Code\":23}\n"))
			reply, err := awaitSupervisorCompletion(ctx, reader, true, func() error {
				calls++
				if mode == "cancel-on-resume" {
					cancel()
				}
				if mode == "stop-error" {
					return failure
				}
				return nil
			})
			if mode == "stop-error" {
				if !errors.Is(err, failure) || calls != 1 {
					t.Fatalf("stop failure lost: %d %v", calls, err)
				}
				return
			}
			wantCalls := 2
			if mode == "cancel-on-resume" {
				wantCalls = 1
			}
			if err != nil || !reply.Complete || reply.Code != 23 || calls != wantCalls {
				t.Fatalf("transition: calls=%d reply=%+v error=%v", calls, reply, err)
			}
		})
	}
}

func TestCanceledStopRejectsInvalidEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, event := range []struct {
		terminal bool
		frame    string
	}{
		{false, "{\"Stopped\":true}\n"},
		{true, "{\"Stopped\":true,\"Complete\":true}\n"},
		{true, "{\"Stopped\":true,\"Ready\":true}\n"},
		{true, "{\"Stopped\":true,\"Canceled\":true}\n"},
		{true, "{\"Stopped\":true,\"Code\":17}\n"},
		{true, "{\"Stopped\":true,\"Error\":\"failed\"}\n"},
	} {
		_, err := awaitSupervisorCompletion(ctx, bufio.NewReader(strings.NewReader(event.frame)), event.terminal, func() error {
			t.Fatal("invalid event caused stop")
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "unexpected supervisor stop event") {
			t.Fatalf("invalid event accepted: %v", err)
		}
	}
}

func TestCanceledStopDrainsCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, complete := range []bool{true, false} {
		stream := "{\"Stopped\":true}\n{\"Stopped\":true}\n"
		if complete {
			stream += "{\"Complete\":true,\"Canceled\":true,\"Code\":137}\n"
		}
		reply, err := awaitSupervisorCompletion(ctx, bufio.NewReader(strings.NewReader(stream)), true, func() error {
			t.Fatal("canceled caller must not stop")
			return nil
		})
		if complete {
			if err != nil || !reply.Complete || !reply.Canceled || reply.Code != 137 {
				t.Fatalf("lost completion: %+v %v", reply, err)
			}
		} else if err == nil || reply.Complete {
			t.Fatalf("missing confirmation accepted: %+v %v", reply, err)
		}
	}
}
