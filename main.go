package main

import (
	"context"
	"database/sql"
	"github.com/heroiclabs/nakama-common/runtime"
	"time"
)

// Nakama InitModule
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	initStart := time.Now()

	err := initializer.RegisterRpc("health_check", RpcHealthCheck)
	if err != nil {
		return err
	}
	err = initializer.RegisterRpc("get_world_id", RPCGetWorldID)
	if err != nil {
		return err
	}

	err = initializer.RegisterRpc("register_character_name", RPCRegisterCharacterName)
	if err != nil {
		return err
	}

	err = initializer.RegisterRpc("remove_character_name", RPCRemoveCharacterName)
	if err != nil {
		return err
	}

	if err := initializer.RegisterMatch("world", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
		return &Match{}, nil
	}); err != nil {
		logger.Error("unable to register: %v", err)
		return err
	}

	logger.Info("Module loaded in %dms", time.Since(initStart).Milliseconds())
	return nil
}
