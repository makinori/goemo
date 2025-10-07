package goemo

import (
	"context"
	"hash/crc64"
	"log/slog"
	"math/rand"
	"slices"
	"strings"
	"unsafe"
)

var (
	pageStylesKey = "goemoPageStyles"

	usingHashWords bool
	hashWords      []string
	hashWordsMap   = map[string]uint{}
)

type pageStyle struct {
	ClassName   string
	SnippetSCSS string
}

func InitContext(parent context.Context) context.Context {
	return context.WithValue(parent, pageStylesKey, &[]pageStyle{})
}

func UseWords(words []string, seedString string) {
	usingHashWords = true
	hashWords = []string{}

	for _, word := range words {
		if len(word) == 0 {
			continue
		}

		word = strings.TrimSpace(word)
		word = strings.ToLower(word)
		word = strings.ReplaceAll(word, " ", "-")

		if !slices.Contains(hashWords, word) {
			hashWords = append(hashWords, word)
		}
	}

	seedUint := crc64.Checksum([]byte(seedString), crc64.MakeTable(crc64.ECMA))

	r := rand.New(rand.NewSource(*(*int64)(unsafe.Pointer(&seedUint))))

	for i := len(hashWords) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		hashWords[i], hashWords[j] = hashWords[j], hashWords[i]
	}
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

	if usingHashWords {
		hashWordIndex, exists := hashWordsMap[className]
		if !exists {
			hashWordIndex = uint(len(hashWordsMap))
			hashWordsMap[className] = hashWordIndex
		}
		className = hashWords[hashWordIndex]
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
