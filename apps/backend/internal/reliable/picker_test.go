package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProjectPickerResponses(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		err        error
		status     int
	}{
		{"selection", `E:\项目 中文`, nil, http.StatusOK},
		{"cancel", "", nil, http.StatusOK},
		{"failure", "", errors.New("native dialog failed"), http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &HTTP{Engine: &Engine{}, Token: "picker-test", PickProject: func(context.Context) (string, error) { return tc.path, tc.err }}
			response := httpRequest(h, "POST", "/api/v1/reliable/pick-project", `{}`, h.Token)
			require.Equal(t, tc.status, response.Code)
			if tc.err == nil {
				var result map[string]string
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				require.Equal(t, tc.path, result["path"])
			} else {
				require.Contains(t, response.Body.String(), "native dialog failed")
			}
		})
	}
}

func TestProjectPickerOutlastsServerWriteTimeout(t *testing.T) {
	h := &HTTP{Engine: &Engine{}, Token: "picker-test", PickProject: func(context.Context) (string, error) {
		<-time.After(120 * time.Millisecond)
		return `E:\项目 中文`, nil
	}}
	server := httptest.NewUnstartedServer(h)
	t.Cleanup(server.Close)
	server.Config.WriteTimeout = 40 * time.Millisecond
	server.Start()
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/reliable/pick-project", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-XBD-Token", h.Token)
	response, err := server.Client().Do(req)
	require.NoError(t, err, "folder selection must survive the ordinary short API write timeout")
	t.Cleanup(func() { _ = response.Body.Close() })
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.JSONEq(t, `{"path":"E:\\项目 中文"}`, string(body))
}
