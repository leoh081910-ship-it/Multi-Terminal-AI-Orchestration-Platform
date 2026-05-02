package decompose

import "fmt"

// ValidateDecomposition checks a decomposition result for structural correctness.
func ValidateDecomposition(result *DecompositionResult) error {
	if result == nil {
		return fmt.Errorf("decomposition result is nil")
	}
	if len(result.Subtasks) == 0 {
		return fmt.Errorf("decomposition produced no subtasks")
	}

	ids := make(map[string]bool)
	for i, spec := range result.Subtasks {
		if spec.Title == "" {
			return fmt.Errorf("subtask %d: title is required", i)
		}
		if spec.Type == "" {
			spec.Type = "task" // default type
		}
		ids[spec.Title] = true
	}

	// Validate dependency references exist
	for _, spec := range result.Subtasks {
		for _, dep := range spec.DependsOn {
			if !ids[dep] {
				return fmt.Errorf("subtask %q depends on unknown subtask %q", spec.Title, dep)
			}
		}
	}

	return nil
}
