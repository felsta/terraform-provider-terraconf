package terraconf

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Represents an error found when validating a YAML document against a JSON
// schema. Message is the message that we want to display to the user and
// Pointer points to the location of the error.
type SchemaError struct {
	Message string
	Pointer JsonPointer
}

// CompileSchema reads and compiles the JSON schema at the given path. The
// compiled schema can be reused to validate multiple values.
func CompileSchema(schemaPath string) (*jsonschema.Schema, error) {
	path, err := filepath.Abs(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path of schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("compiling schema: %w", err)
	}

	return schema, nil
}

// ValidateWithSchema validates the given value against the compiled schema
// and returns a list of SchemaErrors representing the errors found during
// validation. If the error returned by the jsonschema library is not a
// ValidationError, an error is returned.
func ValidateWithSchema(schema *jsonschema.Schema, value any) ([]SchemaError, error) {
	err := schema.Validate(value)
	if err == nil {
		return nil, nil
	}

	validationError, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return nil, fmt.Errorf("found an error that is not a validation error: %w", err)
	}

	schemaErrors := []SchemaError{}
	for _, leaf := range sortOutputUnits(validationError.DetailedOutput().Errors, compareByInstance) {
		schemaError, err := findError(leaf)
		if err != nil {
			return nil, fmt.Errorf("finding error message in schema validation error output: %w", err)
		}

		schemaErrors = append(schemaErrors, schemaError)
	}

	return schemaErrors, nil
}

// Finds the error message that we want to display for a specific OutputUnit
// from the jsonschema library.
func findError(unit jsonschema.OutputUnit) (SchemaError, error) {
	if unit.Error != nil {
		return SchemaError{
			Message: unit.Error.String(),
			Pointer: NewJsonPointer(unit.InstanceLocation),
		}, nil
	}

	if len(unit.Errors) > 0 {
		// FIXME: We should probably notify the user that there are other
		//  things they could do to conform to the schema instead of just
		//  showing the first error of a oneOf/anyOf combination.
		//  See Testcase: "test wrong array oneOf none"
		errorMessage, err := findError(sortOutputUnits(unit.Errors, compareByKeyword)[0])
		if err != nil {
			return SchemaError{}, err
		}

		return errorMessage, nil
	}

	return SchemaError{}, fmt.Errorf("no error found")
}

// sortOutputUnits returns a copy of the given output units sorted with the
// given comparison function. The jsonschema library produces some output
// units in a non-deterministic order, since it iterates over maps
// internally. Sorting them makes the reported errors stable between runs.
func sortOutputUnits(units []jsonschema.OutputUnit, compare func(a, b jsonschema.OutputUnit) int) []jsonschema.OutputUnit {
	sorted := slices.Clone(units)
	slices.SortStableFunc(sorted, compare)
	return sorted
}

// compareByInstance orders output units by their location in the validated
// value, so errors are reported in the order they appear in the file.
func compareByInstance(a, b jsonschema.OutputUnit) int {
	return cmp.Or(
		compareJsonPointers(a.InstanceLocation, b.InstanceLocation),
		compareJsonPointers(a.KeywordLocation, b.KeywordLocation),
	)
}

// compareByKeyword orders output units by their location in the schema,
// so the alternatives of e.g. oneOf are considered in the order they are
// defined in.
func compareByKeyword(a, b jsonschema.OutputUnit) int {
	return cmp.Or(
		compareJsonPointers(a.KeywordLocation, b.KeywordLocation),
		compareJsonPointers(a.InstanceLocation, b.InstanceLocation),
	)
}

// compareJsonPointers compares two JSON pointers token by token. Tokens that
// are both integers (array indices) are compared numerically, so that "/10"
// is ordered after "/2".
func compareJsonPointers(a, b string) int {
	tokensA := strings.Split(a, "/")
	tokensB := strings.Split(b, "/")

	for i := 0; i < len(tokensA) && i < len(tokensB); i++ {
		indexA, errA := strconv.Atoi(tokensA[i])
		indexB, errB := strconv.Atoi(tokensB[i])

		var result int
		if errA == nil && errB == nil {
			result = cmp.Compare(indexA, indexB)
		} else {
			result = strings.Compare(tokensA[i], tokensB[i])
		}

		if result != 0 {
			return result
		}
	}

	return cmp.Compare(len(tokensA), len(tokensB))
}
