package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type jsonType string

const (
	jsonObject  jsonType = "object"
	jsonString  jsonType = "string"
	jsonNumber  jsonType = "number"
	jsonInteger jsonType = "integer"
	jsonBoolean jsonType = "boolean"
)

type Schema struct {
	Type                 jsonType            `json:"type"`
	Properties           map[string]*Schema  `json:"properties,omitempty"`
	Required             []string            `json:"required,omitempty"`
	AdditionalProperties *bool               `json:"additionalProperties,omitempty"`
}

func Object(props map[string]*Schema, required ...string) *Schema {
	return &Schema{
		Type:       jsonObject,
		Properties: props,
		Required:   required,
	}
}

func String() *Schema  { return &Schema{Type: jsonString} }
func Number() *Schema  { return &Schema{Type: jsonNumber} }
func Integer() *Schema { return &Schema{Type: jsonInteger} }
func Boolean() *Schema { return &Schema{Type: jsonBoolean} }

func (s *Schema) Validate(r io.Reader) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := s.validate(v); err != nil {
		return err
	}
	return nil
}

func (s *Schema) validate(v any) error {
	switch s.Type {
	case jsonObject:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object, got %T", v)
		}
		required := make(map[string]bool)
		for _, r := range s.Required {
			required[r] = true
		}
		for key := range required {
			if _, exists := m[key]; !exists {
				return fmt.Errorf("missing required field: %q", key)
			}
		}
		for key, val := range m {
			propSchema, exists := s.Properties[key]
			if !exists {
				if s.AdditionalProperties != nil && !*s.AdditionalProperties {
					return fmt.Errorf("unknown field: %q", key)
				}
				continue
			}
			if err := propSchema.validate(val); err != nil {
				return fmt.Errorf("%q: %w", key, err)
			}
		}
	case jsonString:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("expected string, got %T", v)
		}
	case jsonNumber:
		if _, ok := v.(float64); !ok {
			return fmt.Errorf("expected number, got %T", v)
		}
	case jsonInteger:
		n, ok := v.(float64)
		if !ok {
			return fmt.Errorf("expected integer, got %T", v)
		}
		if n != float64(int64(n)) {
			return fmt.Errorf("expected integer, got %v", n)
		}
	case jsonBoolean:
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", v)
		}
	}
	return nil
}

func Middleware(schema *Schema) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to read body: %v", err))
				return
			}
			r.Body.Close()
			if len(body) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if err := schema.Validate(bytes.NewReader(body)); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
