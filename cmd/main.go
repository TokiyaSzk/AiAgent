package main

import (
	"context"
	"fmt"
	"log"

	// 修正导入路径，使用相对路径导入本地包
	"github.com/aiagent/pkg"
	"github.com/tmc/langchaingo/llms"
)

func main() {

	llm, err := pkg.CreateLLMClient()
	if err != nil {
		log.Fatalf("Error creating LLM: %s", err)
	}

	ctx := context.Background()
	messages := []llms.MessageContent{}
	// completion, err := pkg.CallLLM("What is the capital of France?", llm, ctx)
	// if err != nil {
	// 	log.Fatalf("Error calling LLM: %s", err)
	// }
	// fmt.Printf("LLM response: %s\n", completion)
	for {
		var command string
		fmt.Print("请输入问题：")
		_, err := fmt.Scanln(&command)
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		newMessages, err := pkg.MakeMessages(llms.ChatMessageTypeHuman, command, messages)
		messages = newMessages
		if err != nil {
			log.Fatalf("Error creating messages: %s", err)
		}

		response, err := llm.GenerateContent(ctx, messages)
		if err != nil {
			log.Fatalf("Error calling LLM: %s", err)
		}

		// 打印返回的响应结果
		if response != nil && len(response.Choices) > 0 {
			fmt.Printf("AI response: %s\n", response.Choices[0].Content)
			newMessages, err := pkg.MakeMessages(llms.ChatMessageTypeAI, response.Choices[0].Content, messages)
			messages = newMessages

			if err != nil {
				log.Fatalf("Error creating messages: %s", err)
			} else {
				fmt.Print(messages)
			}
		} else {
			fmt.Println("Received empty response from LLM")
		}
	}
}
