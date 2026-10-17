package cmd

import (
	"fmt"
	"strings"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	"github.com/GrayCodeAI/reflect/internal/errhint"
	"github.com/GrayCodeAI/reflect/internal/reflecterr"
)

// friendlyErrorMessage returns the user-friendly message for an error.
// Delegates to the shared reflecterr.ClassifyErrorMessage for the base message,
// then enriches specific cases with dynamic hints that require the config
// package (reflecterr can't import internal/config without a cycle).
func friendlyErrorMessage(err error) string {
	msg := reflecterr.ClassifyErrorMessage(err)

	if err == nil {
		return msg
	}

	ec := reflecterr.ClassifyError(err)
	low := strings.ToLower(err.Error())

	switch ec.ExitCode {
	case reflecterr.ExitNotFound:
		// Enrich model-not-found errors with concrete examples from the catalog.
		if strings.Contains(low, "model") || strings.Contains(low, "unknown") ||
			strings.Contains(low, "does not exist") {
			ex1, ex2 := reflectconfig.ExampleModelHints()
			msg = fmt.Sprintf(
				"Model not found. Check your model name with /model.\n  Examples from the relay catalog: %s, %s\n  Use /models to list all models, or /config to change provider.",
				ex1, ex2,
			)
		}
	case reflecterr.ExitAuth:
		msg += "\n  Check your API key with /config. Keys can expire or be revoked."
	case reflecterr.ExitNetwork:
		msg += "\n  Check your internet connection. If you're behind a proxy, configure it with /config."
	case reflecterr.ExitTimeout:
		msg += "\n  The request took too long. Try again, or use /model to switch to a faster provider."
	}

	// Provider-specific one-line hint. errhint.Classify is deliberately
	// conservative (gates on a provider-origin marker), so local errors draw no
	// hint here.
	if h := errhint.TUIHint(err); h != "" {
		msg += "\n  " + h
	}

	return msg
}
