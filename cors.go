package main

import (
	"net/http"
	"strconv"
	"strings"
)

// Inspiration for this code was taken from labstack's echo middleware

const (
	headerOrigin                        = "Origin"
	headerVary                          = "Vary"
	headerAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	headerAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	headerAccessControlRequestMethod    = "Access-Control-Request-Method"
	headerAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	headerAccessControlMaxAge           = "Access-Control-Max-Age"
	headerAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
)

// CorsConfig is the configuration needed to set up cors
type CorsConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig is the default configuration for CORS
var DefaultCORSConfig = CorsConfig{
	AllowOrigins: []string{"*"},
	AllowMethods: []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPut,
		http.MethodPatch,
		http.MethodPost,
		http.MethodDelete,
	},
}

// AllowCors returns a function returning a function
// that wraps the handler with configurations required to
// allow Cross-Origin Resource Sharing(CORS)
func AllowCors(cc CorsConfig) func(http.Handler) http.Handler {

	// use settings here
	if len(cc.AllowOrigins) == 0 {
		cc.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	if len(cc.AllowMethods) == 0 {
		cc.AllowMethods = DefaultCORSConfig.AllowMethods
	}

	allowMethods := strings.Join(cc.AllowMethods, ",")
	allowHeaders := strings.Join(cc.AllowHeaders, ",")

	// todo(uz) use AllowHeaders
	maxAge := strconv.Itoa(cc.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// modify w
			origin := r.Header.Get(headerOrigin)
			allowOrigin := ""

			// allow the origin making the request if *
			// or if the origin is in the list
			for _, o := range cc.AllowOrigins {
				if o == "*" || o == origin {
					allowOrigin = o
					break
				}
			}

			// handle regular request
			if r.Method != http.MethodOptions {
				w.Header().Add(headerVary, headerOrigin)
				w.Header().Set(headerAccessControlAllowOrigin, allowOrigin)
				if cc.AllowCredentials {
					w.Header().Set(headerAccessControlAllowCredentials, "true")
				}
				next.ServeHTTP(w, r)
				return
			}

			// handle CORS preflight request
			w.Header().Add(headerVary, headerOrigin)
			w.Header().Add(headerVary, headerAccessControlRequestMethod)
			w.Header().Set(headerAccessControlAllowOrigin, allowOrigin)
			w.Header().Set(headerAccessControlAllowMethods, allowMethods)
			if cc.AllowCredentials {
				w.Header().Set(headerAccessControlAllowCredentials, "true")
			}
			if allowHeaders != "" {
				w.Header().Set(headerAccessControlAllowHeaders, allowHeaders)
			} else {
				h := r.Header.Get(headerAccessControlAllowHeaders)
				if h != "" {
					w.Header().Set(headerAccessControlAllowHeaders, h)
				}
			}
			if cc.MaxAge > 0 {
				w.Header().Set(headerAccessControlMaxAge, maxAge)
			}
			next.ServeHTTP(w, r)
		})
	}
}
