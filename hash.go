package goemo

import (
	"github.com/cespare/xxhash/v2"
)

const base52 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func hashBytes(data []byte) string {
	hash := xxhash.Sum64(data)
	if hash == 0 {
		return string(base52[0])
	}

	var name string
	for hash > 0 {
		name = string(base52[hash%52]) + name
		hash /= 52
	}

	return name
}

func hashString(data string) string {
	return hashBytes([]byte(data))
}
