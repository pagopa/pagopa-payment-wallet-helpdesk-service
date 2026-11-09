package main

import (
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
)

func newRequestValidator(path string) (gin.HandlerFunc, error) {
	spec, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI specification %s: %w", path, err)
	}
	// Deployment URLs document the API, but must not restrict local or probe hosts.
	spec.Servers = nil
	return ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{
		ErrorHandler: func(c *gin.Context, message string, status int) {
			code := int32(status)
			title := http.StatusText(status)
			c.AbortWithStatusJSON(status, api.ProblemJson{
				Status: &code, Title: &title, Detail: &message,
			})
		},
		Options: openapi3filter.Options{
			// The API key middleware authenticates requests before schema validation.
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}), nil
}
