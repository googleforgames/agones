package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	coresdk "agones.dev/agones/pkg/sdk"
	sdk "agones.dev/agones/sdks/go"
)

// ready attempts to mark this gameserver as ready
func ready(s *sdk.SDK) {
	err := s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
	}
}

// allocate attempts to allocate this gameserver
func allocate(s *sdk.SDK) {
	err := s.Allocate()
	if err != nil {
		log.Fatalf("could not allocate gameserver: %v", err)
	}
}

// reserve for 10 seconds
func reserve(s *sdk.SDK, duration time.Duration) {
	if err := s.Reserve(duration); err != nil {
		log.Fatalf("could not reserve gameserver: %v", err)
	}
}

// exit shutdowns the server
func exit(s *sdk.SDK) {
	log.Printf("Received EXIT command. Exiting.")
	// This tells Agones to shutdown this Game Server
	shutdownErr := s.Shutdown()
	if shutdownErr != nil {
		log.Printf("Could not shutdown")
	}
	// The process will exit when Agones removes the pod and the
	// container receives the SIGTERM signal
}

// gameServerName returns the GameServer name
func gameServerName(s *sdk.SDK) string {
	var gs *coresdk.GameServer
	gs, err := s.GameServer()
	if err != nil {
		log.Fatalf("Could not retrieve GameServer: %v", err)
	}
	var j []byte
	j, err = json.Marshal(gs)
	if err != nil {
		log.Fatalf("error mashalling GameServer to JSON: %v", err)
	}
	log.Printf("GameServer: %s \n", string(j))
	return "NAME: " + gs.ObjectMeta.Name + "\n"
}

// watchGameServerEvents creates a callback to log when
// gameserver events occur
func watchGameServerEvents(s *sdk.SDK) {
	err := s.WatchGameServer(func(gs *coresdk.GameServer) {
		j, err := json.Marshal(gs)
		if err != nil {
			log.Fatalf("error mashalling GameServer to JSON: %v", err)
		}
		log.Printf("GameServer Event: %s \n", string(j))
	})
	if err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
}

// setAnnotation sets a given annotation
func setAnnotation(s *sdk.SDK, key, value string) {
	log.Printf("Setting annotation %v=%v", key, value)
	err := s.SetAnnotation(key, value)
	if err != nil {
		log.Fatalf("could not set annotation: %v", err)
	}
}

// setLabel sets a given label
func setLabel(s *sdk.SDK, key, value string) {
	log.Printf("Setting label %v=%v", key, value)
	// label values can only be alpha, - and .
	err := s.SetLabel(key, value)
	if err != nil {
		log.Fatalf("could not set label: %v", err)
	}
}

// setPlayerCapacity sets the player capacity to the given value
func setPlayerCapacity(s *sdk.SDK, capacity int64) {
	log.Printf("Setting Player Capacity to %d", capacity)
	if err := s.Alpha().SetPlayerCapacity(capacity); err != nil {
		log.Fatalf("could not set capacity: %v", err)
	}
}

// getPlayerCapacity returns the current player capacity as a string
func getPlayerCapacity(s *sdk.SDK) string {
	log.Print("Getting Player Capacity")
	capacity, err := s.Alpha().GetPlayerCapacity()
	if err != nil {
		log.Fatalf("could not get capacity: %v", err)
	}
	return strconv.FormatInt(capacity, 10) + "\n"
}

// playerConnect connects a given player
func playerConnect(s *sdk.SDK, id string) {
	log.Printf("Connecting Player: %s", id)
	if _, err := s.Alpha().PlayerConnect(id); err != nil {
		log.Fatalf("could not connect player: %v", err)
	}
}

// playerDisconnect disconnects a given player
func playerDisconnect(s *sdk.SDK, id string) {
	log.Printf("Disconnecting Player: %s", id)
	if _, err := s.Alpha().PlayerDisconnect(id); err != nil {
		log.Fatalf("could not disconnect player: %v", err)
	}
}

// playerIsConnected returns a bool as a string if a player is connected
func playerIsConnected(s *sdk.SDK, id string) string {
	log.Printf("Checking if player %s is connected", id)

	connected, err := s.Alpha().IsPlayerConnected(id)
	if err != nil {
		log.Fatalf("could not retrieve if player is connected: %v", err)
	}

	return strconv.FormatBool(connected) + "\n"
}

// getConnectedPlayers returns a comma delimeted list of connected players
func getConnectedPlayers(s *sdk.SDK) string {
	log.Print("Retrieving connected player list")
	list, err := s.Alpha().GetConnectedPlayers()
	if err != nil {
		log.Fatalf("could not retrieve connected players: %s", err)
	}

	return strings.Join(list, ",") + "\n"
}

// getPlayerCount returns the count of connected players as a string
func getPlayerCount(s *sdk.SDK) string {
	log.Print("Retrieving connected player count")
	count, err := s.Alpha().GetPlayerCount()
	if err != nil {
		log.Fatalf("could not retrieve player count: %s", err)
	}
	return strconv.FormatInt(count, 10) + "\n"
}

// getCounterCount returns the Count of the given Counter as a string
func getCounterCount(s *sdk.SDK, counterName string) (string, error) {
	log.Printf("Retrieving Counter %s Count", counterName)
	count, err := s.Beta().GetCounterCount(counterName)
	if err != nil {
		log.Printf("Error getting Counter %s Count: %s", counterName, err)
		return strconv.FormatInt(count, 10), err
	}
	return "COUNTER: " + strconv.FormatInt(count, 10) + "\n", nil
}

// incrementCounter returns the if the Counter Count was incremented successfully or not
func incrementCounter(s *sdk.SDK, counterName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("Could not increment Counter %s by unparseable amount %s: %s", counterName, amount, err)
	}
	log.Printf("Incrementing Counter %s Count by amount %d", counterName, amountInt)
	err = s.Beta().IncrementCounter(counterName, amountInt)
	if err != nil {
		log.Printf("Error incrementing Counter %s Count by amount %d: %s", counterName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// decrementCounter returns if the Counter Count was decremented successfully or not
func decrementCounter(s *sdk.SDK, counterName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not decrement Counter %s by unparseable amount %s: %s", counterName, amount, err)
	}
	log.Printf("Decrementing Counter %s Count by amount %d", counterName, amountInt)
	err = s.Beta().DecrementCounter(counterName, amountInt)
	if err != nil {
		log.Printf("Error decrementing Counter %s Count by amount %d: %s", counterName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// setCounterCount returns the if the Counter was set to a new Count successfully or not
func setCounterCount(s *sdk.SDK, counterName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not set Counter %s to unparseable amount %s: %s", counterName, amount, err)
	}
	log.Printf("Setting Counter %s Count to amount %d", counterName, amountInt)
	err = s.Beta().SetCounterCount(counterName, amountInt)
	if err != nil {
		log.Printf("Error setting Counter %s Count by amount %d: %s", counterName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// getCounterCapacity returns the Capacity of the given Counter as a string
func getCounterCapacity(s *sdk.SDK, counterName string) (string, error) {
	log.Printf("Retrieving Counter %s Capacity", counterName)
	count, err := s.Beta().GetCounterCapacity(counterName)
	if err != nil {
		log.Printf("Error getting Counter %s Capacity: %s", counterName, err)
		return strconv.FormatInt(count, 10), err
	}
	return "CAPACITY: " + strconv.FormatInt(count, 10) + "\n", nil
}

// setCounterCapacity returns the if the Counter was set to a new Capacity successfully or not
func setCounterCapacity(s *sdk.SDK, counterName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not set Counter %s to unparseable amount %s: %s", counterName, amount, err)
	}
	log.Printf("Setting Counter %s Capacity to amount %d", counterName, amountInt)
	err = s.Beta().SetCounterCapacity(counterName, amountInt)
	if err != nil {
		log.Printf("Error setting Counter %s Capacity to amount %d: %s", counterName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// getListCapacity returns the Capacity of the given List as a string
func getListCapacity(s *sdk.SDK, listName string) (string, error) {
	log.Printf("Retrieving List %s Capacity", listName)
	capacity, err := s.Beta().GetListCapacity(listName)
	if err != nil {
		log.Printf("Error getting List %s Capacity: %s", listName, err)
		return strconv.FormatInt(capacity, 10), err
	}
	return "CAPACITY: " + strconv.FormatInt(capacity, 10) + "\n", nil
}

// setListCapacity returns if the List was set to a new Capacity successfully or not
func setListCapacity(s *sdk.SDK, listName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not set List %s to unparseable amount %s: %s", listName, amount, err)
	}
	log.Printf("Setting List %s Capacity to amount %d", listName, amountInt)
	err = s.Beta().SetListCapacity(listName, amountInt)
	if err != nil {
		log.Printf("Error setting List %s Capacity to amount %d: %s", listName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// listContains returns true if the given value is in the given List, false otherwise
func listContains(s *sdk.SDK, listName string, value string) (string, error) {
	log.Printf("Getting List %s contains value %s", listName, value)
	ok, err := s.Beta().ListContains(listName, value)
	if err != nil {
		log.Printf("Error getting List %s contains value %s: %s", listName, value, err)
		return strconv.FormatBool(ok), err
	}
	return "FOUND: " + strconv.FormatBool(ok) + "\n", nil
}

// getListLength returns the length (number of values) of the given List as a string
func getListLength(s *sdk.SDK, listName string) (string, error) {
	log.Printf("Getting List %s length", listName)
	length, err := s.Beta().GetListLength(listName)
	if err != nil {
		log.Printf("Error getting List %s length: %s", listName, err)
		return strconv.Itoa(length), err
	}
	return "LENGTH: " + strconv.Itoa(length) + "\n", nil
}

// getListValues return the values in the given List as a comma delineated string
func getListValues(s *sdk.SDK, listName string) (string, error) {
	log.Printf("Getting List %s values", listName)
	values, err := s.Beta().GetListValues(listName)
	if err != nil {
		log.Printf("Error getting List %s values: %s", listName, err)
		return "INVALID LIST NAME", err
	}
	if len(values) > 0 {
		return "VALUES: " + strings.Join(values, ",") + "\n", nil
	}
	return "VALUES: <none>\n", nil
}

// appendListValue returns if the given value was successfuly added to the List or not
func appendListValue(s *sdk.SDK, listName string, value string) (string, error) {
	log.Printf("Appending Value %s to List %s", value, listName)
	err := s.Beta().AppendListValue(listName, value)
	if err != nil {
		log.Printf("Error appending Value %s to List %s: %s", value, listName, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// deleteListValue returns if the given value was successfuly deleted from the List or not
func deleteListValue(s *sdk.SDK, listName string, value string) (string, error) {
	log.Printf("Deleting Value %s from List %s", value, listName)
	err := s.Beta().DeleteListValue(listName, value)
	if err != nil {
		log.Printf("Error deleting Value %s to List %s: %s", value, listName, err)
		return "", err
	}
	return "SUCCESS\n", nil
}