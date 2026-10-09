package jsonlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type logEntry struct {
	Level      string            `json:"level"`
	Time       string            `json:"time"`
	Message    string            `json:"message"`
	Properties map[string]string `json:"properties"`
	Trace      string            `json:"trace"`
}

func decodeSingleEntry(t *testing.T, buf *bytes.Buffer) logEntry {
	t.Helper()

	line := bytes.TrimSpace(buf.Bytes())
	if len(line) == 0 {
		t.Fatal("logger wrote no output")
	}

	var entry logEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		t.Fatalf("logger output is not valid JSON: %v (%q)", err, line)
	}

	return entry
}

func TestPrintInfoWritesStructuredLine(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, LevelInfo)

	l.PrintInfo("ready", map[string]string{"port": "4000"})

	entry := decodeSingleEntry(t, &buf)
	if entry.Level != "INFO" {
		t.Errorf("level = %q, want INFO", entry.Level)
	}
	if entry.Message != "ready" {
		t.Errorf("message = %q, want ready", entry.Message)
	}
	if entry.Time == "" {
		t.Error("time is empty")
	}
	if entry.Properties["port"] != "4000" {
		t.Errorf("properties[port] = %q, want 4000", entry.Properties["port"])
	}
	if entry.Trace != "" {
		t.Errorf("info line should not include a trace, got %d bytes", len(entry.Trace))
	}
}

func TestPrintErrorWritesErrorLevelWithTrace(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, LevelInfo)

	l.PrintError(errors.New("db down"), nil)

	entry := decodeSingleEntry(t, &buf)
	if entry.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", entry.Level)
	}
	if entry.Message != "db down" {
		t.Errorf("message = %q, want db down", entry.Message)
	}
	if entry.Trace == "" {
		t.Error("error line should include a stack trace")
	}
}

func TestLoggerFiltersBelowMinimumLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, LevelFatal)

	l.PrintInfo("quiet", nil)
	l.PrintError(errors.New("also quiet"), nil)

	if buf.Len() != 0 {
		t.Errorf("expected no output below the minimum level, got %q", buf.String())
	}
}

func TestLoggerLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelInfo, "INFO"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestWriteEmitsErrorLine(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, LevelInfo)

	n, err := l.Write([]byte("boom"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n == 0 {
		t.Error("Write() reported zero bytes written")
	}

	entry := decodeSingleEntry(t, &buf)
	if entry.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", entry.Level)
	}
	if entry.Message != "boom" {
		t.Errorf("message = %q, want boom", entry.Message)
	}
	if !strings.Contains(entry.Trace, "goroutine") {
		t.Error("Write() line should include a stack trace")
	}
}
