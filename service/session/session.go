package session

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/code"
	"GopherAI/dao/message"
	"GopherAI/dao/session"
	"GopherAI/model"
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ctx = context.Background()

func GetUserSessionsByUserName(userName string) ([]model.SessionInfo, error) {
	sessionRecords, err := session.GetSessionsByUserName(userName)
	if err != nil {
		return nil, err
	}

	sessionInfos := make([]model.SessionInfo, 0, len(sessionRecords))
	for _, sessionRecord := range sessionRecords {
		sessionInfos = append(sessionInfos, model.SessionInfo{
			SessionID: sessionRecord.ID,
			Title:     sessionRecord.Title,
			ModelType: normalizeModelType(sessionRecord.ModelType),
		})
	}

	return sessionInfos, nil
}

func normalizeModelType(modelType string) string {
	if modelType == "" {
		return "1"
	}
	return modelType
}

func loadSessionHelper(userName string, sessionID string, modelType string) (*aihelper.AIHelper, code.Code) {
	sessionRecord, err := session.GetSessionByID(sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, code.CodeRecordNotFound
		}
		log.Println("loadSessionHelper GetSessionByID error:", err)
		return nil, code.CodeServerBusy
	}

	if sessionRecord.UserName != userName {
		return nil, code.CodeForbidden
	}

	resolvedModelType := normalizeModelType(sessionRecord.ModelType)
	if resolvedModelType == "1" && modelType != "" {
		resolvedModelType = normalizeModelType(modelType)
	}

	manager := aihelper.GetGlobalManager()
	helper, exists := manager.GetAIHelper(userName, sessionID)
	if !exists {
		helper, err = manager.GetOrCreateAIHelper(
			userName,
			sessionID,
			resolvedModelType,
			aihelper.BuildModelConfig(resolvedModelType),
		)
		if err != nil {
			log.Println("loadSessionHelper GetOrCreateAIHelper error:", err)
			return nil, code.AIModelFail
		}
	}

	if len(helper.GetMessages()) == 0 {
		dbMessages, err := message.GetMessagesBySessionID(sessionID)
		if err != nil {
			log.Println("loadSessionHelper GetMessagesBySessionID error:", err)
			return nil, code.CodeServerBusy
		}

		for _, msg := range dbMessages {
			helper.AddMessage(msg.Content, msg.UserName, msg.IsUser, false)
		}
	}

	return helper, code.CodeSuccess
}

func CreateSessionAndSendMessage(userName string, userQuestion string, modelType string, enableWebSearch bool) (string, string, code.Code) {
	modelType = normalizeModelType(modelType)

	newSession := &model.Session{
		ID:        uuid.New().String(),
		UserName:  userName,
		Title:     userQuestion,
		ModelType: modelType,
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateSessionAndSendMessage CreateSession error:", err)
		return "", "", code.CodeServerBusy
	}

	manager := aihelper.GetGlobalManager()
	helper, err := manager.GetOrCreateAIHelper(
		userName,
		createdSession.ID,
		modelType,
		aihelper.BuildModelConfig(modelType),
	)
	if err != nil {
		log.Println("CreateSessionAndSendMessage GetOrCreateAIHelper error:", err)
		return "", "", code.AIModelFail
	}

	aiResponse, err := helper.GenerateResponse(userName, ctx, userQuestion, aihelper.ChatOptions{
		EnableWebSearch: enableWebSearch,
	})
	if err != nil {
		log.Println("CreateSessionAndSendMessage GenerateResponse error:", err)
		return "", "", code.AIModelFail
	}

	return createdSession.ID, aiResponse.Content, code.CodeSuccess
}

func CreateStreamSessionOnly(userName string, userQuestion string, modelType string) (string, code.Code) {
	modelType = normalizeModelType(modelType)

	newSession := &model.Session{
		ID:        uuid.New().String(),
		UserName:  userName,
		Title:     userQuestion,
		ModelType: modelType,
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateStreamSessionOnly CreateSession error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

func StreamMessageToExistingSession(userName string, sessionID string, userQuestion string, modelType string, enableWebSearch bool, writer http.ResponseWriter) code.Code {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		log.Println("StreamMessageToExistingSession: streaming unsupported")
		return code.CodeServerBusy
	}

	helper, code_ := loadSessionHelper(userName, sessionID, modelType)
	if code_ != code.CodeSuccess {
		return code_
	}

	cb := func(msg string) {
		log.Printf("[SSE] Sending chunk: %s (len=%d)\n", msg, len(msg))
		if _, err := writer.Write([]byte("data: " + msg + "\n\n")); err != nil {
			log.Println("[SSE] Write error:", err)
			return
		}
		flusher.Flush()
	}

	if _, err := helper.StreamResponse(userName, ctx, cb, userQuestion, aihelper.ChatOptions{
		EnableWebSearch: enableWebSearch,
	}); err != nil {
		log.Println("StreamMessageToExistingSession StreamResponse error:", err)
		return code.AIModelFail
	}

	if _, err := writer.Write([]byte("data: [DONE]\n\n")); err != nil {
		log.Println("StreamMessageToExistingSession write DONE error:", err)
		return code.AIModelFail
	}
	flusher.Flush()

	return code.CodeSuccess
}

func CreateStreamSessionAndSendMessage(userName string, userQuestion string, modelType string, enableWebSearch bool, writer http.ResponseWriter) (string, code.Code) {
	sessionID, code_ := CreateStreamSessionOnly(userName, userQuestion, modelType)
	if code_ != code.CodeSuccess {
		return "", code_
	}

	code_ = StreamMessageToExistingSession(userName, sessionID, userQuestion, modelType, enableWebSearch, writer)
	if code_ != code.CodeSuccess {
		return sessionID, code_
	}

	return sessionID, code.CodeSuccess
}

func ChatSend(userName string, sessionID string, userQuestion string, modelType string, enableWebSearch bool) (string, code.Code) {
	helper, code_ := loadSessionHelper(userName, sessionID, modelType)
	if code_ != code.CodeSuccess {
		return "", code_
	}

	aiResponse, err := helper.GenerateResponse(userName, ctx, userQuestion, aihelper.ChatOptions{
		EnableWebSearch: enableWebSearch,
	})
	if err != nil {
		log.Println("ChatSend GenerateResponse error:", err)
		return "", code.AIModelFail
	}

	return aiResponse.Content, code.CodeSuccess
}

func GetChatHistory(userName string, sessionID string) ([]model.History, code.Code) {
	sessionRecord, err := session.GetSessionByID(sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, code.CodeRecordNotFound
		}
		log.Println("GetChatHistory GetSessionByID error:", err)
		return nil, code.CodeServerBusy
	}

	if sessionRecord.UserName != userName {
		return nil, code.CodeForbidden
	}

	manager := aihelper.GetGlobalManager()
	if helper, exists := manager.GetAIHelper(userName, sessionID); exists {
		messages := helper.GetMessages()
		history := make([]model.History, 0, len(messages))
		for _, msg := range messages {
			history = append(history, model.History{
				IsUser:  msg.IsUser,
				Content: msg.Content,
			})
		}
		return history, code.CodeSuccess
	}

	dbMessages, err := message.GetMessagesBySessionID(sessionID)
	if err != nil {
		log.Println("GetChatHistory GetMessagesBySessionID error:", err)
		return nil, code.CodeServerBusy
	}

	history := make([]model.History, 0, len(dbMessages))
	for _, msg := range dbMessages {
		history = append(history, model.History{
			IsUser:  msg.IsUser,
			Content: msg.Content,
		})
	}

	return history, code.CodeSuccess
}

func ChatStreamSend(userName string, sessionID string, userQuestion string, modelType string, enableWebSearch bool, writer http.ResponseWriter) code.Code {
	return StreamMessageToExistingSession(userName, sessionID, userQuestion, modelType, enableWebSearch, writer)
}

func DeleteSession(userName string, sessionID string) code.Code {
	sessionRecord, err := session.GetSessionByID(sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return code.CodeRecordNotFound
		}
		log.Println("DeleteSession GetSessionByID error:", err)
		return code.CodeServerBusy
	}

	if sessionRecord.UserName != userName {
		return code.CodeForbidden
	}

	if err := message.DeleteMessagesBySessionID(sessionID); err != nil {
		log.Println("DeleteSession DeleteMessagesBySessionID error:", err)
		return code.CodeServerBusy
	}

	if err := session.DeleteSessionByID(sessionID); err != nil {
		log.Println("DeleteSession DeleteSessionByID error:", err)
		return code.CodeServerBusy
	}

	aihelper.GetGlobalManager().RemoveAIHelper(userName, sessionID)
	return code.CodeSuccess
}
