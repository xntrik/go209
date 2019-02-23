package go209

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

// RedisDefaultExpiration is the default period of time a redis state should last for
// slack has a 30 min window for interactive messages and the response_url
// even though we don't use the response_url, let's set the timeout slightly shorter
//
// @TODO: Should this be much much shorter, like, 5 minutes?
// How long is an interaction meant to take?
const RedisDefaultExpiration = "29m"

// RedisSubTermExpiration is the default period of time a redis state should last when handling sub-term matching
const RedisSubTermExpiration = "5m"

// newSubTermState takes the user and the search term, saving the state
// This occurs at the start of a sub-term word search
func newSubTermState(db *redis.Client, redKey, searchTerm string) error {
	err := db.HSet(redKey, "searchTerm", searchTerm).Err()
	if err != nil {
		return fmt.Errorf("Error setting new hash: %s", err)
	}

	dur, err := time.ParseDuration(RedisSubTermExpiration)
	if err != nil {
		return fmt.Errorf("Couldn't parse duration for redis expiry: %s", err)
	}

	err = db.Expire(redKey, dur).Err()
	if err != nil {
		return fmt.Errorf("Error expiring hash: %s", err)
	}

	return nil
}

// newState takes the user and interaction and saves the state
// This occurs at the start of an interaction
func newState(db *redis.Client, redKey, user, username string, interaction *Interaction) error {
	err := db.HSet(redKey, "interaction", interaction.InteractionID).Err()
	if err != nil {
		return fmt.Errorf("Error setting new hash: %s", err)
	}

	dur, err := time.ParseDuration(RedisDefaultExpiration)
	if err != nil {
		return fmt.Errorf("Couldn't parse duration for redis expiry: %s", err)
	}

	err = db.Expire(redKey, dur).Err()
	if err != nil {
		return fmt.Errorf("Error expiring hash: %s", err)
	}

	err = db.HSet(redKey, "stop_word", interaction.StopWord).Err()
	if err != nil {
		return fmt.Errorf("Error adding new key to hash: %s", err)
	}

	err = db.HSet(redKey, "userid", user).Err()
	if err != nil {
		return fmt.Errorf("Error adding new key to hash: %s", err)
	}

	err = db.HSet(redKey, "username", username).Err()
	if err != nil {
		return fmt.Errorf("Error adding new key to hash: %s", err)
	}

	err = db.HSet(redKey, "type", interaction.Type).Err()
	if err != nil {
		return fmt.Errorf("Error adding new key to hash: %s", err)
	}

	err = db.HSet(redKey, "next_interaction", interaction.NextInteraction).Err()
	if err != nil {
		return fmt.Errorf("Error adding new key to hash: %s", err)
	}

	return nil
}

// updateState occurs within a set of interactions, and updates the redis state
func updateState(db *redis.Client, redKey string, interaction *Interaction) error {
	err := db.HSet(redKey, "interaction", interaction.InteractionID).Err()
	if err != nil {
		return fmt.Errorf("Error updating hash: %s", err)
	}

	err = db.HSet(redKey, "type", interaction.Type).Err()
	if err != nil {
		return fmt.Errorf("Error updating hash: %s", err)
	}

	err = db.HSet(redKey, "next_interaction", interaction.NextInteraction).Err()
	if err != nil {
		return fmt.Errorf("Error updating hash: %s", err)
	}

	return nil
}
