package goemo

import (
	"fmt"
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
	HTTPDisableContentEncoding = false
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

func HTTPServeOptimized(
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

	if HTTPDisableContentEncoding ||
		slices.Contains(ignoreEncoding, contentType) {
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

func HTTPFileServerOptimized(fs fs.FS) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.PathValue("file")

		file, err := fs.Open(filename)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		stat, err := file.Stat()
		if err != nil {
			slog.Error("failed to file stat", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		data := make([]byte, stat.Size())

		_, err = file.Read(data)
		if err != nil {
			slog.Error("failed to read file", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		HTTPServeOptimized(w, r, data, filename, true)
	}
}

var portRegexp = regexp.MustCompile(":[0-9]+$")

func HTTPGetIPAddress(r *http.Request) string {
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress != "" {
		ipAddress = strings.Split(ipAddress, ",")[0]
		ipAddress = strings.TrimSpace(ipAddress)
	} else {
		ipAddress = portRegexp.ReplaceAllString(r.RemoteAddr, "")
	}

	ipAddress = strings.TrimPrefix(ipAddress, "[")
	ipAddress = strings.TrimSuffix(ipAddress, "]")

	return ipAddress
}

func HTTPGetFullURL(r *http.Request) url.URL {
	fullUrl := *r.URL // shallow copy

	fullUrl.Scheme = r.Header.Get("X-Forwarded-Proto")
	if fullUrl.Scheme == "" {
		fullUrl.Scheme = "http"
	}
	fullUrl.Host = r.Host

	return fullUrl
}
