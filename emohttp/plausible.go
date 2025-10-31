package emohttp

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

var (
	PlausibleDisable bool
	PlausibleDebug   bool

	PlausibleDomain  string
	PlausibleBaseURL string

	ignorePlausibleUserAgents = []string{
		"gatus", // health checks
		"curl",
	}
)

func PlausibleEventFromNotABot(incomingReq *http.Request) bool {
	if PlausibleDisable {
		return false
	}

	if PlausibleDomain == "" || PlausibleBaseURL == "" {
		slog.Error("plausible domain or base url not set")
		return false
	}

	userAgent := incomingReq.Header.Get("User-Agent")
	lowerUserAgent := strings.ToLower(userAgent)

	for _, ignoreUA := range ignorePlausibleUserAgents {
		if strings.Contains(lowerUserAgent, ignoreUA) {
			return false
		}
	}

	notabot, err := NotABotDecode(incomingReq.URL.RawQuery)
	if err != nil {
		slog.Error("failed to get notabot data", "err", err.Error())
		return false
	}

	// https://plausible.io/docs/events-api

	fullUrl := GetFullURL(incomingReq)
	fullUrl.Path = notabot.Path
	fullUrl.RawQuery = ""

	body, err := json.Marshal(map[string]string{
		"name":     "pageview",
		"url":      fullUrl.String(),
		"domain":   PlausibleDomain,
		"referrer": notabot.Ref,
	})

	if err != nil {
		slog.Error("failed to marshal json for plausible", "err", err.Error())
		return false
	}

	plausibleReq, err := http.NewRequest(
		"POST", PlausibleBaseURL+"/api/event",
		bytes.NewReader(body),
	)

	plausibleReq.Header.Add("User-Agent", userAgent)
	plausibleReq.Header.Add("Content-Type", "application/json")

	ipAddress := GetIPAddress(incomingReq)
	if ipAddress != "" {
		// traefik will rewrite this
		// plausibleReq.Header.Add("X-Forwarded-For", ipAddress)
		//https://github.com/plausible/analytics/blob/master/lib/plausible_web/remote_ip.ex
		plausibleReq.Header.Add("X-Plausible-IP", ipAddress)
	}

	if PlausibleDebug {
		// isn't even in the plausible code. docs need to be updated
		plausibleReq.Header.Add("X-Debug-Request", "true")
		slog.Info(
			"plausible:\n" +
				"  data: " + string(body) + "\n" +
				"  ip: " + ipAddress + "\n" +
				"  ua: " + userAgent,
		)
	}

	client := http.Client{}
	res, err := client.Do(plausibleReq)
	if err != nil {
		slog.Error("failed to send plausible event", "err", err.Error())
		return false
	}

	if PlausibleDebug {
		resData, err := io.ReadAll(res.Body)
		if err != nil {
			slog.Error("failed to read plausible res body", "err", err.Error())
		}
		slog.Info("res: " + string(resData))
	}

	return true
}
