// Package graphql provides GraphQL parsing and transformation for the Dagster proxy.
package graphql

import (
	"encoding/json"
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// GraphQLRequest represents an incoming GraphQL request.
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// ParsedOperation represents a parsed GraphQL operation.
type ParsedOperation struct {
	Type      OperationType
	Name      string
	Fields    []FieldSelection
	Variables map[string]interface{}
	RawQuery  string
}

// FieldSelection represents a selected field in a GraphQL query.
type FieldSelection struct {
	Name      string
	Alias     string
	Arguments map[string]interface{}
	Fields    []FieldSelection
}

// Parser parses GraphQL requests.
type Parser struct{}

// NewParser creates a new GraphQL parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a GraphQL request body and returns the parsed operation.
func (p *Parser) Parse(body []byte) (*ParsedOperation, error) {
	// Parse JSON request
	var req GraphQLRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Parse the GraphQL query
	doc, err := parser.ParseQuery(&ast.Source{Input: req.Query})
	if err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL query: %w", err)
	}

	// Find the operation to execute
	var operation *ast.OperationDefinition
	for _, op := range doc.Operations {
		if req.OperationName == "" || op.Name == req.OperationName {
			operation = op
			break
		}
	}

	if operation == nil {
		return nil, fmt.Errorf("operation not found: %s", req.OperationName)
	}

	// Convert to parsed operation
	parsed := &ParsedOperation{
		Type:      toOperationType(operation.Operation),
		Name:      operation.Name,
		Variables: req.Variables,
		RawQuery:  req.Query,
		Fields:    p.extractFields(operation.SelectionSet, doc, req.Variables),
	}

	return parsed, nil
}

// extractFields extracts field selections from a selection set.
func (p *Parser) extractFields(selSet ast.SelectionSet, doc *ast.QueryDocument, vars map[string]interface{}) []FieldSelection {
	var fields []FieldSelection

	for _, sel := range selSet {
		switch s := sel.(type) {
		case *ast.Field:
			field := FieldSelection{
				Name:      s.Name,
				Alias:     s.Alias,
				Arguments: p.extractArguments(s.Arguments, vars),
			}
			if s.SelectionSet != nil {
				field.Fields = p.extractFields(s.SelectionSet, doc, vars)
			}
			fields = append(fields, field)

		case *ast.FragmentSpread:
			// Find and inline fragment
			for _, frag := range doc.Fragments {
				if frag.Name == s.Name {
					fields = append(fields, p.extractFields(frag.SelectionSet, doc, vars)...)
					break
				}
			}

		case *ast.InlineFragment:
			fields = append(fields, p.extractFields(s.SelectionSet, doc, vars)...)
		}
	}

	return fields
}

// extractArguments extracts argument values, resolving variables.
func (p *Parser) extractArguments(args ast.ArgumentList, vars map[string]interface{}) map[string]interface{} {
	if len(args) == 0 {
		return nil
	}

	result := make(map[string]interface{})
	for _, arg := range args {
		result[arg.Name] = p.resolveValue(arg.Value, vars)
	}
	return result
}

// resolveValue resolves a GraphQL value, handling variables and nested structures.
func (p *Parser) resolveValue(value *ast.Value, vars map[string]interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch value.Kind {
	case ast.Variable:
		if vars != nil {
			return vars[value.Raw]
		}
		return nil

	case ast.IntValue, ast.FloatValue:
		return value.Raw

	case ast.StringValue, ast.EnumValue:
		return value.Raw

	case ast.BooleanValue:
		return value.Raw == "true"

	case ast.NullValue:
		return nil

	case ast.ListValue:
		list := make([]interface{}, len(value.Children))
		for i, child := range value.Children {
			list[i] = p.resolveValue(child.Value, vars)
		}
		return list

	case ast.ObjectValue:
		obj := make(map[string]interface{})
		for _, child := range value.Children {
			obj[child.Name] = p.resolveValue(child.Value, vars)
		}
		return obj

	default:
		return value.Raw
	}
}

// toOperationType converts ast operation type to our OperationType.
func toOperationType(op ast.Operation) OperationType {
	switch op {
	case ast.Query:
		return OperationQuery
	case ast.Mutation:
		return OperationMutation
	case ast.Subscription:
		return OperationSubscription
	default:
		return OperationQuery
	}
}

// GetTopLevelFields returns the names of top-level fields in the operation.
func (p *ParsedOperation) GetTopLevelFields() []string {
	var names []string
	for _, f := range p.Fields {
		names = append(names, f.Name)
	}
	return names
}

// HasField checks if the operation contains a specific field.
func (p *ParsedOperation) HasField(name string) bool {
	for _, f := range p.Fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

// GetField returns a specific field by name.
func (p *ParsedOperation) GetField(name string) *FieldSelection {
	for i := range p.Fields {
		if p.Fields[i].Name == name {
			return &p.Fields[i]
		}
	}
	return nil
}

// GetFieldArgument returns an argument value from a field.
func (f *FieldSelection) GetFieldArgument(name string) interface{} {
	if f.Arguments == nil {
		return nil
	}
	return f.Arguments[name]
}

// GetStringArgument returns a string argument value.
func (f *FieldSelection) GetStringArgument(name string) string {
	if val := f.GetFieldArgument(name); val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// GetNestedString extracts a string from a nested path like "selector.pipelineName".
func GetNestedValue(data map[string]interface{}, path string) interface{} {
	// Simple path parsing - split by dots
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	var value interface{} = data
	for _, part := range parts {
		if m, ok := value.(map[string]interface{}); ok {
			value = m[part]
		} else {
			return nil
		}
	}
	return value
}
