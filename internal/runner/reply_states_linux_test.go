package runner

import (
	"bufio"
	"context"
	"strings"
	"testing"
)

func TestSupervisorRejectsReadinessAsCompletion(t *testing.T) {
	for _, frame := range []string{
		`{"Ready":true}`,
		`{"Ready":true,"Complete":true,"Code":0}`,
		`{"Ready":true,"Complete":true,"Canceled":true,"Code":137}`,
	} {
		_, err := awaitSupervisorCompletion(context.Background(), bufio.NewReader(strings.NewReader(frame+"\n")), false, func() error {
			t.Fatal("readiness must not stop caller")
			return nil
		})
		if err == nil {
			t.Fatalf("accepted readiness as completion: %s", frame)
		}
	}
}

func TestSupervisorReadinessState(t *testing.T) {
	if !validSupervisorReadiness(supervisorReply{Ready: true}) {
		t.Fatal("valid readiness rejected")
	}
	for _, reply := range []supervisorReply{
		{}, {Ready: true, Complete: true}, {Ready: true, Stopped: true},
		{Ready: true, Canceled: true}, {Ready: true, Code: 17}, {Ready: true, Error: "failed"},
	} {
		if validSupervisorReadiness(reply) {
			t.Fatalf("accepted invalid readiness: %+v", reply)
		}
	}
}
