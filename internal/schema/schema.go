package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	devlaunchassets "github.com/mahmoud-nn/devlaunch"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	manifestSchemaAsset = "skills/devlaunch/references/manifest.v1.schema.json"
	stateSchemaAsset    = "skills/devlaunch/references/state.v1.schema.json"
	manifestSchemaURL   = "devlaunch://manifest.v1.schema.json"
	stateSchemaURL      = "devlaunch://state.v1.schema.json"
)

type ValidationIssue struct {
	Path     string
	Code     string
	Message  string
	Expected string
	Actual   string
}

type ValidationReport struct {
	DocumentType string
	Version      int
	SchemaRef    string
	Valid        bool
	Issues       []ValidationIssue
}

type ValidationError struct {
	Report ValidationReport
}

func (e ValidationError) Error() string {
	if len(e.Report.Issues) == 0 {
		return fmt.Sprintf("%s validation failed", e.Report.DocumentType)
	}
	first := e.Report.Issues[0]
	return fmt.Sprintf("%s validation failed at %s: %s", e.Report.DocumentType, first.Path, first.Message)
}

var (
	compileOnce    sync.Once
	compileErr     error
	manifestSchema *jsonschema.Schema
	stateSchema    *jsonschema.Schema
)

func ValidateManifestBytes(data []byte) (ValidationReport, error) {
	report := ValidationReport{
		DocumentType: "manifest",
		SchemaRef:    manifestSchemaAsset,
		Version:      detectVersion(data),
	}
	schemaDoc, err := getManifestSchema()
	if err != nil {
		report.Issues = append(report.Issues, ValidationIssue{Path: "$", Code: "schema_load", Message: err.Error()})
		return report, ValidationError{Report: report}
	}
	raw, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		report.Issues = append(report.Issues, ValidationIssue{Path: "$", Code: "json_decode", Message: err.Error()})
		return report, ValidationError{Report: report}
	}
	if err := schemaDoc.Validate(raw); err != nil {
		report.Issues = flattenValidationError(err)
		return report, ValidationError{Report: report}
	}
	report.Valid = true
	return report, nil
}

func ValidateStateBytes(data []byte) (ValidationReport, error) {
	report := ValidationReport{
		DocumentType: "state",
		SchemaRef:    stateSchemaAsset,
		Version:      detectVersion(data),
	}
	schemaDoc, err := getStateSchema()
	if err != nil {
		report.Issues = append(report.Issues, ValidationIssue{Path: "$", Code: "schema_load", Message: err.Error()})
		return report, ValidationError{Report: report}
	}
	raw, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		report.Issues = append(report.Issues, ValidationIssue{Path: "$", Code: "json_decode", Message: err.Error()})
		return report, ValidationError{Report: report}
	}
	if err := schemaDoc.Validate(raw); err != nil {
		report.Issues = flattenValidationError(err)
		return report, ValidationError{Report: report}
	}
	report.Valid = true
	return report, nil
}

func getManifestSchema() (*jsonschema.Schema, error) {
	if err := compileSchemas(); err != nil {
		return nil, err
	}
	return manifestSchema, nil
}

func getStateSchema() (*jsonschema.Schema, error) {
	if err := compileSchemas(); err != nil {
		return nil, err
	}
	return stateSchema, nil
}

func compileSchemas() error {
	compileOnce.Do(func() {
		manifestRaw, err := devlaunchassets.ReadEmbeddedFile(manifestSchemaAsset)
		if err != nil {
			compileErr = fmt.Errorf("read %s: %w", manifestSchemaAsset, err)
			return
		}
		stateRaw, err := devlaunchassets.ReadEmbeddedFile(stateSchemaAsset)
		if err != nil {
			compileErr = fmt.Errorf("read %s: %w", stateSchemaAsset, err)
			return
		}

		var manifestDoc any
		if err := json.Unmarshal(manifestRaw, &manifestDoc); err != nil {
			compileErr = fmt.Errorf("parse %s: %w", manifestSchemaAsset, err)
			return
		}
		var stateDoc any
		if err := json.Unmarshal(stateRaw, &stateDoc); err != nil {
			compileErr = fmt.Errorf("parse %s: %w", stateSchemaAsset, err)
			return
		}

		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource(manifestSchemaURL, manifestDoc); err != nil {
			compileErr = fmt.Errorf("add manifest schema resource: %w", err)
			return
		}
		if err := compiler.AddResource(stateSchemaURL, stateDoc); err != nil {
			compileErr = fmt.Errorf("add state schema resource: %w", err)
			return
		}

		manifestSchema, err = compiler.Compile(manifestSchemaURL)
		if err != nil {
			compileErr = fmt.Errorf("compile manifest schema: %w", err)
			return
		}
		stateSchema, err = compiler.Compile(stateSchemaURL)
		if err != nil {
			compileErr = fmt.Errorf("compile state schema: %w", err)
			return
		}
	})
	return compileErr
}

func flattenValidationError(err error) []ValidationIssue {
	var validationErr *jsonschema.ValidationError
	if !asValidationError(err, &validationErr) {
		return []ValidationIssue{{Path: "$", Code: "schema_validate", Message: err.Error()}}
	}

	issues := make([]ValidationIssue, 0, len(validationErr.Causes)+1)
	collectIssues(validationErr, &issues)
	if len(issues) == 0 {
		issues = append(issues, ValidationIssue{Path: "$", Code: "schema_validate", Message: validationErr.Error()})
	}
	return issues
}

func collectIssues(err *jsonschema.ValidationError, issues *[]ValidationIssue) {
	path := toJSONPath(err.InstanceLocation)
	*issues = append(*issues, ValidationIssue{
		Path:    path,
		Code:    "schema_validate",
		Message: err.Error(),
	})
	for _, cause := range err.Causes {
		collectIssues(cause, issues)
	}
}

func toJSONPath(parts []string) string {
	if len(parts) == 0 {
		return "$"
	}
	builder := strings.Builder{}
	builder.WriteString("$")
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err == nil {
			builder.WriteString("[")
			builder.WriteString(part)
			builder.WriteString("]")
			continue
		}
		builder.WriteString(".")
		builder.WriteString(part)
	}
	return builder.String()
}

func detectVersion(data []byte) int {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0
	}
	value, ok := doc["version"]
	if !ok {
		return 0
	}
	switch parsed := value.(type) {
	case float64:
		return int(parsed)
	case int:
		return parsed
	default:
		return 0
	}
}

func asValidationError(err error, target **jsonschema.ValidationError) bool {
	return errors.As(err, target)
}
