package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Ccmuyu/my_agent/internal/agent"
)

var desktopAgent *agent.DesktopAgent

func Run(a *agent.DesktopAgent, host string, port int) {
	desktopAgent = a

	r := gin.Default()

	r.POST("/api/tasks", createTask)
	r.GET("/api/tasks", listTasks)
	r.GET("/api/tasks/:id", getTask)
	r.POST("/api/tasks/:id/confirm", confirmTask)
	r.POST("/api/tasks/:id/cancel", cancelTask)
	r.GET("/api/tools", listTools)
	r.GET("/api/screen", screenCapture)
	r.GET("/api/screen/ocr", screenOCR)
	r.GET("/api/history", getHistory)
	r.GET("/health", health)

	addr := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("启动Web服务: http://%s\n", addr)
	r.Run(addr)
}

type TaskRequest struct {
	Intent   string `json:"intent" binding:"required"`
	Confirm bool   `json:"confirm"`
}

type TaskResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func createTask(c *gin.Context) {
	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := desktopAgent.CreateTask(req.Intent, req.Confirm)
	c.JSON(http.StatusOK, TaskResponse{
		TaskID:  task.ID,
		Status:  string(task.Status),
		Message: "任务已创建",
	})
}

func getTask(c *gin.Context) {
	id := c.Param("id")
	task, ok := desktopAgent.GetTask(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task.ToMap())
}

type ConfirmRequest struct {
	Confirmed bool `json:"confirmed"`
}

func confirmTask(c *gin.Context) {
	id := c.Param("id")

	var req ConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Confirmed {
		task, err := desktopAgent.ConfirmTask(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, task.ToMap())
	} else {
		task, ok := desktopAgent.GetTask(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusOK, task.ToMap())
	}
}

func listTools(c *gin.Context) {
	tools := desktopAgent.GetTools()
	c.JSON(http.StatusOK, tools)
}

func screenCapture(c *gin.Context) {
	dir := "/tmp/github.com/Ccmuyu/my_agent/screenshots"
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("screen_%s.png", time.Now().Format("20060102_150405"))
	path := fmt.Sprintf("%s/%s", dir, filename)

	var cmd *exec.Cmd
	cmd = exec.Command("gnome-screenshot", "-f", path)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("scrot", path)
		if err := cmd.Run(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "screenshot failed"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"path": path})
}

func screenOCR(c *gin.Context) {
	dir := "/tmp/github.com/Ccmuyu/my_agent/screenshots"
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("ocr_%d.png", time.Now().Unix())
	path := fmt.Sprintf("%s/%s", dir, filename)

	cmd := exec.Command("gnome-screenshot", "-f", path)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("scrot", path)
		if err := cmd.Run(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "screenshot failed"})
			return
		}
	}

	ocrCmd := exec.Command("tesseract", path, "-")
	output, err := ocrCmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ocr failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"text": string(output)})
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func listTasks(c *gin.Context) {
	tasks := desktopAgent.ListTasks()
	result := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, t.ToMap())
	}
	c.JSON(http.StatusOK, result)
}

func cancelTask(c *gin.Context) {
	id := c.Param("id")
	task, ok := desktopAgent.GetTask(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	if err := desktopAgent.CancelTask(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task.ToMap())
}

func getHistory(c *gin.Context) {
	history := desktopAgent.GetToolHistory()
	c.JSON(http.StatusOK, history)
}