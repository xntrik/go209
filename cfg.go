package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// loadDotEnv loads your config from the .env file
func loadDotEnv() {
	if _, err := os.Stat(".env"); err != nil {
		return
	}

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

// getSlackToken fetches the token used to authenticate to slack
func getSlackToken() (string, error) {
	value := os.Getenv("SLACK_TOKEN")

	if len(value) == 0 {
		return "", fmt.Errorf("Missing SLACK_TOKEN ENV variable. Check --help for options")
	}

	return value, nil
}

// getSlackSigningSecret fetches the token used to validate messages from slack
func getSlackSigningSecret() (string, error) {
	value := os.Getenv("SLACK_SIGNING_SECRET")

	if len(value) == 0 {
		return "", fmt.Errorf("Missing SLACK_SIGNING_SECRET ENV variable. Check --help for options")
	}

	return value, nil
}

// getRedisAddr fetches the address (host:port) to connect to redis
func getRedisAddr() (string, error) {
	value := os.Getenv("REDIS_ADDR")

	if len(value) == 0 {
		return "", fmt.Errorf("Missing REDIS_ADDR ENV variable. Check --help for options")
	}

	return value, nil
}

// getRedisPwd fetches the password used to connect to redis (defaults to "")
func getRedisPwd() string {
	return os.Getenv("REDIS_PWD")
}

// getRedisDB fetches the redis DB instance to connect to (defaults to 0)
func getRedisDB() int {
	value := os.Getenv("REDIS_DB")

	if len(value) == 0 {
		return 0
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return i
}

// getRulesFileLocation fetches the address of the rules.json to load
func getRulesFileLocation() string {
	value := os.Getenv("JSON_RULES")

	if len(value) == 0 {
		return "rules.json"
	}

	return value

}

// getWebListen fetches the listening server address for the web server
func getWebListen() string {
	value := os.Getenv("WEB_ADDR")

	if len(value) == 0 {
		return ":8000"
	}

	return value
}
