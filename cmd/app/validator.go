package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
)

func newRequestValidator(path string) (gin.HandlerFunc, error) {
	spec, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI specification %s: %w", path, err)
	}
	// Deployment URLs document the API, but must not restrict local or probe hosts.
	spec.Servers = nil
	return ginmiddleware.OapiRequestValidator(spec), nil
}
