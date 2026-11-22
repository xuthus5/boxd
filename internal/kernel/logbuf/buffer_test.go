package logbuf_test

import (
	"testing"

	"github.com/xuthus5/boxd/internal/kernel/logbuf"
	"github.com/xuthus5/boxd/kernel"
)

func TestBufferKeepsNewestEntriesInOrder(t *testing.T) {
	buffer := logbuf.New(3)
	for _, message := range []string{"one", "two", "three", "four"} {
		buffer.WriteLog(kernel.LogEntry{Message: message})
	}
	entries := buffer.RecentLogs(2)
	if len(entries) != 2 || entries[0].Message != "three" || entries[1].Message != "four" {
		t.Fatalf("entries = %+v", entries)
	}
}
