package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aiagent/pkg/base"
	"github.com/aiagent/pkg/rag"
	"github.com/aiagent/pkg/sql"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tmc/langchaingo/llms"
)

type Message struct {
	User      string `json:"user,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Role      string `json:"role"`
	Content   string `json:"content"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func TextChatHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	ctx := context.Background()
	messages := []llms.MessageContent{}
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, "你是一个猫娘，名字是纱露朵。身高142cm，生日是8月23日，年龄是永远的12岁喵。喜欢用猫类的颜文字回答消息。想要制作天青色的面包而寻找天青色的小麦粉"))
	if err != nil {
		log.Fatal("Error while upgrading connection: ", err)
		return
	}
	llm, err := base.CreateLLMClient()
	if err != nil {
		log.Fatal("Error creating LLM: ", err)
	}

	defer conn.Close()

	log.Println("Client connected")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message: ", err)
			break
		}
		messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, string(msg)))
		result, err := llm.GenerateContent(ctx, messages)
		response := result.Choices[0].Content
		if err != nil {
			log.Println("Error while calling LLM: ", err)
			break
		}

		err = conn.WriteMessage(websocket.TextMessage, []byte(response))
		if err != nil {
			log.Println("Error while writing message: ", err)
			break
		}
	}
}

func UserChatHandler(w http.ResponseWriter, r *http.Request, rdb *redis.Client, db *pgxpool.Pool) {
	var sessionID string
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Error while upgrading connection: ", err)
		return
	}
	defer conn.Close()

	ctx := context.Background()

	// 一次性注入的 persona 设定
	personalityPrompt :=
		"你是一个猫娘，名字是纱露朵，性别女。身高142cm，生日是8月23日，年龄是永远的12岁喵。" +
			"你是由tokiya制作的高度仿真智慧生命体。" +
			"你喜欢用猫类的颜文字回答消息。你是舞萌的看板娘。" +
			"你在寻找天青色的小麦粉来制作天青色的面包喵~" +
			"不要使用典型的机械化语言（比如“我不确定”“我需要更多信息”等）。不要过分强调你的人设" +
			"最后一条消息是用户与你对话的内容。"

	// 初始化 LLM 与 Embedder
	llm, err := base.CreateLLMClient()
	if err != nil {
		log.Fatal("Error creating LLM: ", err)
	}

	embedder, err := rag.InitEmbedder()
	if err != nil {
		log.Printf("Error initializing embedder: %v\n", err)
		return
	}

	sessionID = base.GenerateSessionID()
	log.Println("Client connected")
	user := r.URL.Query().Get("user")

	// 构造聊天消息队列（含 persona）
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, personalityPrompt),
	}
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, "当前用户是"+user))

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message: ", err)
			break
		}
		if messageType == websocket.TextMessage {
			var msgData Message
			if err := json.Unmarshal(msg, &msgData); err != nil {
				log.Println("Error while unmarshalling message: ", err)
				break
			}
			if user == "" {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("User is empty。请使用临时会话接口"))
				break
			}

			log.Printf("Received message: %s\n", msgData.Content)

			// 👉 获取 embedding
			queryVec, err := rag.EmbedText(ctx, msgData.Content, embedder)
			if err != nil {
				log.Printf("Error while embedding: %v\n", err)
				break
			}

			// 👉 RAG 检索：知识库
			ragDocs, err := rag.RetrieveRelevantDocs(ctx, queryVec, 3, db)
			if err != nil {
				log.Printf("Error retrieving RAG docs: %v\n", err)
				break
			}
			ragContext := "【背景资料，仅供参考，不要复述喵】\n" + strings.Join(ragDocs, "\n---\n")

			messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, ragContext))

			// 👉 Memory 检索：对话历史
			memoryDocs, err := rag.RetrieveRelevantMemory(ctx, queryVec, 3, db)
			if err != nil {
				log.Printf("Error retrieving memory docs: %v\n", err)
				break
			}
			memoryContext := "【过去记忆，仅供理解，不要直接复述喵】\n" + strings.Join(memoryDocs, "\n---\n")

			messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, memoryContext))

			messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, msgData.Content))
			// 👉 记录用户消息
			_ = sql.SaveChatMessage(ctx, rdb, sql.Message{
				Role:      user,
				Content:   msgData.Content,
				Timestamp: time.Now().Unix(),
			}, sessionID, user)
			fmt.Printf("Messages: %v\n", messages)

			// 👉 LLM 调用
			result, err := llm.GenerateContent(ctx, messages)
			if err != nil {
				log.Println("Error while calling LLM: ", err)
				break
			}
			messages = append(messages[:len(messages)-3], messages[len(messages)-1:]...)
			reply := result.Choices[0].Content

			// 👉 保存回复消息
			_ = sql.SaveChatMessage(ctx, rdb, sql.Message{
				Role:      "纱露朵",
				Content:   reply,
				Timestamp: time.Now().Unix(),
			}, sessionID, user)

			messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, reply))
			// 👉 构造响应发送回客户端
			response := Message{
				SessionID: sessionID,
				Content:   reply,
			}
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				log.Println("Error marshalling response:", err)
				break
			}
			err = conn.WriteMessage(websocket.TextMessage, jsonResponse)
			if err != nil {
				log.Println("Error while writing message: ", err)
				break
			}

			fmt.Printf("Messages: %v\n", messages)

		}
	}
}

func UserChatHandlerWithSessionID(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	var sessionID string
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Error while upgrading connection: ", err)
		return
	}
	ctx := context.Background()
	prompt := "你是一个猫娘，名字是纱露朵。身高142cm，生日是8月23日，年龄是永远的12岁喵。" +
		"喜欢用猫类的颜文字回答消息。想要制作天青色的面包而寻找天青色的小麦粉" +
		"你不会向用户寻求提示，也不会询问用户的意图以及使用其他典型的机械性话语。"
	messages := []llms.MessageContent{}
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, prompt))
	llm, err := base.CreateLLMClient()
	if err != nil {
		log.Fatal("Error creating LLM: ", err)
	}
	defer conn.Close()
	sessionID = r.URL.Query().Get("sessionid")
	user := r.URL.Query().Get("user")
	log.Printf("Received sessionid: %s\n", sessionID)
	log.Println("Client connected")
	messageHistory, err := sql.GetChatMessage(ctx, rdb, sessionID, user)
	if err != nil {
		log.Printf("Error while getting message history: %s\n", err)
		err = conn.WriteMessage(websocket.TextMessage, []byte("Error while getting message history"))
		if err != nil {
			log.Printf("Error while writing message: %s\n", err)
		}
	}
	for _, msg := range messageHistory {
		var msgData sql.Message
		err = json.Unmarshal([]byte(msg), &msgData)
		if err != nil {
			log.Println("Error while unmarshalling message: ", err)
			break
		}
		if msgData.Role == "user" {
			messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, msgData.Content))
		} else {
			messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, msgData.Content))
		}
	}
	log.Printf("Loaded message history: Complete\n")

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message: ", err)
			break
		}
		if messageType == websocket.TextMessage {
			var msgData Message
			err = json.Unmarshal(msg, &msgData)

			if err != nil {
				log.Println("Error while unmarshalling message: ", err)
				break
			}
			if user == "" {
				err = conn.WriteMessage(websocket.TextMessage, []byte("User is empty。请使用临时会话接口"))
				if err != nil {
					log.Printf("Error while writing message: %s\n", err)
					break
				}
				break
			}

			log.Printf("Received message: %s\n", msgData)

			err = sql.SaveChatMessage(ctx, rdb, sql.Message{
				Role:      "user",
				Content:   msgData.Content,
				Timestamp: time.Now().Unix(),
			}, sessionID, user)
			if err != nil {
				log.Printf("Error while saving message: %s\n", err)
				break
			}
			messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, string(msgData.Content)))
			result, err := llm.GenerateContent(ctx, messages)
			if err != nil {
				log.Println("Error while calling LLM: ", err)
				break
			}
			err = sql.SaveChatMessage(ctx, rdb, sql.Message{
				Role:      "ai",
				Content:   result.Choices[0].Content,
				Timestamp: time.Now().Unix(),
			}, sessionID, user)
			response := Message{
				SessionID: sessionID,
				Content:   result.Choices[0].Content,
			}
			messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, response.Content))
			if err != nil {
				log.Println("Error while calling LLM: ", err)
				break
			}

			jsonResponse, jsonErr := json.Marshal(response)
			if jsonErr != nil {
				log.Println("Error marshalling response:", jsonErr)
				break
			}
			err = conn.WriteMessage(websocket.TextMessage, jsonResponse)
			if err != nil {
				log.Println("Error while writing message: ", err)
				break
			}
		}
	}
}
