package emohtml

import (
	"context"
	"io"

	"github.com/makinori/goemo"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// nice! if only we can do this for goemo itself lol
type StackSCSS string

func (_ StackSCSS) Render(_ io.Writer) error {
	return nil
}

func stack(
	ctx context.Context, flexDir string, children ...Node,
) Node {
	class := goemo.SCSS(ctx, `
		display: flex;
		flex-direction: `+flexDir+`;
		gap: 8px;
	`)

	for _, node := range children {
		switch scss := node.(type) {
		case StackSCSS:
			class += " " + goemo.SCSS(ctx, string(scss))
		}
	}

	return Div(
		Class(class),
		Group(children),
	)
}
func HStack(ctx context.Context, children ...Node) Node {
	return stack(ctx, "row", children...)
}

func VStack(ctx context.Context, children ...Node) Node {
	return stack(ctx, "column", children...)
}
