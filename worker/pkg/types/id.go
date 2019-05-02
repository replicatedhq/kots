package types

import "math/rand"

func GenerateID() string {
	var letters = "abcdefghijklmnopqrstuvwxyz0123456789"

	id := make([]byte, 32)
	for i := range id {
		id[i] = letters[rand.Intn(len(letters))]
	}

	return string(id)
}
