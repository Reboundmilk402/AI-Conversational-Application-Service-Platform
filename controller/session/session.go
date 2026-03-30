package session

import (
	"GopherAI/common/code"
	"GopherAI/controller"
	"GopherAI/model"
	sessionservice "GopherAI/service/session"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	GetUserSessionsResponse struct {
		controller.Response
		Sessions []model.SessionInfo `json:"sessions,omitempty"`
	}

	CreateSessionAndSendMessageRequest struct {
		UserQuestion    string `json:"question" binding:"required"`
		ModelType       string `json:"modelType" binding:"required"`
		EnableWebSearch bool   `json:"enableWebSearch,omitempty"`
	}

	CreateSessionAndSendMessageResponse struct {
		AiInformation string `json:"Information,omitempty"`
		SessionID     string `json:"sessionId,omitempty"`
		controller.Response
	}

	ChatSendRequest struct {
		UserQuestion    string `json:"question" binding:"required"`
		ModelType       string `json:"modelType" binding:"required"`
		SessionID       string `json:"sessionId,omitempty" binding:"required"`
		EnableWebSearch bool   `json:"enableWebSearch,omitempty"`
	}

	ChatSendResponse struct {
		AiInformation string `json:"Information,omitempty"`
		controller.Response
	}

	ChatHistoryRequest struct {
		SessionID string `json:"sessionId,omitempty" binding:"required"`
	}

	ChatHistoryResponse struct {
		History []model.History `json:"history"`
		controller.Response
	}

	DeleteSessionRequest struct {
		SessionID string `json:"sessionId" binding:"required"`
	}

	DeleteSessionResponse struct {
		controller.Response
	}
)

func GetUserSessionsByUserName(c *gin.Context) {
	res := new(GetUserSessionsResponse)
	userName := c.GetString("userName")

	userSessions, err := sessionservice.GetUserSessionsByUserName(userName)
	if err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.Sessions = userSessions
	c.JSON(http.StatusOK, res)
}

func CreateSessionAndSendMessage(c *gin.Context) {
	req := new(CreateSessionAndSendMessageRequest)
	res := new(CreateSessionAndSendMessageResponse)
	userName := c.GetString("userName")
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	sessionID, aiInformation, code_ := sessionservice.CreateSessionAndSendMessage(
		userName,
		req.UserQuestion,
		req.ModelType,
		req.EnableWebSearch,
	)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.AiInformation = aiInformation
	res.SessionID = sessionID
	c.JSON(http.StatusOK, res)
}

func CreateStreamSessionAndSendMessage(c *gin.Context) {
	req := new(CreateSessionAndSendMessageRequest)
	userName := c.GetString("userName")
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "Invalid parameters"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no")

	sessionID, code_ := sessionservice.CreateStreamSessionOnly(userName, req.UserQuestion, req.ModelType)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to create session"})
		return
	}

	c.Writer.WriteString(fmt.Sprintf("data: {\"sessionId\": \"%s\"}\n\n", sessionID))
	c.Writer.Flush()

	code_ = sessionservice.StreamMessageToExistingSession(
		userName,
		sessionID,
		req.UserQuestion,
		req.ModelType,
		req.EnableWebSearch,
		http.ResponseWriter(c.Writer),
	)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to send message"})
		return
	}
}

func ChatSend(c *gin.Context) {
	req := new(ChatSendRequest)
	res := new(ChatSendResponse)
	userName := c.GetString("userName")
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	aiInformation, code_ := sessionservice.ChatSend(
		userName,
		req.SessionID,
		req.UserQuestion,
		req.ModelType,
		req.EnableWebSearch,
	)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.AiInformation = aiInformation
	c.JSON(http.StatusOK, res)
}

func ChatStreamSend(c *gin.Context) {
	req := new(ChatSendRequest)
	userName := c.GetString("userName")
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "Invalid parameters"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no")

	code_ := sessionservice.ChatStreamSend(
		userName,
		req.SessionID,
		req.UserQuestion,
		req.ModelType,
		req.EnableWebSearch,
		http.ResponseWriter(c.Writer),
	)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to send message"})
		return
	}
}

func ChatHistory(c *gin.Context) {
	req := new(ChatHistoryRequest)
	res := new(ChatHistoryResponse)
	userName := c.GetString("userName")
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	history, code_ := sessionservice.GetChatHistory(userName, req.SessionID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.History = history
	c.JSON(http.StatusOK, res)
}

func DeleteSession(c *gin.Context) {
	req := new(DeleteSessionRequest)
	res := new(DeleteSessionResponse)
	userName := c.GetString("userName")
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	code_ := sessionservice.DeleteSession(userName, req.SessionID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	c.JSON(http.StatusOK, res)
}
