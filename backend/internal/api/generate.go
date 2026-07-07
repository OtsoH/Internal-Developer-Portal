// Package api contains the OpenAPI-generated server interfaces and their
// implementation. gen.go is produced from ../../api/openapi.yaml — regenerate
// with `go generate ./...` after changing the spec; never edit it by hand.
package api

//go:generate go tool oapi-codegen -config ../../api/oapi-codegen.yaml ../../api/openapi.yaml
