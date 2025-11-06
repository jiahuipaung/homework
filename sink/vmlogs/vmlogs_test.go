package victorialogs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flashcatcloud/go-pkg/x/log"
	"go.uber.org/zap/zapcore"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	log.SetConfig(log.Config{
		Level: zap.NewAtomicLevelAt(zapcore.InfoLevel),
	}, "stdout")
}

// mockVMLogsServer is a mock VictoriaLogs server for testing.
type mockVMLogsServer struct {
	server      *httptest.Server
	requests    chan []byte
	requestLock sync.Mutex
	allRequests [][]byte
	failCount   int32
	failUntil   int32
	endpoints   []string
	url         string

	// handlerFunc can be overridden by tests for custom behavior
	handlerFunc http.HandlerFunc
}

// newMockVMLogsServer creates a new mock server. The server will use the defaultHandler
// unless the handlerFunc field is set.
func newMockVMLogsServer() *mockVMLogsServer {
	s := &mockVMLogsServer{
		requests: make(chan []byte, 100),
	}
	// The server calls the ServeHTTP method, which then dispatches to the appropriate handler.
	s.server = httptest.NewServer(s)
	s.url = s.server.URL
	s.endpoints = []string{s.server.URL}
	return s
}

func (s *mockVMLogsServer) Close() {
	s.server.Close()
	close(s.requests)
}

// ServeHTTP makes mockVMLogsServer implement the http.Handler interface.
func (s *mockVMLogsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.handlerFunc != nil {
		s.handlerFunc(w, r)
		return
	}
	s.defaultHandler(w, r)
}

func (s *mockVMLogsServer) defaultHandler(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&s.failCount) < s.failUntil {
		atomic.AddInt32(&s.failCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.requestLock.Lock()
	s.allRequests = append(s.allRequests, body)
	s.requestLock.Unlock()

	s.requests <- body
	w.WriteHeader(http.StatusNoContent)
}

func (s *mockVMLogsServer) getReceivedLogs() []types.ExtractedLog {
	var logs []types.ExtractedLog
	s.requestLock.Lock()
	defer s.requestLock.Unlock()

	for _, reqBody := range s.allRequests {
		lines := strings.Split(strings.TrimSpace(string(reqBody)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			var log types.ExtractedLog
			if err := json.Unmarshal([]byte(line), &log); err == nil {
				logs = append(logs, log)
			}
		}
	}
	return logs
}

func TestVMLogsOutput_IsDiff(t *testing.T) {
	base := &VMLogsOutput{
		Endpoints:      []string{"http://localhost:9090"},
		Username:       "user",
		Password:       "pass",
		SkipTlsVerify:  false,
		StreamFields:   []string{"stream1", "stream2"},
		BatchActions:   1000,
		BatchSize:      1024,
		NumOfWorker:    2,
		RequestTimeout: 10 * time.Second,
	}

	t.Run("is different with nil", func(t *testing.T) {
		assert.True(t, base.IsDiff(nil))
	})

	t.Run("is different with other type", func(t *testing.T) {
		type dummyOutput struct{ types.Output }
		assert.True(t, base.IsDiff(&dummyOutput{}))
	})

	t.Run("is same with identical config", func(t *testing.T) {
		other := *base
		assert.False(t, base.IsDiff(&other))
	})

	t.Run("is different with different endpoints", func(t *testing.T) {
		other := *base
		other.Endpoints = []string{"http://localhost:9091", "http://localhost:9092"}
		assert.True(t, base.IsDiff(&other))
	})
}

func TestVMLogsOutput_Init(t *testing.T) {
	t.Run("init fails with no endpoints", func(t *testing.T) {
		output := &VMLogsOutput{}
		err := output.Init(context.Background(), 1)
		assert.Error(t, err)
		assert.Equal(t, "vmlogs endpoints are not set", err.Error())
	})

	t.Run("init succeeds with valid config", func(t *testing.T) {
		output := &VMLogsOutput{
			Endpoints: []string{"http://localhost:9090"},
		}
		err := output.Init(context.Background(), 2)
		assert.NoError(t, err)
		assert.NotNil(t, output.Client)
		assert.Equal(t, 1000, output.BatchActions) // Check default
		assert.Equal(t, 2<<20, output.BatchSize)   // Check default
		assert.Equal(t, 2, output.NumOfWorker)     // Check worker override
	})
}

func TestVMLogsOutput_Sink_BatchingByActions(t *testing.T) {
	mockServer := newMockVMLogsServer()
	defer mockServer.Close()

	output := &VMLogsOutput{
		Endpoints:          mockServer.endpoints,
		BatchActions:       10,
		BatchFlushInterval: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := output.Init(ctx, 1)
	assert.NoError(t, err)

	err = output.Start(ctx, zap.NewNop())
	assert.NoError(t, err)
	defer output.Stop(ctx)

	// Send 9 logs, should not trigger a flush
	for i := 0; i < 9; i++ {
		err := output.Sink(ctx, types.ExtractedLog{"message": fmt.Sprintf("log %d", i)})
		assert.NoError(t, err)
	}

	select {
	case <-mockServer.requests:
		t.Fatal("should not have received a request yet")
	case <-time.After(100 * time.Millisecond):
		// Correct, no request yet
	}

	// Send 10th log, should trigger a flush
	err = output.Sink(ctx, types.ExtractedLog{"message": "log 9"})
	assert.NoError(t, err)

	select {
	case reqBody := <-mockServer.requests:
		assert.Contains(t, string(reqBody), "\"message\":\"log 0\"")
		assert.Contains(t, string(reqBody), "\"message\":\"log 9\"")
		lines := strings.Split(strings.TrimSpace(string(reqBody)), "\n")
		assert.Len(t, lines, 10)
	case <-time.After(100 * time.Second):
		t.Fatal("timed out waiting for request")
	}
}

func TestVMLogsOutput_Sink_BatchingByInterval(t *testing.T) {
	mockServer := newMockVMLogsServer()
	defer mockServer.Close()

	output := &VMLogsOutput{
		Endpoints:          mockServer.endpoints,
		BatchActions:       100,
		BatchFlushInterval: 100 * time.Millisecond, // Short flush interval
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := output.Init(ctx, 1)
	assert.NoError(t, err)

	err = output.Start(ctx, zap.NewNop())
	assert.NoError(t, err)
	defer output.Stop(ctx)

	// Send a few logs
	for i := 0; i < 3; i++ {
		err := output.Sink(ctx, types.ExtractedLog{"message": fmt.Sprintf("log %d", i)})
		assert.NoError(t, err)
	}

	// Wait for the flush interval to pass
	select {
	case reqBody := <-mockServer.requests:
		assert.Contains(t, string(reqBody), "\"message\":\"log 0\"")
		assert.Contains(t, string(reqBody), "\"message\":\"log 2\"")
		lines := strings.Split(strings.TrimSpace(string(reqBody)), "\n")
		assert.Len(t, lines, 3)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for request")
	}
}

func TestVMLogsOutput_Sink_BatchingBySize(t *testing.T) {
	mockServer := newMockVMLogsServer()
	defer mockServer.Close()

	logEntry := types.ExtractedLog{"message": "a"} // Marshaled: {"message":"a"} -> 16 bytes
	logBytes, _ := json.Marshal(logEntry)
	logSize := len(logBytes)

	output := &VMLogsOutput{
		Endpoints:          mockServer.endpoints,
		BatchActions:       100,
		BatchSize:          logSize*5 + 1, // Just enough for 5 logs
		BatchFlushInterval: 1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := output.Init(ctx, 1)
	assert.NoError(t, err)

	err = output.Start(ctx, zap.NewNop())
	assert.NoError(t, err)
	defer output.Stop(ctx)

	// Send 5 logs, should not trigger flush yet (size is 5*logSize)
	for i := 0; i < 5; i++ {
		err := output.Sink(ctx, logEntry)
		assert.NoError(t, err)
	}

	select {
	case <-mockServer.requests:
		t.Fatal("should not have received a request yet")
	case <-time.After(100 * time.Millisecond):
		// Correct
	}

	// Send 6th log, should trigger flush
	err = output.Sink(ctx, logEntry)
	assert.NoError(t, err)

	select {
	case reqBody := <-mockServer.requests:
		lines := strings.Split(strings.TrimSpace(string(reqBody)), "\n")
		assert.Len(t, lines, 6)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for request")
	}
}

func TestVMLogsOutput_Sink_Retry(t *testing.T) {
	mockServer := newMockVMLogsServer()
	defer mockServer.Close()

	// Fail the first 2 requests
	mockServer.failUntil = 2

	output := &VMLogsOutput{
		Endpoints:          mockServer.endpoints,
		BatchActions:       5,
		BatchFlushInterval: 1 * time.Second,
		MaxRetryBackoff:    10 * time.Millisecond, // Use MaxRetryBackoff for testing
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := output.Init(ctx, 1)
	assert.NoError(t, err)

	err = output.Start(ctx, zap.NewNop())
	assert.NoError(t, err)
	defer output.Stop(ctx)

	for i := 0; i < 5; i++ {
		err := output.Sink(ctx, types.ExtractedLog{"message": "log"})
		assert.NoError(t, err)
	}

	// It should fail, retry, then succeed
	select {
	case reqBody := <-mockServer.requests:
		assert.Equal(t, int32(2), atomic.LoadInt32(&mockServer.failCount), "should have failed twice")
		lines := strings.Split(strings.TrimSpace(string(reqBody)), "\n")
		assert.Len(t, lines, 5)
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for request")
	}
}

func TestVMLogsOutput_Sink_MultiEndpoint(t *testing.T) {
	var (
		requestCounts = make(map[string]int32)
		mu            sync.Mutex
	)

	handler := func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		host := r.Host
		requestCounts[host]++
		w.WriteHeader(http.StatusNoContent)
	}

	server1 := httptest.NewServer(http.HandlerFunc(handler))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(handler))
	defer server2.Close()

	output := &VMLogsOutput{
		Endpoints:          []string{server1.URL, server2.URL},
		BatchActions:       1, // Flush every time
		BatchFlushInterval: 1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := output.Init(ctx, 1)
	assert.NoError(t, err)

	err = output.Start(ctx, zap.NewNop())
	assert.NoError(t, err)

	// Send 10 logs, which should result in 10 requests
	for i := 0; i < 10; i++ {
		err := output.Sink(ctx, types.ExtractedLog{"message": "log"})
		assert.NoError(t, err)
		time.Sleep(20 * time.Millisecond) // Give time for the request to be processed
	}

	output.Stop(ctx) // Stop to ensure all inflight requests are finished

	s1Host := strings.TrimPrefix(server1.URL, "http://")
	s2Host := strings.TrimPrefix(server2.URL, "http://")

	mu.Lock()
	s1Count := requestCounts[s1Host]
	s2Count := requestCounts[s2Host]
	mu.Unlock()

	assert.Equal(t, int32(10), s1Count+s2Count, "should have received 10 requests total")
	assert.True(t, s1Count > 0, "server 1 should have received requests")
	assert.True(t, s2Count > 0, "server 2 should have received requests")
	// Check for rough balance
	assert.InDelta(t, s1Count, s2Count, 2, "requests should be roughly balanced")
}

func TestVMLogsOutput_Sink_TimeParsing(t *testing.T) {
	mockServer := newMockVMLogsServer()
	defer mockServer.Close()

	// Use a custom handler to check the query parameter
	mockServer.handlerFunc = func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "my_timestamp", r.URL.Query().Get("_time_field"))
		mockServer.defaultHandler(w, r)
	}

	output := &VMLogsOutput{
		Endpoints:          mockServer.endpoints,
		BatchActions:       1,
		TimeField:          "my_timestamp",
		BatchFlushInterval: 1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := output.Init(ctx, 1)
	assert.NoError(t, err)
	err = output.Start(ctx, zap.NewNop())
	assert.NoError(t, err)
	defer output.Stop(ctx)

	// Test with a valid time.Time object
	now := time.Now()
	err = output.Sink(ctx, types.ExtractedLog{"my_timestamp": now})
	assert.NoError(t, err)

	select {
	case reqBody := <-mockServer.requests:
		var log map[string]interface{}
		err := json.Unmarshal(reqBody, &log)
		assert.NoError(t, err)
		// The default JSON marshaling for time.Time is RFC3339
		parsedTime, err := time.Parse(time.RFC3339, log["my_timestamp"].(string))
		assert.NoError(t, err)
		assert.WithinDuration(t, now, parsedTime, time.Second)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for request")
	}
}

func TestVMLogsOutput_Sink_StreamFields(t *testing.T) {
	mockServer := newMockVMLogsServer()
	defer mockServer.Close()

	// Create a mock logger to capture warnings
	var loggedWarnings []string
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(io.Discard), // Discard output to avoid polluting test logs
		zap.WarnLevel,
	)
	logger := zap.New(core)
	// Redirect logger output to capture warnings
	logger = logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		if entry.Level == zap.WarnLevel {
			loggedWarnings = append(loggedWarnings, entry.Message)
		}
		return nil
	}))

	output := &VMLogsOutput{
		Endpoints:          mockServer.endpoints,
		BatchActions:       1,
		BatchFlushInterval: 1 * time.Second,
		StreamFields:       []string{"field1", "field2"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := output.Init(ctx, 1)
	assert.NoError(t, err)
	err = output.Start(ctx, logger) // Pass the mock logger
	assert.NoError(t, err)
	defer output.Stop(ctx)

	t.Run("missing stream field should log warning", func(t *testing.T) {
		loggedWarnings = []string{}                                     // Reset warnings
		err := output.Sink(ctx, types.ExtractedLog{"field1": "value1"}) // field2 is missing
		assert.NoError(t, err)

		// Wait for the processor to potentially process the log and trigger the warning
		time.Sleep(100 * time.Millisecond)

		assert.Contains(t, loggedWarnings, "The stream_field is not exist")
	})

	t.Run("all stream fields present should not log warning", func(t *testing.T) {
		loggedWarnings = []string{} // Reset warnings
		err := output.Sink(ctx, types.ExtractedLog{"field1": "value1", "field2": "value2"})
		assert.NoError(t, err)

		// Wait for the processor to potentially process the log
		time.Sleep(100 * time.Millisecond)

		assert.Empty(t, loggedWarnings, "no warnings should be logged when all stream fields are present")
	})
}

func TestCommitLogic_Direct(t *testing.T) {
	mockServer := newMockVMLogsServer()
	defer mockServer.Close()

	output := &VMLogsOutput{
		Endpoints: mockServer.endpoints,
	}
	// We need a client, so we must call Init
	err := output.Init(context.Background(), 1)
	require.NoError(t, err)

	// We need a processor to call the commit method
	processor := NewBatchProcessor(output)

	// 1. 手动创建一个包含2条日志的批次
	reqs := make([]cachedLog, 2)
	reqs[0] = cachedLog{log: types.ExtractedLog{"message": "direct_log_1"}}
	reqs[1] = cachedLog{log: types.ExtractedLog{"message": "direct_log_2"}}

	// 2. 手动将这2条日志序列化
	for i := range reqs {
		bytes, err := json.Marshal(reqs[i].log)
		require.NoError(t, err)
		reqs[i].bytes = bytes
		fmt.Printf("[DIRECT TEST] Manually marshalled log %d, len %d\n", i, len(reqs[i].bytes))
	}

	// 3. 直接调用 commit 函数
	fmt.Println("[DIRECT TEST] Calling commit directly...")
	err = processor.commit(context.Background(), reqs)
	require.NoError(t, err)

	// 4. 检查 mock server 是否收到了正确的数据
	select {
	case reqBody := <-mockServer.requests:
		fmt.Printf("[DIRECT TEST] Mock server received body:\n%s\n", string(reqBody))
		assert.Contains(t, string(reqBody), "direct_log_1")
		assert.Contains(t, string(reqBody), "direct_log_2")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for direct commit")
	}
}

func TestVMLogsOutput_Integration(t *testing.T) {
	// This is an integration test and requires a running VMLogs instance.
	// It is skipped by default. To run it, comment out the t.Skip() line
	// and ensure the credentials and endpoint are correct.
	//t.Skip("Skipping integration test")

	endpoint := "http://10.99.1.15:9428/"
	username := "root"
	password := "root.2020"

	output := &VMLogsOutput{
		Endpoints:          []string{endpoint},
		Username:           username,
		Password:           password,
		BatchActions:       3,
		BatchFlushInterval: 1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a real logger for integration testing to see output
	logger, _ := zap.NewDevelopment()
	err := output.Init(ctx, 1)
	require.NoError(t, err)

	err = output.Start(ctx, logger)
	require.NoError(t, err)
	defer output.Stop(ctx)

	t.Log("Sending 3 logs to VMLogs instance...")
	for i := 1; i <= 3; i++ {
		log := types.ExtractedLog{
			"integration_test_marker": "true",
			"message":                 fmt.Sprintf("Hello VMLogs integration test %d", i),
			"index":                   i,
			"timestamp":               time.Now().Format(time.RFC3339Nano),
		}
		err := output.Sink(ctx, log)
		assert.NoError(t, err)
	}

	t.Log("Waiting for 5 seconds for logs to be flushed...")
	time.Sleep(5 * time.Second)

	t.Log("Integration test finished.")
	//t.Logf("Please check your VMLogs instance at %s for logs with the label {integration_test_marker="true"}", endpoint)
}
