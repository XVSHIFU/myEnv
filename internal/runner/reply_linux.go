package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
)

func awaitSupervisorCompletion(ctx context.Context, reader *bufio.Reader, terminal bool, stop func() error) (supervisorReply, error) {
	for {
		var reply supervisorReply
		if err := decodeSupervisorReply(reader, &reply); err != nil {
			return reply, err
		}
		if !reply.Stopped {
			if reply.Ready {
				return reply, fmt.Errorf("unexpected supervisor readiness event after launch")
			}
			return reply, nil
		}
		if !terminal || reply.Complete || reply.Ready || reply.Canceled || reply.Code != 0 || reply.Error != "" {
			return reply, fmt.Errorf("unexpected supervisor stop event")
		}
		// Cancellation closes the lifetime pipe. Drain the final confirmation
		// instead of discarding it because a stop notification was already queued.
		if ctx.Err() != nil {
			continue
		}
		if err := stop(); err != nil {
			return reply, err
		}
	}
}

func validSupervisorReadiness(reply supervisorReply) bool {
	return reply.Ready && !reply.Stopped && !reply.Complete && !reply.Canceled && reply.Code == 0 && reply.Error == ""
}

// The supervisor's json.Encoder emits one newline-terminated reply at a time.
// Bound each frame, not the lifetime stream: stop events may recur indefinitely.
func decodeSupervisorReply(reader *bufio.Reader, reply *supervisorReply) error {
	line, err := reader.ReadSlice('\n')
	if len(line) > 64<<10 || err == bufio.ErrBufferFull {
		return fmt.Errorf("supervisor reply exceeds 64 KiB")
	}
	if err != nil {
		return fmt.Errorf("read supervisor reply: %w", err)
	}
	*reply = supervisorReply{}
	return json.Unmarshal(line, reply)
}
