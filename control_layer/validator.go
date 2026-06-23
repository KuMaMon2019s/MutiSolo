package controllayer

import "strings"

func Validate(prompt string, generation Generation, approveSystem bool) Validation {
	class := Classify(prompt + "\n" + generation.Content)
	reasons := make([]string, 0)

	if strings.TrimSpace(generation.Content) == "" {
		return Validation{
			Class:   class,
			Status:  ValidationBlocked,
			Reasons: []string{"generated content is empty"},
		}
	}

	if class == ClassSystemDesign && !approveSystem {
		reasons = append(reasons, "system design code is blocked by default")
		reasons = append(reasons, "generated output must not modify prompts, architecture, runtime, or coordination internals")
		return Validation{
			Class:   class,
			Status:  ValidationBlocked,
			Reasons: reasons,
		}
	}

	reasons = append(reasons, "output is stored as an artifact and not applied to runtime")
	return Validation{
		Class:   class,
		Status:  ValidationAllowed,
		Reasons: reasons,
	}
}
