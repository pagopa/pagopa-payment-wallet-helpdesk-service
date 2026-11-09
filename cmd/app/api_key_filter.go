package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
)

func newAPIKeyFilter() (gin.HandlerFunc, error) {
	const pathsEnv = "SECURITY_API_KEYS_SECURED_PATHS"
	paths := strings.Split(os.Getenv(pathsEnv), ",")
	for i, path := range paths {
		path = strings.TrimSpace(path)
		if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?# \t\r\n") {
			return nil, fmt.Errorf("%s must contain non-empty, comma-separated absolute path prefixes", pathsEnv)
		}
		paths[i] = path
	}
	var keys [2][sha256.Size]byte
	for i, name := range []string{"SECURITY_API_KEYS_PRIMARY", "SECURITY_API_KEYS_SECONDARY"} {
		key := os.Getenv(name)
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("%s must be set and non-blank", name)
		}
		keys[i] = sha256.Sum256([]byte(key))
	}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if c.Request.Method == http.MethodGet && path == "/info" {
			c.Next()
			return
		}
		for _, prefix := range paths {
			if !strings.HasPrefix(path, prefix) {
				continue
			}
			headers := c.Request.Header.Values("x-api-key")
			if len(headers) == 1 && headers[0] != "" {
				// Compare fixed-size hashes so both keys are checked in constant time.
				candidate := sha256.Sum256([]byte(headers[0]))
				match := subtle.ConstantTimeCompare(candidate[:], keys[0][:]) |
					subtle.ConstantTimeCompare(candidate[:], keys[1][:])
				if match == 1 {
					c.Next()
					return
				}
			}
			log.Print("Unauthorized request: missing or invalid x-api-key header")
			status := int32(http.StatusUnauthorized)
			title := http.StatusText(http.StatusUnauthorized)
			detail := "Missing or invalid x-api-key header"
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.ProblemJson{
				Status: &status, Title: &title, Detail: &detail,
			})
			return
		}
		c.Next()
	}, nil
}
