package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/heroiclabs/nakama-common/runtime"
)

type OpCodeType int

const (
	updatePosition OpCodeType = 1
	updateInput    OpCodeType = 2
	updateState    OpCodeType = 3
	updateJump     OpCodeType = 4
	doSpawn        OpCodeType = 5
	updateColor    OpCodeType = 6
	initialState   OpCodeType = 7
)

var spawnPosition = []float64{1800.0, 1280.0}

type MatchState struct {
	presences  map[string]runtime.Presence
	emptyTicks int
	inputs     map[string]Input    `json:"inp"`
	positions  map[string]Position `json:"pos"`
	colors     map[string]string   `json:"col"`
	names      map[string]string   `json:"nms"`
}

func newMatchState() *MatchState {
	matchState := new(MatchState)
	matchState.presences = make(map[string]runtime.Presence)
	matchState.inputs = make(map[string]Input)
	matchState.positions = make(map[string]Position)
	matchState.colors = make(map[string]string)
	matchState.names = make(map[string]string)
	matchState.emptyTicks = 0

	return matchState
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type UpdatePositionMessage struct {
	Id  string   `json:"id"`
	Pos Position `json:"pos"`
}

type Input struct {
	Direction float64 `json:"dir"`
	Jump      int     `json:"jmp"`
}

type UpdateInputMessage struct {
	Id    string  `json:"id"`
	Input float64 `json:"inp"`
}

type UpdateColorMessage struct {
	Id    string `json:"id"`
	Color string `json:"color"` // example "0.811765,1,0.439216,1"
}

type DoSpawnMessage struct {
	Id    string `json:"id"`
	Color string `json:"col"` // example "0.811765,1,0.439216,1"
	Name  string `json:"nm"`
}

type UpdateStateMessage struct {
	Positions map[string]Position `json:"pos"`
	Inputs    map[string]Input    `json:"inp"`
}

var Operations = map[OpCodeType]func(ctx context.Context, msg runtime.MatchData,
	state *MatchState, logger runtime.Logger, nk runtime.NakamaModule,
	dispatcher runtime.MatchDispatcher) error{

	updatePosition: updatePositionOp,
	updateInput:    updateInputOp,
	updateJump:     updateJumpOp,
	doSpawn:        doSpawnOp,
	updateColor:    updateColorOp,
}

func updatePositionOp(ctx context.Context, msg runtime.MatchData, state *MatchState,
	_ runtime.Logger, nk runtime.NakamaModule,
	_ runtime.MatchDispatcher) error {

	data := msg.GetData()
	positionUpdate := new(UpdatePositionMessage)
	err := json.Unmarshal(data, positionUpdate)
	if err != nil {
		return err
	}
	id := positionUpdate.Id
	position := positionUpdate.Pos
	_, ok := state.positions[id]
	if !ok {
		return errors.New(fmt.Sprintf("position does not exist for user: %s", id))
	}
	state.positions[id] = position

	return nil
}

func updateInputOp(ctx context.Context, msg runtime.MatchData, state *MatchState,
	_ runtime.Logger, nk runtime.NakamaModule,
	_ runtime.MatchDispatcher) error {

	data := msg.GetData()
	inputUpdate := new(UpdateInputMessage)
	err := json.Unmarshal(data, inputUpdate)
	if err != nil {
		return err
	}
	id := inputUpdate.Id
	dir := inputUpdate.Input
	input, ok := state.inputs[id]
	if !ok {
		return errors.New(fmt.Sprintf("inputs does not exist for user: %s", id))
	}
	input.Direction = dir

	return nil
}

func updateJumpOp(ctx context.Context, msg runtime.MatchData, state *MatchState,
	_ runtime.Logger, nk runtime.NakamaModule,
	_ runtime.MatchDispatcher) error {

	data := msg.GetData()
	inputUpdate := new(UpdateInputMessage)
	err := json.Unmarshal(data, inputUpdate)
	if err != nil {
		return err
	}
	id := inputUpdate.Id
	input, ok := state.inputs[id]
	if !ok {
		return errors.New(fmt.Sprintf("inputs does not exist for user: %s", id))
	}

	input.Jump = 1

	return nil
}

func doSpawnOp(ctx context.Context, msg runtime.MatchData, state *MatchState,
	logger runtime.Logger, nk runtime.NakamaModule,
	dispatcher runtime.MatchDispatcher) error {

	data := msg.GetData()
	spawn := new(DoSpawnMessage)
	err := json.Unmarshal(data, spawn)
	if err != nil {
		return err
	}
	userId := spawn.Id
	name := spawn.Name

	_, ok := state.names[userId]
	if !ok {
		return errors.New("cannot update name for userID, existing name does not exist")
	}
	_, ok = state.colors[userId]
	if !ok {
		return errors.New("cannot update color for userID, existing color does not exist")
	}

	state.names[userId] = name
	state.colors[userId] = spawn.Color

	readParams := []*runtime.StorageRead{
		{
			Collection: "player_data",
			Key:        "position_" + name,
			UserID:     msg.GetUserId(),
		},
	}
	positions, err := nk.StorageRead(ctx, readParams)
	if err != nil {
		logger.Error("failed to read storage: %#v", readParams)
		return err
	}

	position := new(Position)
	isSet := false
	for _, object := range positions {
		err = json.Unmarshal([]byte(object.Value), position)
		if err != nil {
			return err
		}
		state.positions[userId] = *position
		isSet = true
	}

	if !isSet {
		state.positions[userId] = Position{
			X: spawnPosition[0],
			Y: spawnPosition[1],
		}
	}

	dispatchState := struct {
		Positions map[string]Position `json:"pos"`
		Inputs    map[string]Input    `json:"inp"`
		Colors    map[string]string   `json:"col"`
		Names     map[string]string   `json:"nms"`
	}{
		Positions: state.positions,
		Inputs:    state.inputs,
		Colors:    state.colors,
		Names:     state.names,
	}

	encoded, err := json.Marshal(dispatchState)
	if err != nil {
		logger.Error("failed to marshal data: %#v", dispatchState)
		return err
	}
	presence, ok := state.presences[userId]
	if !ok {
		logger.Error("presence not found for user: %s", userId)
		return err
	}
	err = dispatcher.BroadcastMessage(int64(initialState), encoded, nil, presence, true)
	if err != nil {
		return err
	}
	err = dispatcher.BroadcastMessage(int64(doSpawn), msg.GetData(), nil, presence, true)
	if err != nil {
		return err
	}

	return nil
}

func updateColorOp(ctx context.Context, msg runtime.MatchData, state *MatchState,
	_ runtime.Logger, nk runtime.NakamaModule,
	dispatcher runtime.MatchDispatcher) error {

	data := msg.GetData()
	colorUpdate := new(UpdateColorMessage)
	err := json.Unmarshal(data, colorUpdate)
	if err != nil {
		return err
	}
	id := colorUpdate.Id
	color := colorUpdate.Color
	_, ok := state.colors[id]
	if ok {
		state.colors[id] = color
	}

	err = dispatcher.BroadcastMessage(int64(updateColor), data, nil, nil, true)
	if err != nil {
		return err
	}

	return nil
}

type Match struct{}

func (m *Match) MatchJoinAttempt(
	_ context.Context,
	_ runtime.Logger,
	_ *sql.DB,
	_ runtime.NakamaModule,
	_ runtime.MatchDispatcher,
	_ int64,
	state any,
	presence runtime.Presence,
	_ map[string]string) (
	any, bool, string) {

	matchState, ok := state.(*MatchState)
	if !ok {
		return state, false, "state not a valid lobby state object"
	}

	_, ok = matchState.presences[presence.GetUserId()]
	if ok {
		return state, false, "the user is already logged in"
	}

	return state, true, "user can join"
}

func (m *Match) MatchTerminate(
	ctx context.Context,
	logger runtime.Logger,
	_ *sql.DB,
	nk runtime.NakamaModule,
	_ runtime.MatchDispatcher,
	_ int64,
	state any,
	_ int) any {

	matchState, ok := state.(*MatchState)
	if !ok {
		return state
	}

	for userID, position := range matchState.positions {
		value, err := json.Marshal(position)
		if err != nil {
			logger.Error("could not marshal position %#v", err)
			continue
		}

		err = writeToStorage(ctx, nk, "player_data", "position_"+matchState.names[userID], userID, string(value))
		if err != nil {
			logger.Error("could not save player %s position %#v", userID, err)
			continue
		}
	}

	return state
}

func (m *Match) MatchSignal(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state any, data string) (any, string) {
	return state, ""
}

func (m *Match) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]any) (any, int, string) {
	state := newMatchState()
	state.emptyTicks = 0
	tickRate := 20
	label := "Social World"

	return state, tickRate, label
}

func (m *Match) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state any, presences []runtime.Presence) any {
	lobbyState, ok := state.(*MatchState)
	if !ok {
		logger.Error("state not a valid match state object")
	}

	for i := 0; i < len(presences); i++ {
		userId := presences[i].GetUserId()
		lobbyState.presences[userId] = presences[i]
		lobbyState.positions[userId] = Position{X: 0, Y: 0}
		lobbyState.inputs[userId] = Input{
			Direction: 0,
			Jump:      0,
		}
		lobbyState.colors[userId] = "1,1,1,1"
		lobbyState.names[userId] = "User"
	}

	return lobbyState
}

func (m *Match) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB,
	nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64,
	state any, presences []runtime.Presence) any {

	matchState, ok := state.(*MatchState)
	if !ok {
		logger.Error("state not a valid lobby state object")
		return matchState
	}
	for i := 0; i < len(presences); i++ {
		userID := presences[i].GetUserId()
		name, ok := matchState.names[userID]
		if !ok {
			continue
		}

		position, ok := matchState.positions[userID]
		if !ok {
			continue
		}

		value, err := json.Marshal(position)
		if err != nil {
			logger.Error("could not marshal position %#v", err)
			continue
		}
		
		err = writeToStorage(ctx, nk, "player_data", userID, "position_"+name, string(value))
		if err != nil {
			logger.Error("could not save player's position %#v", err)
		}

		delete(matchState.presences, userID)
		delete(matchState.positions, userID)
		delete(matchState.inputs, userID)
		delete(matchState.colors, userID)
		delete(matchState.names, userID)
	}

	return matchState
}

func writeToStorage(ctx context.Context, nk runtime.NakamaModule, collection, userID, key, value string) error {
	w := &runtime.StorageWrite{
		Collection: collection,
		UserID:     userID,
		Key:        key,
		Value:      value,
	}
	_, err := nk.StorageWrite(ctx, []*runtime.StorageWrite{w})
	if err != nil {
		return err
	}

	return nil
}

func (m *Match) MatchLoop(
	ctx context.Context,
	logger runtime.Logger,
	_ *sql.DB,
	nk runtime.NakamaModule,
	dispatcher runtime.MatchDispatcher,
	_ int64,
	state any,
	messages []runtime.MatchData) any {

	matchState, ok := state.(*MatchState)
	if !ok {
		logger.Error("state not a valid lobby state object")
		return nil
	}

	for _, msg := range messages {
		opCode := OpCodeType(msg.GetOpCode())

		err := Operations[opCode](ctx, msg, matchState, logger, nk, dispatcher)
		if err != nil {
			logger.Error("operation failed: type: %d, err: %s", opCode, err)
			continue
		}
	}
	err := broadcastState(dispatcher, matchState)
	if err != nil {
		logger.Error("failed to broadcast state update: %s", err)
	}

	return matchState
}

func broadcastState(dispatcher runtime.MatchDispatcher,
	state *MatchState) error {

	data := UpdateStateMessage{
		Positions: state.positions,
		Inputs:    state.inputs,
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return dispatcher.BroadcastMessage(int64(updateState), encoded, nil, nil, true)
}
