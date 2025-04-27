package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aiagent/pkg/rag"
	"github.com/aiagent/pkg/sql"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

type RagMessage struct {
	User      string `json:"user,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Operate   string `json:"operate,omitempty"`
}

func RagHandler(w http.ResponseWriter, r *http.Request, rdb *redis.Client, db *pgxpool.Pool, embedder *embeddings.EmbedderImpl, llm *openai.LLM) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error while upgrading connection: %s", err)
	}
	ctx := context.Background()
	err = conn.WriteMessage(websocket.TextMessage, []byte("连接成功"))
	if err != nil {
		fmt.Printf("Error while writing message: %s", err)
	}

	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Error while reading message: %s", err)
			break
		}
		var ragMessage RagMessage
		err = json.Unmarshal(msg, &ragMessage)
		if err != nil {
			fmt.Printf("Error while unmarshalling message: %s", err)
			break
		}

		switch ragMessage.Operate {
		case "addDoc":
			err = rag.InsertDocument(ctx, db, ragMessage.Content, embedder)
			if err != nil {
				fmt.Printf("Error while inserting document: %s", err)
				err = conn.WriteMessage(websocket.TextMessage, []byte("插入失败"))
				if err != nil {
					fmt.Printf("Error while writing message: %s", err)
				}
			}
			err = conn.WriteMessage(websocket.TextMessage, []byte("插入成功"))
			if err != nil {
				fmt.Printf("Error while writing message: %s", err)
			}
		case "scanDoc":
			result, err := rag.ScanDocuments(ctx, db)
			if err != nil {
				fmt.Printf("Error while scanning documents: %s", err)
			}
			var response string
			for _, doc := range result {
				response += doc + "\n"
			}
			err = conn.WriteMessage(websocket.TextMessage, []byte(response))
			if err != nil {
				fmt.Printf("Error while writing message: %s", err)
			}
		case "createMemory":
			var request string
			request = "总结下面的对话内容，并生成一段记忆内容。对象是" + ragMessage.User + "\n\n对话内容：\n"
			result, err := sql.GetChatMessage(ctx, rdb, ragMessage.SessionID, ragMessage.User)
			if err != nil {
				fmt.Printf("Error while getting chat message: %s", err)
				conn.WriteMessage(websocket.TextMessage, []byte("获取失败"))
				break
			}
			for _, message := range result {
				request += message + "\n"
			}
			response, err := llm.Call(ctx, request)
			if err != nil {
				fmt.Printf("Error while generating content: %s", err)
			}
			rag.InsertMemory(ctx, db, response, embedder)
			conn.WriteMessage(websocket.TextMessage, []byte("总结的记忆为"+response))
		case "scanMemory":
			result, err := rag.ScanMemory(ctx, db)
			if err != nil {
				fmt.Printf("Error while scanning memory: %s", err)
			}
			var response string
			for _, doc := range result {
				response += doc + "\n"
			}
			err = conn.WriteMessage(websocket.TextMessage, []byte(response))
			if err != nil {
				fmt.Printf("Error while writing message: %s", err)
			}
		case "scanChat":
			result, err := sql.GetAllChatMessionID(ctx, rdb, ragMessage.User)
			if err != nil {
				fmt.Printf("Error while scanning chat: %s", err)
			}
			for _, sessionID := range result {
				err = conn.WriteMessage(websocket.TextMessage, []byte(sessionID))
				if err != nil {
					fmt.Printf("Error while writing message: %s", err)
				}
			}
		case "viewChat":
			result, err := sql.GetChatMessage(ctx, rdb, ragMessage.SessionID, ragMessage.User)
			if err != nil {
				fmt.Printf("Error while getting chat message: %s", err)
			}
			for _, message := range result {
				var msg sql.Message
				err = json.Unmarshal([]byte(message), &msg)
				if err != nil {
					fmt.Printf("Error while unmarshalling message: %s", err)
				}
				response := msg.Role + ": " + msg.Content
				err = conn.WriteMessage(websocket.TextMessage, []byte(response))
				if err != nil {
					fmt.Printf("Error while writing message: %s", err)
				}
			}
		}

	}
}
