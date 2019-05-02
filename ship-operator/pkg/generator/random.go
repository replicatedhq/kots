package generator

import "math/rand"

func GenerateID(size int) string {
	var letters = "abcdefghijklmnopqrstuvwxyz0123456789"

	id := make([]byte, size)
	for i := range id {
		id[i] = letters[rand.Intn(len(letters))]
	}

	return string(id)
}
