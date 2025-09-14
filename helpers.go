/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
)

func GetRandomObject(input []string) string {
	if len(input) == 0 {
		return ""
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := rng.Intn(len(input))
	return input[randomIndex]
}

func GenerateUUID() (randomID string) {
	id := uuid.New()
	randomID = id.String()
	return
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func EnvVarExists(varName string) bool {
	if value, exists := os.LookupEnv(varName); !exists || value == "" {
		return false
	} else {
		return true
	}
}

// GetEnv returns the value of the environment variable 'key' or the provided fallback if not set.
func GetEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
