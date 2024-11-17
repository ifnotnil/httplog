package httplog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ifnotnil/httplog/internal/testingx"
	"github.com/stretchr/testify/assert"
)

func parseLogJSONLines(t *testing.T, b *bytes.Buffer) []map[string]any {
	t.Helper()

	r := make([]map[string]any, 0, 10)
	for {
		l, err := b.ReadBytes('\n')

		if len(l) != 0 {
			m := map[string]any{}
			if err = json.Unmarshal(l, &m); err != nil {
				t.Errorf("error while json unmarshal log line: %s", err.Error())
			} else {
				r = append(r, m)
			}
		}

		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Errorf("error while reading logs from byte buffer: %s", err.Error())
			}
			break
		}
	}

	return r
}

type serverTestCase struct {
	t          *testing.T
	httpClient *http.Client
	testSrv    *httptest.Server
	logOutput  *bytes.Buffer
}

func (s *serverTestCase) Init(handler func(w http.ResponseWriter, r *http.Request)) {
	s.t.Helper()
	s.logOutput = &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(s.logOutput, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch {
			case len(groups) == 0 && a.Key == "time":
				return slog.Attr{}
			case len(groups) == 0 && a.Key == "duration":
				return slog.Attr{}
			case len(groups) == 1 && groups[0] == "request" && a.Key == "host":
				return slog.Attr{}
			case len(groups) == 1 && groups[0] == "request" && a.Key == "remoteAddr":
				return slog.Attr{}
			default:
				return a
			}
		},
	}))

	il := NewHTTPLogger(
		WithLogger(logger),
		WithLogInLevel(slog.LevelInfo),
	)

	s.testSrv = httptest.NewServer(il.Handler(http.HandlerFunc(handler)))

	s.httpClient = NewHTTPClient(2 * time.Second)

	s.t.Cleanup(s.Close)
}

func (s *serverTestCase) Do(
	ctx context.Context,
	requestFn func(ctx context.Context, srvURL string) (*http.Request, error),
	assertResponse func(t *testing.T, r *http.Response),
) {
	s.t.Helper()
	req, err := requestFn(ctx, s.testSrv.URL)
	if err != nil {
		s.t.Fatalf("error while creating http request: %s", err.Error())
	}
	res, err := s.httpClient.Do(req)
	if err != nil {
		s.t.Fatalf("error while performing http request: %s", err.Error())
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if assertResponse != nil {
		assertResponse(s.t, res)
	}
}

func (s *serverTestCase) Verify(expected []map[string]any) {
	s.t.Helper()
	logsObjects := parseLogJSONLines(s.t, s.logOutput)
	assert.Equal(s.t, expected, logsObjects)
}

func (s *serverTestCase) Close() {
	s.testSrv.Close()
}

func TestInbound(t *testing.T) {
	tests := map[string]struct {
		srvHandler     func(t *testing.T) func(w http.ResponseWriter, r *http.Request)
		requestFn      func(ctx context.Context, srvURL string) (*http.Request, error)
		assertResponse func(t *testing.T, r *http.Response)
		expectedLogs   []map[string]any
	}{
		"simple": {
			requestFn: func(ctx context.Context, srvURL string) (*http.Request, error) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, srvURL, strings.NewReader(`"request body"`))
				if err != nil {
					return nil, err
				}
				req.Header.Set(`Content-Type`, `application/json`)
				return req, nil
			},
			srvHandler: func(t *testing.T) func(w http.ResponseWriter, r *http.Request) { //nolint:thelper
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// assert received request
					requestBody := testingx.ReadAndClose(t, r.Body)
					assert.Equal(t, `"request body"`, string(requestBody))

					w.Header().Set(`Content-Type`, `application/json`)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`"response body"`))
				})
			},
			assertResponse: func(t *testing.T, r *http.Response) { //nolint:thelper
				// assert received response
				requestBody := testingx.ReadAndClose(t, r.Body)
				assert.Equal(t, `"response body"`, string(requestBody))
			},
			expectedLogs: []map[string]any{
				{
					"level": "INFO",
					"msg":   "http inbound",
					"request": map[string]any{
						"body":          map[string]any{"size": float64(14), "value": `\"request body\"`}, // json as json string
						"contentLength": float64(14),
						"headers":       map[string]any{"Accept-Encoding": "gzip", "Content-Length": "14", "Content-Type": "application/json", "User-Agent": "Go-http-client/1.1"},
						"method":        "GET",
						"proto":         "HTTP/1.1",
						"requestUri":    "/",
						"url":           map[string]any{"fragment": "", "full": ":///", "host": "", "opaque": "", "path": "/", "scheme": ""},
					},
					"response": map[string]any{
						"body":    map[string]any{"size": float64(15), "value": `\"response body\"`}, // json as json string
						"headers": map[string]any{"Content-Type": "application/json"},
						"status":  map[string]any{"code": float64(200), "name": "OK"},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			srv := serverTestCase{t: t}
			srv.Init(tc.srvHandler(t))
			srv.Do(ctx, tc.requestFn, tc.assertResponse)
			srv.Verify(tc.expectedLogs)
		})
	}
}
