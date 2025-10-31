package emohttp

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cespare/xxhash/v2"
)

var ignoreEncoding = []string{
	"image/png",
	"image/jpg",
	"image/jpeg",
}

var (
	// go tool air proxy wont work if encoding
	DisableContentEncodingForHTML = false

	portRegexp = regexp.MustCompile(":[0-9]+$")
)

func InCommaSeperated(commaSeparated string, needle string) bool {
	if commaSeparated == "" {
		return needle == ""
	}
	for v := range strings.SplitSeq(commaSeparated, ",") {
		if needle == strings.TrimSpace(v) {
			return true
		}
	}
	return false
}

func ServeOptimized(
	w http.ResponseWriter, r *http.Request, data []byte,
	filename string, allowCache bool,
) {
	// incase it was already set
	contentType := w.Header().Get("Content-Type")

	if allowCache {
		// unset content type incase etag matches
		w.Header().Del("Content-Type")

		// etag := fmt.Sprintf(`W/"%x"`, xxhash.Sum64(data))
		etag := fmt.Sprintf(`"%x"`, xxhash.Sum64(data))

		ifMatch := r.Header.Get("If-Match")
		if ifMatch != "" {
			if !InCommaSeperated(ifMatch, etag) && !InCommaSeperated(ifMatch, "*") {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
		}

		ifNoneMatch := r.Header.Get("If-None-Match")
		if ifNoneMatch != "" {
			if InCommaSeperated(ifNoneMatch, etag) || InCommaSeperated(ifNoneMatch, "*") {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Add("ETag", etag)
	} else {
		w.Header().Add("Cache-Control", "no-store")
	}

	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
	}

	w.Header().Add("Content-Type", contentType)

	// rest is encoding related

	if slices.Contains(ignoreEncoding, contentType) ||
		(DisableContentEncodingForHTML &&
			strings.HasPrefix(contentType, "text/html")) {

		w.Write(data)
		return
	}

	var err error
	var compressed []byte
	contentEncoding := ""

	acceptEncoding := r.Header.Get("Accept-Encoding")

	if strings.Contains(acceptEncoding, "zstd") {
		contentEncoding = "zstd"
		compressed, err = EncodeZstd(data)
	} else if strings.Contains(acceptEncoding, "br") {
		contentEncoding = "br"
		compressed, err = EncodeBrotli(data)
	}

	if err != nil {
		slog.Error("failed to encode", "name", filename, "err", err.Error())
		w.Write(data)
		return
	}

	if contentEncoding == "" || len(compressed) == 0 {
		w.Write(data)
		return
	}

	if len(compressed) < len(data) {
		w.Header().Add("Content-Encoding", contentEncoding)
		w.Write(compressed)
		return
	}

	slog.Warn(
		"ineffecient compression!", "name", filename,
		"type", contentType,
	)

	w.Write(data)
}

// example usage:
// emohttp.HandleFunc("GET /{file...}", goemo.FileServerOptimized(publicFS))
func FileServerOptimized(fs fs.FS) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.PathValue("file")

		file, err := fs.Open(filename)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		data, err := io.ReadAll(file)
		if err != nil {
			slog.Error("failed to read file", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ServeOptimized(w, r, data, filename, true)
	}
}

func GetIPAddress(r *http.Request) string {
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress != "" {
		ipAddress = strings.Split(ipAddress, ",")[0]
		ipAddress = strings.TrimSpace(ipAddress)
	} else {
		ipAddress = r.RemoteAddr
	}

	ipAddress = portRegexp.ReplaceAllString(ipAddress, "")

	ipAddress = strings.TrimPrefix(ipAddress, "[")
	ipAddress = strings.TrimSuffix(ipAddress, "]")

	return ipAddress
}

func GetFullURL(r *http.Request) url.URL {
	fullUrl := *r.URL // shallow copy

	fullUrl.Scheme = r.Header.Get("X-Forwarded-Proto")
	if fullUrl.Scheme == "" {
		fullUrl.Scheme = "http"
	}
	fullUrl.Host = r.Host

	return fullUrl
}
