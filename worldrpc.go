package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/heroiclabs/nakama-common/runtime"
)

func getCharacterNameCollection(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule) ([]string, error) {
	params := []*runtime.StorageRead{
		{
			Collection: "global_data",
			Key:        "names",
		},
	}
	objects, err := nk.StorageRead(ctx, params)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	var data map[string]any
	for _, object := range objects {
		err := json.Unmarshal([]byte(object.GetValue()), &data)
		if err != nil {
			return nil, err
		}
		logger.Info("Get Character Collection: value: %#v; type: %T", data, data["names"])
		list, ok := data["names"].([]string)
		if !ok {
			break
		}
		for _, name := range list {
			names = append(names, name)
		}
	}

	return names, nil
}

func updateNames(ctx context.Context, nk runtime.NakamaModule, names []string) error {
	data := make(map[string]any)
	data["names"] = names
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	params := []*runtime.StorageWrite{
		{
			Collection:      "global_data",
			Key:             "names",
			Value:           string(bytes),
			PermissionRead:  2,
			PermissionWrite: 1,
		},
	}
	_, err = nk.StorageWrite(ctx, params)
	if err != nil {
		return err
	}

	return nil
}

func RPCRegisterCharacterName(ctx context.Context, logger runtime.Logger, db *sql.DB,
	nk runtime.NakamaModule, name string) (string, error) {
	names, err := getCharacterNameCollection(ctx, logger, nk)
	if err != nil {
		return "", err
	}

	for _, n := range names {
		if n == name {
			return "0", nil
		}
	}

	names = append(names, name)
	err = updateNames(ctx, nk, names)
	if err != nil {
		return "", err
	}

	return "1", nil
}

func RPCRemoveCharacterName(ctx context.Context, logger runtime.Logger, db *sql.DB,
	nk runtime.NakamaModule, name string) (string, error) {

	names, err := getCharacterNameCollection(ctx, logger, nk)
	if err != nil {
		return "", err
	}

	removed := false
	updated := make([]string, 0, len(names))
	for _, n := range names {
		if n == name {
			removed = true
			continue
		}
		updated = append(updated, n)
	}
	if removed {
		return "0", nil
	}

	names = append(names, name)
	err = updateNames(ctx, nk, names)
	if err != nil {
		return "", err
	}

	return "1", nil
}

func RPCGetWorldID(ctx context.Context, logger runtime.Logger, db *sql.DB,
	nk runtime.NakamaModule, payload string) (string, error) {
	limit := 1
	authoritative := true
	label := ""
	minSize := 0
	maxSize := 4
	query := "*"
	// Get list of matches from Nakama
	matches, err := nk.MatchList(ctx, limit, authoritative, label, &minSize, &maxSize, query)
	if err != nil {
		logger.Error("Error getting match list: %v", err)
		return "", err
	}
	logger.Info("Matches: %#v", matches)

	if len(matches) == 0 {
		// No matches found, create a new one
		params := make(map[string]any)
		matchID, err := nk.MatchCreate(ctx, "world", params)
		if err != nil {
			logger.Error("Error creating match: %v", err)
			return "", err
		}
		return matchID, nil
	}

	return matches[0].MatchId, nil
}
