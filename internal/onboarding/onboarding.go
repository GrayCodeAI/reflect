package onboarding

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	internaltheme "github.com/GrayCodeAI/reflect/internal/theme"

	"github.com/GrayCodeAI/reflect/internal/ui/icons"
)

var (
	teal  string
	dim   string
	bold  string
	red   string
	reset string
	brand string
)

func init() {
	initColorCodes()
}

// initColorCodes sets the ANSI escape codes, honoring NO_COLOR/FORCE_COLOR/TTY
// so the onboarding wizard stays plain in scripted or NO_COLOR environments.
func initColorCodes() {
	teal = ""
	dim = ""
	bold = ""
	red = ""
	reset = ""
	brand = ""
	if !internaltheme.ColorEnabled() {
		return
	}
	teal = "\033[38;2;78;205;196m"
	dim = "\033[2m"
	bold = "\033[1m"
	red = "\033[38;2;224;85;85m"
	reset = "\033[0m"
	brand = internaltheme.BrandANSI
}

// Welcome prints a short setup introduction.
func Welcome(version string) {
	fmt.Println()
	fmt.Printf("  %s%sReflect%s  %sv%s%s\n", brand, bold, reset, dim, version, reset)
	fmt.Println("  Set up a provider and model to start coding in this folder.")
	fmt.Println()
	fmt.Println("  Keys stay in your OS secret store.")
	fmt.Println("  You can change the model later with /config.")
	fmt.Println()
}

// NeedsSetup returns true only when reflect setup is explicitly requested.
// Normal reflect startup uses /config inside the TUI instead of blocking setup.
func NeedsSetup() bool {
	return false
}

// RunSetup runs the interactive first-run setup.
func RunSetup() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(teal + bold + "  Developer setup" + reset)
	fmt.Println()
	fmt.Println(dim + "  Keys are stored in " + reflectconfig.CredentialStoreName() + ", not .env or shell env." + reset)
	fmt.Println()

	// Provider selection
	fmt.Println("  Choose your LLM provider:")
	fmt.Println()
	providers := []struct {
		name   string
		envKey string
		desc   string
	}{
		{"anthropic", "ANTHROPIC_API_KEY", "Claude (recommended)"},
		{"openai", "OPENAI_API_KEY", "GPT-4o, o1, o3"},
		{"gemini", "GEMINI_API_KEY", "Gemini 2.5"},
		{"openrouter", "OPENROUTER_API_KEY", "200+ models"},
		{"groq", "GROQ_API_KEY", "Fast inference"},
		{"poolside", "POOLSIDE_API_KEY", "Poolside models"},
		{"ollama", "", "Local models (no API key needed)"},
	}

	for i, p := range providers {
		fmt.Printf("  %s%d%s) %s%-12s%s %s\n", teal, i+1, reset, bold, p.name, reset, dim+p.desc+reset)
	}
	fmt.Println()
	fmt.Print("  Enter number (1-7): ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(providers) {
		return fmt.Errorf("select a provider number from 1 to %d", len(providers))
	}
	selected := providers[idx-1]
	fmt.Println()
	fmt.Printf("  Selected: %s%s%s\n", teal, selected.name, reset)

	// API key input
	if selected.envKey != "" && !reflectconfig.HasStoredCredentialForProvider(context.Background(), selected.name) {
		fmt.Println()
		fmt.Printf("  Enter your %s API key:\n", selected.name)
		fmt.Printf("  %s(Get one at the provider's website)%s\n", dim, reset)
		fmt.Print("  > ")

		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		if apiKey == "" {
			fmt.Println(red + "  No API key entered. Run reflect and use /config to save a key securely." + reset)
			return fmt.Errorf("no API key")
		}

		// Check for an empty key before saving. Relay validates provider access.
		if warning, valid := validateAPIKey(selected.name, apiKey); !valid {
			fmt.Printf("  %s"+icons.Alert()+" %s%s\n", red, warning, reset)
			fmt.Println(red + "  API key not saved. Please check your key and try again." + reset)
			return fmt.Errorf("invalid API key")
		} else if warning != "" {
			fmt.Printf("  %s"+icons.Alert()+" %s (saving anyway)%s\n", dim, warning, reset)
		}

		ctx := context.Background()
		if err := reflectconfig.PersistAPIKey(ctx, selected.envKey, apiKey); err != nil {
			fmt.Printf("  %sWarning: couldn't save API key: %s%s\n", dim, err, reset)
			return err
		}

		if err := reflectconfig.SetActiveProvider(context.Background(), selected.name); err != nil {
			fmt.Printf("  %sWarning: couldn't save provider: %s%s\n", dim, err, reset)
		}

		fmt.Println()
		fmt.Printf("  %s"+icons.CheckBold()+" API key saved to %s%s\n", teal, reflectconfig.CredentialStoreName(), reset)
	} else if selected.name == "ollama" {
		_ = reflectconfig.SetActiveProvider(context.Background(), "ollama")
		fmt.Printf("  %s"+icons.CheckBold()+" Ollama selected (make sure ollama is running)%s\n", teal, reset)
	} else {
		_ = reflectconfig.SetActiveProvider(context.Background(), selected.name)
		fmt.Printf("  %s"+icons.CheckBold()+" Using %s (credential already in %s)%s\n", teal, selected.name, reflectconfig.CredentialStoreName(), reset)
	}

	// Security notes
	fmt.Println()
	fmt.Println(dim + "  ─────────────────────────────────────────" + reset)
	fmt.Println()
	fmt.Println("  " + bold + "Security notes:" + reset)
	fmt.Println("  1. reflect can make mistakes — always review changes")
	fmt.Println("  2. reflect will ask before running commands or writing files")
	fmt.Println("  3. Use /autonomy allow <tool> to auto-approve tools")
	fmt.Println()
	fmt.Println(dim + "  ─────────────────────────────────────────" + reset)
	fmt.Println()
	fmt.Print("  Press Enter to load models... ")
	_, _ = reader.ReadString('\n')

	reflectconfig.DiscoverCatalogAfterSetup(context.Background(), os.Stdout)
	models, err := reflectconfig.ModelIDsForProvider(selected.name)
	if err != nil || len(models) == 0 {
		return fmt.Errorf("no cached models for %s; run reflect models refresh, then use /config to select a model", selected.name)
	}
	currentModel := reflectconfig.ActiveModel(context.Background())
	defaultModel := reflectconfig.DefaultModelForProvider(selected.name)
	defaultIdx := 0
	for i, model := range models {
		if model == currentModel {
			defaultIdx = i
			break
		}
		if model == defaultModel {
			defaultIdx = i
		}
	}
	fmt.Println("  Choose a model:")
	for i, model := range models {
		fmt.Printf("  %d) %s\n", i+1, model)
	}
	fmt.Printf("  Enter number (default %d): ", defaultIdx+1)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	modelIdx := defaultIdx
	if choice != "" {
		n, err := strconv.Atoi(choice)
		if err != nil || n < 1 || n > len(models) {
			return fmt.Errorf("select a model number from 1 to %d", len(models))
		}
		modelIdx = n - 1
	}
	if err := reflectconfig.SetActiveModel(context.Background(), models[modelIdx]); err != nil {
		return fmt.Errorf("save selected model: %w", err)
	}
	fmt.Printf("  %s"+icons.CheckBold()+" Model selected: %s%s\n", teal, models[modelIdx], reset)
	fmt.Println("  Next: run reflect preflight --live, then reflect -p \"Say hello\"")

	return nil
}

// validateAPIKey leaves key format validation to the selected provider.
// Provider tokens may use custom prefixes or opaque formats.
func validateAPIKey(provider, key string) (string, bool) {
	if strings.TrimSpace(key) == "" {
		return "API key is empty", false
	}
	return "", true
}
