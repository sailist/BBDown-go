package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nilaonai/bbdown-go/internal/config"
)

func setupTestServer() *Server {
	cfg := config.NewConfig()
	return NewServer(cfg)
}

func TestAddTask(t *testing.T) {
	s := setupTestServer()
	body := `{"url":"https://www.bilibili.com/video/BV1xx411c7mD"}`
	req, _ := http.NewRequest("POST", "/add-task", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["status"] != "added" {
		t.Errorf("expected status 'added', got %v", resp["status"])
	}

	// Verify task appears in running tasks.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/get-tasks/running", nil)
	s.engine.ServeHTTP(w, req)

	var running []*DownloadTask
	if err := json.Unmarshal(w.Body.Bytes(), &running); err != nil {
		t.Fatalf("failed to unmarshal running tasks: %v", err)
	}
	if len(running) != 1 {
		t.Errorf("expected 1 running task, got %d", len(running))
	}
}

func TestAddTaskDuplicate(t *testing.T) {
	s := setupTestServer()
	body := `{"url":"https://www.bilibili.com/video/BV1xx411c7mD"}`

	req, _ := http.NewRequest("POST", "/add-task", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)

	// Add same task again.
	req, _ = http.NewRequest("POST", "/add-task", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["status"] != "already running" {
		t.Errorf("expected status 'already running', got %v", resp["status"])
	}
}

func TestAddTaskInvalidRequest(t *testing.T) {
	s := setupTestServer()
	body := `{"invalid":"json"}`
	req, _ := http.NewRequest("POST", "/add-task", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetTasks(t *testing.T) {
	s := setupTestServer()

	// Add a task.
	body := `{"url":"https://www.bilibili.com/video/BV1xx411c7mD"}`
	req, _ := http.NewRequest("POST", "/add-task", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)

	// Get all tasks.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/get-tasks/", nil)
	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp tasksResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Running) != 1 {
		t.Errorf("expected 1 running task, got %d", len(resp.Running))
	}
	if len(resp.Finished) != 0 {
		t.Errorf("expected 0 finished tasks, got %d", len(resp.Finished))
	}
}

func TestGetTaskByID(t *testing.T) {
	s := setupTestServer()

	// Add a task.
	body := `{"url":"https://www.bilibili.com/video/BV1xx411c7mD"}`
	req, _ := http.NewRequest("POST", "/add-task", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)

	var addResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &addResp)
	aid := addResp["aid"].(string)

	// Get by ID.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/get-tasks/"+aid, nil)
	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var task DownloadTask
	if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
		t.Fatalf("failed to unmarshal task: %v", err)
	}
	if task.Aid != aid {
		t.Errorf("expected aid %s, got %s", aid, task.Aid)
	}

	// Get non-existent ID.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/get-tasks/nonexistent", nil)
	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestTaskCompletionFlow(t *testing.T) {
	s := setupTestServer()

	body := `{"url":"https://www.bilibili.com/video/BV1xx411c7mD"}`
	req, _ := http.NewRequest("POST", "/add-task", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)

	var addResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &addResp)
	aid := addResp["aid"].(string)

	// Poll for task to move from running to finished.
	var finished []*DownloadTask
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/get-tasks/finished", nil)
		s.engine.ServeHTTP(w, req)
		json.Unmarshal(w.Body.Bytes(), &finished)
		if len(finished) > 0 {
			break
		}
	}

	if len(finished) != 1 {
		t.Fatalf("expected 1 finished task after polling, got %d", len(finished))
	}
	if finished[0].Aid != aid {
		t.Errorf("expected finished aid %s, got %s", aid, finished[0].Aid)
	}
	if !finished[0].IsSuccessful {
		t.Error("expected task to be successful")
	}
	if finished[0].Progress != 1.0 {
		t.Errorf("expected progress 1.0, got %f", finished[0].Progress)
	}
}

func TestRemoveFinished(t *testing.T) {
	s := setupTestServer()

	// Manually inject a finished task.
	s.mu.Lock()
	s.finishedTasks["task1"] = &DownloadTask{
		Aid:            "task1",
		URL:            "http://example.com/1",
		IsSuccessful:   true,
		TaskCreateTime: time.Now().Unix(),
	}
	s.finishedTasks["task2"] = &DownloadTask{
		Aid:            "task2",
		URL:            "http://example.com/2",
		IsSuccessful:   false,
		TaskCreateTime: time.Now().Unix(),
	}
	s.mu.Unlock()

	// Remove by ID.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/remove-finished/task1", nil)
	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/get-tasks/finished", nil)
	s.engine.ServeHTTP(w, req)

	var finished []*DownloadTask
	if err := json.Unmarshal(w.Body.Bytes(), &finished); err != nil {
		t.Fatalf("failed to unmarshal finished tasks: %v", err)
	}
	if len(finished) != 1 {
		t.Errorf("expected 1 finished task after remove-by-id, got %d", len(finished))
	}

	// Remove all.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/remove-finished/", nil)
	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/get-tasks/finished", nil)
	s.engine.ServeHTTP(w, req)

	finished = nil
	if err := json.Unmarshal(w.Body.Bytes(), &finished); err != nil {
		t.Fatalf("failed to unmarshal finished tasks: %v", err)
	}
	if len(finished) != 0 {
		t.Errorf("expected 0 finished tasks after remove-all, got %d", len(finished))
	}
}

func TestRemoveFailedFinished(t *testing.T) {
	s := setupTestServer()

	s.mu.Lock()
	s.finishedTasks["success1"] = &DownloadTask{
		Aid:          "success1",
		URL:          "http://example.com/s1",
		IsSuccessful: true,
	}
	s.finishedTasks["fail1"] = &DownloadTask{
		Aid:          "fail1",
		URL:          "http://example.com/f1",
		IsSuccessful: false,
	}
	s.mu.Unlock()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/remove-finished/failed", nil)
	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/get-tasks/finished", nil)
	s.engine.ServeHTTP(w, req)

	var finished []*DownloadTask
	if err := json.Unmarshal(w.Body.Bytes(), &finished); err != nil {
		t.Fatalf("failed to unmarshal finished tasks: %v", err)
	}
	if len(finished) != 1 {
		t.Errorf("expected 1 finished task after remove-failed, got %d", len(finished))
	}
	if finished[0].Aid != "success1" {
		t.Errorf("expected remaining task to be success1, got %s", finished[0].Aid)
	}
}

func TestCORS(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/get-tasks/", nil)
	s.engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for OPTIONS, got %d", w.Code)
	}
	if h := w.Header().Get("Access-Control-Allow-Origin"); h != "*" {
		t.Errorf("expected CORS origin *, got %s", h)
	}
	if h := w.Header().Get("Access-Control-Allow-Methods"); h != "*" {
		t.Errorf("expected CORS methods *, got %s", h)
	}
	if h := w.Header().Get("Access-Control-Allow-Headers"); h != "*" {
		t.Errorf("expected CORS headers *, got %s", h)
	}
}
