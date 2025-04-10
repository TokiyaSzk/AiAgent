package main

import (
	"context"
	"fmt"
	"log"

	// 修正导入路径，使用相对路径导入本地包
	"github.com/aiagent/pkg/base"
	"github.com/aiagent/pkg/sql"
	"github.com/tmc/langchaingo/llms"
)

func main() {
	var charaPrompt string

	llm, err := base.CreateLLMClient()
	if err != nil {
		log.Fatalf("Error creating LLM: %s", err)
	}

	ctx := context.Background()

	messages := []llms.MessageContent{}
	db, err := sql.CreatePSQLClient(ctx)
	if err != nil {
		log.Fatalf("Error creating database client: %s", err)
	}
	rdb, err := sql.CreateRedisClient(ctx)
	sql.CleanInvalidCharaIDs(ctx, rdb)
	if err != nil {
		log.Fatalf("Error creating database client: %s", err)
	}
	defer db.Close()

	for {
		var command string
		fmt.Print(" 1:Start chat\n 2:create promt\n 3:choice prompt\n")
		fmt.Printf(" 4:search prompt\n 5:remove chara prompt\n exit:exit\n")
		_, err := fmt.Scanln(&command)
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		switch command {
		case "1":
			// Start Chat
			messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, charaPrompt))
			for {
				fmt.Printf("Input you messages:")
				var userMessage string
				_, err := fmt.Scanln(&userMessage)
				if err != nil {
					fmt.Println("Error reading input:", err)
					continue
				}
				messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, userMessage))
				response, err := llm.GenerateContent(ctx, messages)
				if err != nil {
					log.Fatalf("Error calling LLM: %s", err)
				}

				// 打印返回的响应结果
				if response != nil && len(response.Choices) > 0 {
					fmt.Printf("AI response: %s\n", response.Choices[0].Content)
					messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content))
					if err != nil {
						log.Fatalf("Error creating messages: %s", err)
					} else {
						// fmt.Print(messages)
					}
				} else {
					fmt.Println("Received empty response from LLM")
				}
			}
		case "2":
			// create prompt
			fmt.Printf("Please enter the role name: ")
			var chara string
			_, err := fmt.Scanln(&chara)
			if err != nil {
				fmt.Println("Error reading input:", err)
			}
			fmt.Printf("Please enter the role Content: ")
			var content string
			_, err = fmt.Scanln(&content)
			if err != nil {
				fmt.Println("Error reading input:", err)
			}

			err = sql.SaveCharaPrompt(ctx, rdb, chara, content)
			if err != nil {
				log.Fatalf("Error saving chara prompt: %s", err)
			}
		case "3":
			// choice prompt
			fmt.Printf("Please enter the role ID: ")
			var charaID string
			_, err := fmt.Scanln(&charaID)
			if err != nil {
				fmt.Println("Error reading input:", err)
				continue
			}

			charaPrompt, err := sql.GetCharaPrompt(ctx, rdb, "ai:chara:"+charaID)
			if err != nil {
				fmt.Println("Error Search Chara by ID:", err)
				continue
			}

			fmt.Printf("You have choice Role ID %s\n The Prompt is : %s", charaPrompt.Name, charaPrompt.Prompt)
		case "4":
			// search prompt
			result, err := sql.GetAllCharaIDs(ctx, rdb)
			if err != nil {
				fmt.Printf("Error getting chara IDs: %v\n", err)
				continue
			}
			for i := range result {
				chara, err := sql.GetCharaPrompt(ctx, rdb, "ai:chara:"+result[i])
				if err != nil {
					fmt.Printf("Error getting chara prompt: %v\n", err)
					continue
				}
				fmt.Printf("Role ID: %s, Name: %s\n", result[i], chara.Name)
			}
		case "5":
			// delete chara prompt
			fmt.Printf("Please enter the role ID to delete: ")
			var charaID string
			_, err := fmt.Scanln(&charaID)
			if err != nil {
				fmt.Println("Error reading input:", err)
				continue
			}
			err = sql.RemoveCharaPrompt(ctx, rdb, charaID)
			if err != nil {
				fmt.Printf("Error deleting chara prompt: %v\n", err)
				continue
			}
			fmt.Printf("Chara prompt with ID %s deleted successfully\n", charaID)
		case "exit":
			return
		}

	}
}
