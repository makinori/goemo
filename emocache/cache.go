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

func setCache(key string, data any, expiresAt time.Time) error {
	if currentCacheDir == "" {
		return errors.New("cache dir not set")
	}

	os.Mkdir(currentCacheDir, 0755)

	finalData := map[string]any{
		"expiresAt": expiresAt,
		"data":      data,
	}

	jsonBytes, err := json.Marshal(finalData)
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

func getCache[T any](key string, output *T) error {
	if currentCacheDir == "" {
		return errors.New("cache dir not set")
	}

	bytes, err := os.ReadFile(filepath.Join(currentCacheDir, key+".json"))
	if err != nil {
		return err
	}

	var cacheData struct {
		ExpiresAt time.Time `json:"expiresAt"`
		Data      T         `json:"data"`
	}

	err = json.Unmarshal(bytes, &cacheData)
	if err != nil {
		return err
	}

	if time.Now().After(cacheData.ExpiresAt) {
		os.Remove("cache/" + key + ".json")
		return errors.New("cache data expired")
	}

	*output = cacheData.Data

	return nil
}
