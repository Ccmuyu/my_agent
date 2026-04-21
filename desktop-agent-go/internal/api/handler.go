package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"desktop-agent/internal/agent"
)

var desktopAgent *agent.DesktopAgent

func Run(a *agent.DesktopAgent, host string, port int) {
	desktopAgent = a

	r := gin.Default()

	r.POST("/api/tasks", createTask)
	r.GET("/api/tasks/:id", getTask)
	r.POST("/api/tasks/:id/confirm", confirmTask)
	r.GET("/api/tools", listTools)
	r.GET("/api/screen", screenCapture)
	r.GET("/api/screen/ocr", screenOCR)
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
	c.JSON(http.StatusOK, gin.H{"message": "TODO"})
}

func screenOCR(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"text": "TODO"})
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}