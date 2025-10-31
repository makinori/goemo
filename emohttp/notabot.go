package emohttp

import (
	_ "embed"
	"errors"
	"net/http"
	"net/url"
)

var (
	//go:embed 1x1.gif
	gif1x1 []byte
)

func NotABotURLQuery(r *http.Request) string {
	v := url.Values{}
	v.Add("p", r.URL.Path)

	ref := r.Referer()
	if ref != "" {
		v.Add("r", r.Referer())
	}

	return v.Encode()
}

type notABot struct {
	Path string
	Ref  string
}

func NotABotDecode(query string) (notABot, error) {
	v, err := url.ParseQuery(query)
	if err != nil {
		return notABot{}, err
	}

	n := notABot{
		Path: v.Get("p"),
		Ref:  v.Get("r"),
	}

	if n.Path == "" {
		return notABot{}, errors.New("missing path")
	}

	return n, nil
}

// use with `http.HandleFunc("GET /notabot.gif", ...)`
func HandleNotABotGif(
	onRequest func(r *http.Request),
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// respond immediately
		w.Header().Add("Cache-Control", "no-store")
		w.Write(gif1x1)

		go onRequest(r)
	}
}
