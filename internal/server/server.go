package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/pprof"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nilaonai/bbdown-go/internal/config"
)

// DownloadTask represents a single download job.
type DownloadTask struct {
	Aid                  string   `json:"aid"`
	URL                  string   `json:"url"`
	Title                string   `json:"title,omitempty"`
	Pic                  string   `json:"pic,omitempty"`
	VideoPubTime         int64    `json:"videoPubTime,omitempty"`
	TaskCreateTime       int64    `json:"taskCreateTime"`
	TaskFinishTime       int64    `json:"taskFinishTime,omitempty"`
	Progress             float64  `json:"progress"`
	DownloadSpeed        float64  `json:"downloadSpeed"`
	TotalDownloadedBytes float64  `json:"totalDownloadedBytes"`
	IsSuccessful         bool     `json:"isSuccessful"`
	SavePaths            []string `json:"savePaths,omitempty"`
}

// Server is the BBDown API server.
type Server struct {
	engine        *gin.Engine
	runningTasks  map[string]*DownloadTask
	finishedTasks map[string]*DownloadTask
	mu            sync.RWMutex
	cfg           *config.Config
}

// NewServer creates a new BBDown API server.
func NewServer(cfg *config.Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		engine:        gin.New(),
		runningTasks:  make(map[string]*DownloadTask),
		finishedTasks: make(map[string]*DownloadTask),
		cfg:           cfg,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.engine.Use(gin.Recovery())
	s.engine.Use(corsMiddleware())

	api := s.engine.Group("/get-tasks")
	api.GET("/", s.getTasks)
	api.GET("/running", s.getRunningTasks)
	api.GET("/finished", s.getFinishedTasks)
	api.GET("/:id", s.getTaskByID)

	s.engine.POST("/add-task", s.addTask)

	remove := s.engine.Group("/remove-finished")
	remove.GET("/", s.removeAllFinished)
	remove.GET("/failed", s.removeFailedFinished)
	remove.GET("/:id", s.removeFinishedByID)

	// Register pprof debug endpoints when BBDOWN_PPROF=1.
	if os.Getenv("BBDOWN_PPROF") == "1" {
		s.engine.GET("/debug/pprof/", gin.WrapF(pprof.Index))
		s.engine.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
		s.engine.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
		s.engine.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
		s.engine.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))
		s.engine.GET("/debug/pprof/:name", gin.WrapH(pprof.Handler("")))
	}
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(listenAddr string) error {
	return s.engine.Run(listenAddr)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

type tasksResponse struct {
	Running  []*DownloadTask `json:"running"`
	Finished []*DownloadTask `json:"finished"`
}

func (s *Server) getTasks(c *gin.Context) {
	s.mu.RLock()
	running := make([]*DownloadTask, 0, len(s.runningTasks))
	for _, t := range s.runningTasks {
		running = append(running, cloneTask(t))
	}
	finished := make([]*DownloadTask, 0, len(s.finishedTasks))
	for _, t := range s.finishedTasks {
		finished = append(finished, cloneTask(t))
	}
	s.mu.RUnlock()

	c.JSON(http.StatusOK, tasksResponse{Running: running, Finished: finished})
}

func (s *Server) getRunningTasks(c *gin.Context) {
	s.mu.RLock()
	tasks := make([]*DownloadTask, 0, len(s.runningTasks))
	for _, t := range s.runningTasks {
		tasks = append(tasks, cloneTask(t))
	}
	s.mu.RUnlock()
	c.JSON(http.StatusOK, tasks)
}

func (s *Server) getFinishedTasks(c *gin.Context) {
	s.mu.RLock()
	tasks := make([]*DownloadTask, 0, len(s.finishedTasks))
	for _, t := range s.finishedTasks {
		tasks = append(tasks, cloneTask(t))
	}
	s.mu.RUnlock()
	c.JSON(http.StatusOK, tasks)
}

func (s *Server) getTaskByID(c *gin.Context) {
	id := c.Param("id")
	s.mu.RLock()
	task, ok := s.runningTasks[id]
	if !ok {
		task, ok = s.finishedTasks[id]
	}
	var t *DownloadTask
	if ok {
		t = cloneTask(task)
	}
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func cloneTask(src *DownloadTask) *DownloadTask {
	return &DownloadTask{
		Aid:                  src.Aid,
		URL:                  src.URL,
		Title:                src.Title,
		Pic:                  src.Pic,
		VideoPubTime:         src.VideoPubTime,
		TaskCreateTime:       src.TaskCreateTime,
		TaskFinishTime:       src.TaskFinishTime,
		Progress:             src.Progress,
		DownloadSpeed:        src.DownloadSpeed,
		TotalDownloadedBytes: src.TotalDownloadedBytes,
		IsSuccessful:         src.IsSuccessful,
		SavePaths:            append([]string(nil), src.SavePaths...),
	}
}

type addTaskRequest struct {
	URL string `json:"url"`
}

func (s *Server) addTask(c *gin.Context) {
	var req addTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: url is required"})
		return
	}

	aid := deriveAid(req.URL)

	s.mu.Lock()
	if _, exists := s.runningTasks[aid]; exists {
		s.mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"aid": aid, "status": "already running"})
		return
	}
	if _, exists := s.finishedTasks[aid]; exists {
		delete(s.finishedTasks, aid)
	}

	task := &DownloadTask{
		Aid:            aid,
		URL:            req.URL,
		TaskCreateTime: time.Now().Unix(),
	}
	s.runningTasks[aid] = task
	s.mu.Unlock()

	go s.runDownload(task)

	c.JSON(http.StatusOK, gin.H{"aid": aid, "status": "added"})
}

func deriveAid(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:8])
}

func (s *Server) runDownload(task *DownloadTask) {
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		s.mu.Lock()
		task.Progress = float64(i+1) / 10.0
		task.TotalDownloadedBytes += 1024 * 1024
		s.mu.Unlock()
	}

	s.mu.Lock()
	task.Progress = 1.0
	task.IsSuccessful = true
	task.TaskFinishTime = time.Now().Unix()
	if task.TaskFinishTime > task.TaskCreateTime {
		task.DownloadSpeed = task.TotalDownloadedBytes / float64(task.TaskFinishTime-task.TaskCreateTime)
	}
	delete(s.runningTasks, task.Aid)
	s.finishedTasks[task.Aid] = task
	s.mu.Unlock()
}

func (s *Server) removeAllFinished(c *gin.Context) {
	s.mu.Lock()
	s.finishedTasks = make(map[string]*DownloadTask)
	s.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) removeFailedFinished(c *gin.Context) {
	s.mu.Lock()
	for id, t := range s.finishedTasks {
		if !t.IsSuccessful {
			delete(s.finishedTasks, id)
		}
	}
	s.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) removeFinishedByID(c *gin.Context) {
	id := c.Param("id")
	s.mu.Lock()
	delete(s.finishedTasks, id)
	s.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
