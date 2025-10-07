package goemo

import (
	"context"
	"hash/crc64"
	"log/slog"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"unsafe"
)

var (
	pageStylesKey = "goemoPageStyles"
	hashWordsKey  = "goemoHashWords"
)

type pageStyle struct {
	ClassName   string
	SnippetSCSS string
}

type hashWords struct {
	available []string
	cache     map[string]string
	mutex     sync.Mutex
}

func (hashWords *hashWords) getWord(className string) string {
	hashWords.mutex.Lock()
	defer hashWords.mutex.Unlock()

	word, ok := hashWords.cache[className]
	if ok {
		return word
	}

	seedUint := crc64.Checksum([]byte(className), crc64.MakeTable(crc64.ECMA))
	r := rand.New(rand.NewSource(*(*int64)(unsafe.Pointer(&seedUint))))

	i := r.Intn(len(hashWords.available))
	word = hashWords.available[i]

	hashWords.available = slices.Delete(hashWords.available, i, i+1)
	hashWords.cache[className] = word

	return word
}

func InitContext(parent context.Context) context.Context {
	return context.WithValue(parent, pageStylesKey, &[]pageStyle{})
}

func UseWords(
	parent context.Context, words []string, seed string,
) context.Context {
	hashWords := hashWords{
		cache: map[string]string{},
		mutex: sync.Mutex{},
	}

	for _, word := range words {
		if len(word) == 0 {
			continue
		}

		word = strings.TrimSpace(word)
		word = strings.ToLower(word)
		word = strings.ReplaceAll(word, " ", "-")

		if !slices.Contains(hashWords.available, word) {
			hashWords.available = append(hashWords.available, word)
		}
	}

	return context.WithValue(parent, hashWordsKey, &hashWords)
}

// returns class name and injects scss into page
func SCSS(ctx context.Context, snippet string) string {
	if snippet == "" {
		return ""
	}

	pageStyles, ok := ctx.Value(pageStylesKey).(*[]pageStyle)
	if !ok {
		slog.Error("failed to get page scss from context")
		return ""
	}

	// TODO: snippet doesnt consider whitespace
	var className = hashString(snippet)

	hashWords := ctx.Value(hashWordsKey).(*hashWords)
	if hashWords != nil {
		className = hashWords.getWord(className)
	}

	for _, style := range *pageStyles {
		if style.ClassName == className {
			return className
		}
	}

	*pageStyles = append(*pageStyles, pageStyle{
		ClassName:   className,
		SnippetSCSS: snippet,
	})

	return className
}

func GetPageSCSS(ctx context.Context) string {
	pageStyles, ok := ctx.Value(pageStylesKey).(*[]pageStyle)
	if !ok {
		slog.Error("failed to get page scss from context")
		return ""
	}

	var source string

	for _, scss := range *pageStyles {
		source += "." + scss.ClassName + "{" + scss.SnippetSCSS + "}"
	}

	source = strings.TrimSpace(source)

	return source
}
