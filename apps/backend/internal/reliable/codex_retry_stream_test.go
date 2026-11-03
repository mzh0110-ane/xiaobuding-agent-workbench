package reliable

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const codexRetryFixtureMessage = "Reconnecting... 2/5 (stream disconnected before completion: IO error: peer closed connection without sending TLS close_notify: https://docs.rs/rustls/latest/rustls/manual/_03_howto/index.html#unexpected-eof)"

const codexRetryFixtureStart = "{\"type\":\"thread.started\",\"thread_id\":\"recovered-native-thread\"}\n{\"type\":\"turn.started\"}\n"
const codexRetryFixtureComplete = "{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":100,\"output_tokens\":50}}\n"

func codexRetryFixtureError(t *testing.T, message string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"type": "error", "message": message})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw) + "\n"
}

func TestParseSessionAcceptsCompletedNativeReconnect(t *testing.T) {
	var events strings.Builder
	events.WriteString(codexRetryFixtureStart)
	for i := 0; i < 25; i++ {
		fmt.Fprintf(&events, "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_%d\",\"type\":\"command_execution\",\"status\":\"completed\",\"aggregated_output\":\"source evidence\"}}\n", i)
	}
	events.WriteString(codexRetryFixtureError(t, codexRetryFixtureMessage))
	events.WriteString("{\"type\":\"item.completed\",\"item\":{\"type\":\"command_execution\",\"status\":\"completed\",\"aggregated_output\":\"continued source evidence\"}}\n")
	events.WriteString("{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"final schema response\"}}\n")
	events.WriteString(codexRetryFixtureComplete)
	raw := []byte(events.String())
	original := bytes.Clone(raw)
	id, err := parseSession(raw)
	if err != nil || id != "recovered-native-thread" {
		t.Fatalf("completed native reconnect was rejected: session=%q error=%v", id, err)
	}
	if !bytes.Equal(raw, original) {
		t.Fatal("session parsing changed the saved native events")
	}
}

func TestParseSessionReconnectRequiresLaterCompletion(t *testing.T) {
	retry := codexRetryFixtureError(t, codexRetryFixtureMessage)
	for name, events := range map[string]string{
		"retry ends stream":          codexRetryFixtureStart + retry,
		"retry after old completion": codexRetryFixtureStart + codexRetryFixtureComplete + retry,
		"later turn never completes": codexRetryFixtureStart + retry + codexRetryFixtureComplete + "{\"type\":\"turn.started\"}\n",
	} {
		t.Run(name, func(t *testing.T) {
			if id, err := parseSession([]byte(events)); err == nil {
				t.Fatalf("accepted incomplete native turn: session=%q", id)
			}
		})
	}
}

func TestParseSessionReconnectPreservesFatalFailures(t *testing.T) {
	retry := codexRetryFixtureError(t, codexRetryFixtureMessage)
	for name, middle := range map[string]string{
		"generic error":             codexRetryFixtureError(t, "permission denied"),
		"fatal before reconnect":    codexRetryFixtureError(t, "authentication failed") + retry,
		"fatal after reconnect":     retry + codexRetryFixtureError(t, "authentication failed"),
		"turn failed":               "{\"type\":\"turn.failed\",\"error\":{\"message\":\"request failed\"}}\n",
		"failed after reconnect":    retry + "{\"type\":\"turn.failed\"}\n",
		"malformed JSON":            retry + "{\"type\":\n",
		"duplicate session":         retry + "{\"type\":\"thread.started\",\"thread_id\":\"another-thread\"}\n",
		"repeated same session":     retry + "{\"type\":\"thread.started\",\"thread_id\":\"recovered-native-thread\"}\n",
		"malformed reconnect field": "{\"type\":\"error\",\"message\":{\"text\":\"Reconnecting... 2/5 (temporary error)\"}}\n",
	} {
		t.Run(name, func(t *testing.T) {
			events := codexRetryFixtureStart + middle + codexRetryFixtureComplete
			if id, err := parseSession([]byte(events)); err == nil {
				t.Fatalf("accepted failed or ambiguous native turn: session=%q", id)
			}
		})
	}
}

func TestParseSessionRejectsNoncanonicalReconnectErrors(t *testing.T) {
	for name, message := range map[string]string{
		"word only":             "fatal error while reconnecting to provider",
		"embedded notice":       "fatal error: " + codexRetryFixtureMessage,
		"leading space":         " " + codexRetryFixtureMessage,
		"trailing text":         codexRetryFixtureMessage + " request failed",
		"zero attempt":          "Reconnecting... 0/5 (temporary error)",
		"zero limit":            "Reconnecting... 2/0 (temporary error)",
		"attempt exceeds limit": "Reconnecting... 6/5 (temporary error)",
		"unbounded indices":     "Reconnecting... 1000/1000 (temporary error)",
		"leading zero":          "Reconnecting... 02/5 (temporary error)",
		"missing reason":        "Reconnecting... 2/5 ()",
		"blank reason":          "Reconnecting... 2/5 ( )",
		"multiline reason":      "Reconnecting... 2/5 (temporary\nerror)",
	} {
		t.Run(name, func(t *testing.T) {
			events := codexRetryFixtureStart + codexRetryFixtureError(t, message) + codexRetryFixtureComplete
			if id, err := parseSession([]byte(events)); err == nil {
				t.Fatalf("accepted ordinary or malformed error notice: session=%q message=%q", id, message)
			}
		})
	}
}
