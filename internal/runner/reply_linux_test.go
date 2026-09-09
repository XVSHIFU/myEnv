package runner

import (
	"bufio"
	"strings"
	"testing"
)

func TestSupervisorReplyFraming(t *testing.T) {
	const events = 10000
	reader := bufio.NewReaderSize(strings.NewReader(strings.Repeat("{\"Stopped\":true}\n", events)+"{\"Complete\":true,\"Code\":23}\n"), (64<<10)+1)
	var reply supervisorReply
	for i := 0; i < events; i++ {
		if err := decodeSupervisorReply(reader, &reply); err != nil || !reply.Stopped || reply.Complete {
			t.Fatalf("event %d: %+v %v", i, reply, err)
		}
	}
	if err := decodeSupervisorReply(reader, &reply); err != nil || reply.Stopped || !reply.Complete || reply.Code != 23 {
		t.Fatalf("final: %+v %v", reply, err)
	}
	for _, input := range []string{"{\"Ready\":true}", "{} {}\n", strings.Repeat(" ", 64<<10) + "{}\n", "{broken}\n"} {
		reader := bufio.NewReaderSize(strings.NewReader(input), (64<<10)+1)
		if err := decodeSupervisorReply(reader, &reply); err == nil {
			t.Fatalf("accepted malformed or oversized frame length=%d", len(input))
		}
	}
}
