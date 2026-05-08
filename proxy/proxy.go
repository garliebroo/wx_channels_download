package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
)

// VideoInfo holds metadata about a captured WeChat Channels video
type VideoInfo struct {
	URL      string
	FileSize int64
	Headers  http.Header
}

// Handler is the callback invoked when a video URL is intercepted
type Handler func(info VideoInfo)

// Server wraps a goproxy instance configured to intercept WeChat Channels video requests
type Server struct {
	proxy   *goproxy.ProxyHttpServer
	port    int
	handler Handler
}

// videoURLPattern matches the CDN URLs used by WeChat Channels for video delivery
var videoURLPattern = regexp.MustCompile(`(https?://[^"\s]+\.mp4[^"\s]*|https?://finder\.video\.qq\.com[^"\s]+)`)

// NewServer creates a new proxy server that will call handler whenever a
// WeChat Channels video request is detected.
func NewServer(port int, handler Handler) *Server {
	p := goproxy.NewProxyHttpServer()
	// turned off verbose logging - it's way too noisy during normal use
	p.Verbose = false

	// Allow the proxy to intercept HTTPS traffic
	p.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// Disable TLS verification so we can inspect HTTPS responses
	p.Tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}

	s := &Server{
		proxy:   p,
		port:    port,
		handler: handler,
	}

	s.registerHandlers()
	return s
}

// registerHandlers sets up the request/response interception rules.
func (s *Server) registerHandlers() {
	// Intercept responses that look like WeChat Channels video streams
	s.proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil {
			return resp
		}

		url := ctx.Req.URL.String()
		contentType := resp.Header.Get("Content-Type")

		if isVideoResponse(url, contentType) {
			info := VideoInfo{
				URL:      url,
				FileSize: resp.ContentLength,
				Headers:  resp.Header.Clone(),
			}
			// only log if we actually have a size, avoids spammy -1 entries
			if resp.ContentLength > 0 {
				log.Printf("[proxy] captured video URL: %s (size: %d bytes)", url, resp.ContentLength)
			} else {
				// size unknown usually means chunked transfer encoding - still worth capturing
				// skipping the log line here to keep output clean
				_ = info // suppress unused warning if handler is nil
			}
			if s.handler != nil {
				s.handler(info)
			}
		}

		return resp
	})
}

// isVideoResponse returns true when the URL or Content-Type indicates a video.
func isVideoResponse(url, contentType string) bool {
	if strings.Contains(contentType, "video/") {
		return true
	}
	if videoURLPattern.MatchString(url) {
		return true
	}
	// WeChat Channels specific domain check
	if strings.Contains(url, "finder.video.qq.com") ||
		strings.Contains(url, "channels.weixin.qq.com") {
		return true
	}
	return false
}

// Start begins listening on the configured port. This call blocks until the
// server encounters an error.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[proxy] starting proxy server on %s", addr)
	return http.ListenAndServe(addr, s.proxy)
}
