package reliable

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestZcodeResultRequiresNativeCompletionAndSession(t *testing.T) {
	valid := []byte("{\"type\":\"zcode.event\",\"event\":{\"type\":\"turn.completed\",\"sessionId\":\"s\",\"payload\":{\"resultType\":\"success\"}}}\n{\"type\":\"result\",\"session\":\"s\",\"response\":\"{}\"}\n")
	result, err := parseZcodeResult(valid)
	require.NoError(t, err)
	require.Equal(t, "s", result.Session)
	for _, raw := range []string{
		"{\"type\":\"result\",\"session\":\"s\",\"response\":\"{}\"}\n",
		"{\"type\":\"zcode.event\",\"event\":{\"type\":\"turn.failed\",\"sessionId\":\"s\"}}\n",
		string(valid) + "{\"type\":\"result\",\"session\":\"other\",\"response\":\"{}\"}\n",
		string(valid) + "broken\n",
	} {
		_, err = parseZcodeResult([]byte(raw))
		require.Error(t, err)
	}
}
