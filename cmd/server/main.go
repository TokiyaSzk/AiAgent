package main

import (
	"context"
	"log"
	"net/http"

	"github.com/aiagent/internal/handler"
	"github.com/aiagent/pkg/sql"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Fatal("Error while upgrading connection: ", err)
		return
	}
	defer conn.Close()

	log.Println("Client connected")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message: ", err)
			break
		}
		response := "Pong!" + string(msg)

		err = conn.WriteMessage(websocket.TextMessage, []byte(response))
		if err != nil {
			log.Println("Error while writing message: ", err)
			break
		}
	}
}

func main() {
	ctx := context.Background()

	db, err := sql.CreatePSQLClient(ctx)
	if err != nil {
		log.Fatalf("Error creating client: %s", err)
	}
	err = sql.CreatePSQLDatabase(ctx, db)
	if err != nil {
		log.Fatalf("Error creating database: %s", err)
	}
	err = sql.CreatePSQLTable(ctx, db)
	if err != nil {
		log.Fatalf("Error creating table: %s", err)
	}
	rdb, err := sql.CreateRedisClient(ctx)
	if err != nil {
		log.Fatalf("Error creating Redis client: %s", err)
	}
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/ws/chat/temp", handler.TextChatHandler)
	http.HandleFunc("/ws/chat/user", func(w http.ResponseWriter, r *http.Request) {
		handler.UserChatHandler(w, r, rdb)
	})
	http.HandleFunc("/ws/chat/user/continue", func(w http.ResponseWriter, r *http.Request) {
		handler.UserChatHandlerWithSessionID(w, r, rdb)
	})
	log.Println("WebSocket server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
