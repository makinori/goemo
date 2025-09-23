package goemo

import (
	"context"
	"log/slog"
	"strings"
)

var (
	pageStylesKey = "goemoPageStyles"
)

type pageStyle struct {
	ClassName   string
	SnippetSCSS string
}

func InitContext(parent context.Context) context.Context {
	return context.WithValue(parent, pageStylesKey, &[]pageStyle{})
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
	className := hashString(snippet)

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
