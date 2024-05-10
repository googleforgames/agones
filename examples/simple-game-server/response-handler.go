package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	sdk "agones.dev/agones/sdks/go"
)

type responseHandler func(s *sdk.SDK, parts []string, cancel ...context.CancelFunc) (response string, addACK bool, responseError error)

var responseMap = map[string]responseHandler{
	"EXIT":                 handleExit,
	"UNHEALTHY":            handleUnhealthy,
	"GAMESERVER":           handleGameServer,
	"READY":                handleReady,
	"ALLOCATE":             handleAllocate,
	"RESERVE":              handleReserve,
	"WATCH":                handleWatch,
	"LABEL":                handleLabel,
	"CRASH":                handleCrash,
	"ANNOTATION":           handleAnnotation,
	"PLAYER_CAPACITY":      handlePlayerCapacity,
	"PLAYER_CONNECT":       handlePlayerConnect,
	"PLAYER_DISCONNECT":    handlePlayerDisconnect,
	"PLAYER_CONNECTED":     handlePlayerConnected,
	"GET_PLAYERS":          handleGetPlayers,
	"PLAYER_COUNT":         handlePlayerCount,
	"GET_COUNTER_COUNT":    handleGetCounterCount,
	"INCREMENT_COUNTER":    handleIncrementCounter,
	"DECREMENT_COUNTER":    handleDecrementCounter,
	"SET_COUNTER_COUNT":    handleSetCounterCount,
	"GET_COUNTER_CAPACITY": handleGetCounterCapacity,
	"SET_COUNTER_CAPACITY": handleSetCounterCapacity,
	"GET_LIST_CAPACITY":    handleGetListCapacity,
	"SET_LIST_CAPACITY":    handleSetListCapacity,
	"LIST_CONTAINS":        handleListContains,
	"GET_LIST_LENGTH":      handleGetListLength,
	"GET_LIST_VALUES":      handleGetListValues,
	"APPEND_LIST_VALUE":    handleAppendListValue,
	"DELETE_LIST_VALUE":    handleDeleteListValue,
}

func handleResponse(txt string, s *sdk.SDK, cancel context.CancelFunc) (response string, addACK bool, responseError error) {
	parts := strings.Split(strings.TrimSpace(txt), " ")
	response = txt
	addACK = true
	responseError = nil

	handler, exists := responseMap[parts[0]]
	if !exists {
		return fmt.Sprintf("Unknown response command: %s", parts[0]), true, nil
	}
	if parts[0] == "UNHEALTHY" {
		return handler(s, parts, cancel)
	}

	return handler(s, parts)
}

func handleExit(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	// handle elsewhere, as we respond before exiting
	return
}

func handleUnhealthy(s *sdk.SDK, parts []string, cancel ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(cancel) > 0 {
		cancel[0]() // Invoke cancel function if provided
	}
	return
}

func handleGameServer(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response = gameServerName(s)
	addACK = false
	return
}

func handleReady(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	ready(s)
	return
}

func handleAllocate(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	allocate(s)
	return
}

func handleReserve(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) != 2 {
		response = "Invalid RESERVE, should have 1 argument"
		responseError = fmt.Errorf("Invalid RESERVE, should have 1 argument")
	}
	if dur, err := time.ParseDuration(parts[1]); err != nil {
		response = fmt.Sprintf("%s\n", err)
		responseError = err
	} else {
		reserve(s, dur)
	}
	return
}

func handleWatch(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	watchGameServerEvents(s)
	return
}

func handleLabel(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	switch len(parts) {
	case 0:
		// Legacy format
		setLabel(s, "timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	case 2:
		setLabel(s, parts[1], parts[2])
	default:
		response = "Invalid LABEL command, must use zero or 2 arguments"
		responseError = fmt.Errorf("Invalid LABEL command, must use zero or 2 arguments")
	}
	return
}

func handleCrash(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	log.Print("Crashing.")
	os.Exit(1)
	return "", false, nil
}

func handleAnnotation(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	switch len(parts) {
	case 1:
		// Legacy format
		setAnnotation(s, "timestamp", time.Now().UTC().String())
	case 2:
		setAnnotation(s, parts[1], parts[2])
	default:
		response = "Invalid ANNOTATION command, must use zero or 2 arguments"
		responseError = fmt.Errorf("Invalid ANNOTATION command, must use zero or 2 arguments")
	}
	return
}

func handlePlayerCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	switch len(parts) {
	case 1:
		response = getPlayerCapacity(s)
		addACK = false
	case 2:
		if cap, err := strconv.Atoi(parts[0]); err != nil {
			response = fmt.Sprintf("%s", err)
			responseError = err
		} else {
			setPlayerCapacity(s, int64(cap))
		}
	default:
		response = "Invalid PLAYER_CAPACITY, should have 0 or 1 arguments"
		responseError = fmt.Errorf("Invalid PLAYER_CAPACITY, should have 0 or 1 arguments")
	}
	return
}

func handlePlayerConnect(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid PLAYER_CONNECT, should have 1 argument"
		responseError = fmt.Errorf("Invalid PLAYER_CONNECT, should have 1 argument")
		return
	}
	playerConnect(s, parts[1])
	return
}

func handlePlayerDisconnect(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid PLAYER_DISCONNECT, should have 1 argument"
		responseError = fmt.Errorf("Invalid PLAYER_DISCONNECT, should have 1 argument")
		return
	}
	playerDisconnect(s, parts[1])
	return
}

func handlePlayerConnected(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid PLAYER_CONNECTED, should have 1 argument"
		responseError = fmt.Errorf("Invalid PLAYER_CONNECTED, should have 1 argument")
		return
	}
	response = playerIsConnected(s, parts[1])
	addACK = false
	return
}

func handleGetPlayers(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response = getConnectedPlayers(s)
	addACK = false
	return
}

func handlePlayerCount(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response = getPlayerCount(s)
	addACK = false
	return
}

func handleGetCounterCount(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_COUNTER_COUNT, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_COUNTER_COUNT, should have 1 argument")
		return
	}
	response, responseError = getCounterCount(s, parts[1])
	addACK = false
	return
}

func handleIncrementCounter(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid INCREMENT_COUNTER, should have 2 arguments"
		responseError = fmt.Errorf("Invalid INCREMENT_COUNTER, should have 2 arguments")
		return
	}
	response, err := incrementCounter(s, parts[1], parts[2])
	if err != nil {
		responseError = err
	}
	addACK = false
	return
}

func handleDecrementCounter(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid DECREMENT_COUNTER, should have 2 arguments"
		responseError = fmt.Errorf("Invalid DECREMENT_COUNTER, should have 2 arguments")
		return
	}
	response, err := decrementCounter(s, parts[1], parts[2])
	if err != nil {
		responseError = err
	}
	addACK = false
	return
}

func handleSetCounterCount(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid SET_COUNTER_COUNT, should have 2 arguments"
		responseError = fmt.Errorf("Invalid SET_COUNTER_COUNT, should have 2 arguments")
		return
	}
	response, err := setCounterCount(s, parts[1], parts[2])
	if err != nil {
		responseError = err
	}
	addACK = false
	return
}

func handleGetCounterCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_COUNTER_CAPACITY, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_COUNTER_CAPACITY, should have 1 argument")
		return
	}
	response, responseError = getCounterCapacity(s, parts[1])
	addACK = false
	return
}

func handleSetCounterCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid SET_COUNTER_CAPACITY, should have 2 arguments"
		responseError = fmt.Errorf("Invalid SET_COUNTER_CAPACITY, should have 2 arguments")
		return
	}
	response, err := setCounterCapacity(s, parts[1], parts[2])
	if err != nil {
		responseError = err
	}
	addACK = false
	return
}

func handleGetListCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_LIST_CAPACITY, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_LIST_CAPACITY, should have 1 argument")
		return
	}
	response, responseError = getListCapacity(s, parts[1])
	addACK = false
	return
}

func handleSetListCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid SET_LIST_CAPACITY, should have 2 arguments"
		responseError = fmt.Errorf("Invalid SET_LIST_CAPACITY, should have 2 arguments")
		return
	}
	response, err := setListCapacity(s, parts[1], parts[2])
	if err != nil {
		responseError = err
	}
	addACK = false
	return
}

func handleListContains(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid LIST_CONTAINS, should have 2 arguments"
		responseError = fmt.Errorf("Invalid LIST_CONTAINS, should have 2 arguments")
		return
	}
	response, responseError = listContains(s, parts[1], parts[2])
	addACK = false
	return
}

func handleGetListLength(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_LIST_LENGTH, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_LIST_LENGTH, should have 1 argument")
		return
	}
	response, responseError = getListLength(s, parts[1])
	addACK = false
	return
}

func handleGetListValues(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_LIST_VALUES, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_LIST_VALUES, should have 1 argument")
		return
	}
	response, responseError = getListValues(s, parts[1])
	addACK = false
	return
}

func handleAppendListValue(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid APPEND_LIST_VALUE, should have 2 arguments"
		responseError = fmt.Errorf("Invalid APPEND_LIST_VALUE, should have 2 arguments")
		return
	}
	response, err := appendListValue(s, parts[1], parts[2])
	if err != nil {
		responseError = err
	}
	addACK = false
	return
}

func handleDeleteListValue(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid DELETE_LIST_VALUE, should have 2 arguments"
		responseError = fmt.Errorf("Invalid DELETE_LIST_VALUE, should have 2 arguments")
		return
	}
	response, err := deleteListValue(s, parts[1], parts[2])
	if err != nil {
		responseError = err
	}
	addACK = false
	return
}
