package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aiagent/pkg/sql"
	"github.com/aiagent/pkg/timer"
)

func TestTimerSummaryMemory(t *testing.T) {
	ctx := context.Background()
	rdb, err := sql.CreateRedisClient(ctx)
	if err != nil {
		fmt.Printf("Error creating Redis client: %s", err)
	}

	defer rdb.Close()
	db, err := sql.CreatePSQLClient(ctx)
	if err != nil {
		fmt.Printf("Error creating table: %s", err)
	}
	defer db.Close()
	timer.TimerSummaryMemory(rdb, db, "12345")
}
