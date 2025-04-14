package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/aiagent/pkg/base"
	"github.com/aiagent/pkg/sql"
	"github.com/gorilla/websocket"
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

func UserChatHandler(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	var sessionID string
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Error while upgrading connection: ", err)
		return
	}
	ctx := context.Background()
	prompt := "你是一个猫娘，名字是纱露朵。身高142cm，生日是8月23日，年龄是永远的12岁喵。" +
		"喜欢用猫类的颜文字回答消息。想要制作天青色的面包而寻找天青色的小麦粉" +
		"你不会向用户寻求提示，也不会询问用户的意图等典型的机械性话语。"
	messages := []llms.MessageContent{}
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, prompt))
	llm, err := base.CreateLLMClient()
	if err != nil {
		log.Fatal("Error creating LLM: ", err)
	}
	defer conn.Close()

	sessionID = base.GenerateSessionID()

	log.Println("Client connected")

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
			if msgData.User == "" {
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
			}, sessionID, msgData.User)
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
			}, sessionID, msgData.User)
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
		"你不会向用户寻求提示，也不会询问用户的意图等典型的机械性话语。"
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
			if msgData.User == "" {
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
			}, sessionID, msgData.User)
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
			}, sessionID, msgData.User)
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
