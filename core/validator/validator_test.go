package validator

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateObject(t *testing.T) {
	schema := Object(map[string]*Schema{
		"name": String(),
		"age":  Integer(),
	}, "name")

	tests := []struct {
		name  string
		json  string
		valid bool
	}{
		{"valid", `{"name":"Alice","age":30}`, true},
		{"missing required", `{"age":30}`, false},
		{"wrong type age", `{"name":"Alice","age":"30"}`, false},
		{"extra field ok", `{"name":"Alice","extra":true}`, true},
		{"invalid json", `{bad`, false},
		{"not an object", `"hello"`, false},
		{"integer as float", `{"name":"Bob","age":30.0}`, true},
		{"wrong type name", `{"name":123}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(strings.NewReader(tt.json))
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestValidateString(t *testing.T) {
	schema := String()
	if err := schema.validate("hello"); err != nil {
		t.Errorf("expected valid: %v", err)
	}
	if err := schema.validate(123); err == nil {
		t.Errorf("expected error for number")
	}
}

func TestValidateNumber(t *testing.T) {
	schema := Number()
	if err := schema.validate(3.14); err != nil {
		t.Errorf("expected valid: %v", err)
	}
	if err := schema.validate("3.14"); err == nil {
		t.Errorf("expected error for string")
	}
}

func TestValidateInteger(t *testing.T) {
	schema := Integer()
	if err := schema.validate(42.0); err != nil {
		t.Errorf("expected valid: %v", err)
	}
	if err := schema.validate(3.14); err == nil {
		t.Errorf("expected error for float")
	}
	if err := schema.validate("42"); err == nil {
		t.Errorf("expected error for string")
	}
}

func TestValidateBoolean(t *testing.T) {
	schema := Boolean()
	if err := schema.validate(true); err != nil {
		t.Errorf("expected valid: %v", err)
	}
	if err := schema.validate("true"); err == nil {
		t.Errorf("expected error for string")
	}
}

func TestAdditionalPropertiesFalse(t *testing.T) {
	allowAdditional := false
	schema := &Schema{
		Type:                 jsonObject,
		Properties:           map[string]*Schema{"name": String()},
		AdditionalProperties: &allowAdditional,
	}
	err := schema.validate(map[string]any{"name": "Alice", "extra": "nope"})
	if err == nil {
		t.Errorf("expected error for extra field")
	}
}

func TestSchemasConcrete(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		json   string
		valid  bool
	}{
		{"CreateScript valid", CreateScriptSchema, `{"name":"test","label":"Test"}`, true},
		{"CreateScript missing name", CreateScriptSchema, `{"label":"Test"}`, false},
		{"UpdateScript partial", UpdateScriptSchema, `{"name":"new"}`, true},
		{"UpdateScript empty", UpdateScriptSchema, `{}`, true},
		{"DebugScript valid", DebugScriptSchema, `{"skip":0}`, true},
		{"DebugScript empty", DebugScriptSchema, `{}`, true},
		{"SetBreakpoint valid", SetBreakpointSchema, `{"line":10,"enabled":true}`, true},
		{"SetBreakpoint missing line", SetBreakpointSchema, `{"enabled":true}`, false},
		{"RunSQL valid", RunSQLSchema, `{"sql":"SELECT 1"}`, true},
		{"RunSQL missing sql", RunSQLSchema, `{}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(bytes.NewReader([]byte(tt.json)))
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}


