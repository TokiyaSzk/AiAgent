package test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/aiagent/pkg/mem"
	"github.com/aiagent/pkg/sql"
)

func TestSummaryMemmory(t *testing.T) {
	log.SetOutput(os.Stdout)
	ctx := context.Background()
	rdb, err := sql.CreateRedisClient(ctx)
	if err != nil {
		log.Fatalf("Error creating Redis client: %v", err)
	}

	db, err := sql.CreatePSQLClient(ctx)
	if err != nil {
		log.Fatalf("Error creating PSQL client: %v", err)
	}

	sessionID := "20250414210843"
	user := "tokiya"

	summary, err := mem.SummaryMemory(ctx, rdb, db, sessionID, user)
	if err != nil {
		log.Fatalf("Error getting summary: %v", err)
	}
	log.Print("summary:", summary)
}
