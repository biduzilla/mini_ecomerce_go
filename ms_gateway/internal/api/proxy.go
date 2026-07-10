package api

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/sony/gobreaker"
)

func ProxyWithCircuitBreaker(targetURL string, circuitName string, fallbackMsg string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("Invalid target URL: " + err.Error())
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)

			incomingPath := pr.In.URL.Path
			if strings.HasPrefix(incomingPath, "/api/") {
				pr.Out.URL.Path = "/v1/" + strings.TrimPrefix(incomingPath, "/api/")
			} else {
				pr.Out.URL.Path = incomingPath
			}

			pr.SetXForwarded()
		},
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        circuitName,
		MaxRequests: 5,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf("⚠️ Circuit Breaker '%s' mudou de %s para %s", name, from, to)
		},
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturer := &statusCapture{ResponseWriter: w, statusCode: http.StatusOK}

		_, err := cb.Execute(func() (interface{}, error) {
			proxy.ServeHTTP(capturer, r)

			if capturer.statusCode >= 500 {
				return nil, fmt.Errorf("downstream service failed with status %d", capturer.statusCode)
			}
			return nil, nil
		})

		if err != nil {
			if capturer.statusCode == http.StatusOK {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(fallbackMsg))
			}
		}
	})
}

type statusCapture struct {
	http.ResponseWriter
	statusCode int
}

func (s *statusCapture) WriteHeader(code int) {
	s.statusCode = code
	s.ResponseWriter.WriteHeader(code)
}
