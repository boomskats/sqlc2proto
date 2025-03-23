package includes

import (
	"github.com/boomskats/sqlc2proto/internal/parser"
)

// ResolveDependencies resolves dependencies for the included queries
func ResolveDependencies(includes IncludesFile, queryMethods []parser.QueryMethod, messages []parser.ProtoMessage) IncludesFile {
	// Create a map of all messages for quick lookup
	messageMap := make(map[string]parser.ProtoMessage)
	for _, msg := range messages {
		messageMap[msg.Name] = msg
	}

	// Create a set of included models
	includedModels := make(map[string]bool)
	for _, model := range includes.Models {
		includedModels[model] = true
	}

	// For each included query, add its dependencies
	for _, queryName := range includes.Queries {
		// Find the query method
		for _, method := range queryMethods {
			if method.Name == queryName {
				// Add parameter types as dependencies
				for _, param := range method.ParamTypes {
					addModelAndDependencies(param.Type, includedModels, messageMap)
				}

				// Add return type as dependency
				if method.ReturnType != "" {
					addModelAndDependencies(method.ReturnType, includedModels, messageMap)
				}

				break
			}
		}
	}

	// Convert the set back to a slice
	var resolvedModels []string
	for model := range includedModels {
		resolvedModels = append(resolvedModels, model)
	}

	// Create a new includes file with the resolved dependencies
	return IncludesFile{
		Models:  resolvedModels,
		Queries: includes.Queries,
	}
}

// addModelAndDependencies recursively adds a model and its dependencies to the included models set
func addModelAndDependencies(modelName string, includedModels map[string]bool, messageMap map[string]parser.ProtoMessage) {
	// Check if this is a known model
	model, exists := messageMap[modelName]
	if !exists {
		return // Not a model, might be a primitive type
	}

	// Add this model if not already included
	if !includedModels[modelName] {
		includedModels[modelName] = true

		// Recursively add dependencies from fields
		for _, field := range model.Fields {
			// Skip primitive types
			if !isPrimitiveType(field.Type) {
				addModelAndDependencies(field.Type, includedModels, messageMap)
			}
		}
	}
}

// isPrimitiveType checks if a type is a primitive Protocol Buffer type
func isPrimitiveType(typeName string) bool {
	primitiveTypes := map[string]bool{
		"string": true, "int32": true, "int64": true, "float": true,
		"double": true, "bool": true, "bytes": true,
		"google.protobuf.Timestamp": true, // Consider Timestamp as primitive
	}
	return primitiveTypes[typeName]
}

// GetDependencyAdditions returns a list of models that were added due to dependencies
func GetDependencyAdditions(original IncludesFile, resolved IncludesFile) []string {
	// Create a set of original models
	originalModels := make(map[string]bool)
	for _, model := range original.Models {
		originalModels[model] = true
	}

	// Find models in resolved that are not in original
	var additions []string
	for _, model := range resolved.Models {
		if !originalModels[model] {
			additions = append(additions, model)
		}
	}

	return additions
}
