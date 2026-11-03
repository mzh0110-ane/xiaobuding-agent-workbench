package reliable

import (
	"bytes"
	"testing"
)

func TestNativeHarnessSavedCredentialsAreRedactedFromEvidence(t *testing.T) {
	credentials := []byte("version: 1\nrecords:\n  deepseek-account-platform/default:\n    kind: grant\n    payload:\n      version: 1\n      token: native-harness-private-token-value\n      issuer: https://fixture.invalid\n")
	redact := makeRedactor(credentials)
	output := redact([]byte("provider=deepseek-account token=native-harness-private-token-value"))
	if bytes.Contains(output, []byte("native-harness-private-token-value")) {
		t.Fatal("a credential saved only in native Harness escaped through evidence")
	}
	if !bytes.Contains(output, []byte("provider=deepseek-account")) {
		t.Fatal("credential filtering removed the non-sensitive provider identity")
	}
}
