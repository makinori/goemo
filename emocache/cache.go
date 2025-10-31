package emocache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

var (
	// to be set in the init function
	currentCacheDir = ""
)

type cacheData[T any] struct {
	Data    T         `json:"data"`
	Updated time.Time `json:"updated"`
	Expires time.Time `json:"expires"`
}

func setCache[T any](key string, data cacheData[T]) error {
	if currentCacheDir == "" {
		return errors.New("cache dir not set")
	}

	os.Mkdir(currentCacheDir, 0755)

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = os.WriteFile(
		filepath.Join(currentCacheDir, key+".json"),
		jsonBytes, 0644,
	)
	if err != nil {
		return err
	}

	return nil
}

func getCache[T any](key string) (*cacheData[T], error) {
	if currentCacheDir == "" {
		return nil, errors.New("cache dir not set")
	}

	bytes, err := os.ReadFile(filepath.Join(currentCacheDir, key+".json"))
	if err != nil {
		return nil, err
	}

	var cacheData cacheData[T]

	err = json.Unmarshal(bytes, &cacheData)
	if err != nil {
		return nil, err
	}

	if time.Now().After(cacheData.Expires) {
		os.Remove("cache/" + key + ".json")
		return nil, errors.New("cache data expired")
	}

	return &cacheData, nil
}
