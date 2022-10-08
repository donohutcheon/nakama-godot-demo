package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/heroiclabs/nakama-common/runtime"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

func RpcHealthCheck(ctx context.Context, logger runtime.Logger, db *sql.DB,
	nk runtime.NakamaModule, payload string) (string, error) {

	logger.Debug("Healthcheck RPC called")
	response := HealthCheckResponse{
		Status: "OK",
	}
	out, err := json.Marshal(response)
	if err != nil {
		logger.Error("Error marshalling response type to JSON: %v", err)
	}

	return string(out), nil
}
