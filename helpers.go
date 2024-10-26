/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"math/rand"
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
