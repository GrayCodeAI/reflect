package onboarding

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	reflectconfig "github.com/GrayCodeAI/reflect/internal/config"
	internaltheme "github.com/GrayCodeAI/reflect/internal/theme"
	"golang.org/x/term"

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
		{name: "anthropic", desc: "Claude models"},
		{name: "openai", desc: "OpenAI models"},
		{name: "gemini", desc: "Gemini models"},
		{name: "openrouter", desc: "Models through OpenRouter"},
		{name: "groq", desc: "Fast inference"},
		{name: "poolside", desc: "Poolside models"},
		{"ollama", "", "Local models (no API key needed)"},
	}
	for i := range providers {
		if !reflectconfig.IsSetupGateway(providers[i].name) {
			return fmt.Errorf("%s is not available in the Relay gateway catalog", providers[i].name)
		}
		providers[i].envKey = reflectconfig.SetupGatewayCredentialEnv(providers[i].name)
		if providers[i].envKey == "" && providers[i].name != "ollama" {
			return fmt.Errorf("%s credential mapping is unavailable; use /config", providers[i].name)
		}
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

		apiKey, err := readSetupSecret(reader)
		if err != nil {
			return fmt.Errorf("read API key: %w", err)
		}

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
			return fmt.Errorf("save selected provider: %w", err)
		}

		fmt.Println()
		fmt.Printf("  %s"+icons.CheckBold()+" API key saved to %s%s\n", teal, reflectconfig.CredentialStoreName(), reset)
	} else if selected.name == "ollama" {
		if err := reflectconfig.SetActiveProvider(context.Background(), "ollama"); err != nil {
			return fmt.Errorf("save selected provider: %w", err)
		}
		fmt.Printf("  %s"+icons.CheckBold()+" Ollama selected (make sure ollama is running)%s\n", teal, reset)
	} else {
		if err := reflectconfig.SetActiveProvider(context.Background(), selected.name); err != nil {
			return fmt.Errorf("save selected provider: %w", err)
		}
		fmt.Printf("  %s"+icons.CheckBold()+" Using %s (credential already in %s)%s\n", teal, selected.name, reflectconfig.CredentialStoreName(), reset)
	}

	fmt.Println("  Loading models...")
	reflectconfig.DiscoverCatalogAfterSetup(context.Background(), os.Stdout)
	models, err := reflectconfig.ListEngineModels(context.Background(), selected.name, false)
	if err != nil || len(models) == 0 {
		return fmt.Errorf("no cached models for %s; run reflect models refresh, then use /config to select a model", selected.name)
	}
	currentModel := reflectconfig.ActiveModel(context.Background())
	defaultModel := reflectconfig.DefaultModelForProvider(selected.name)
	defaultIdx := preferredSetupModel(models, currentModel, defaultModel)
	fmt.Println("  Choose a model:")
	for i, model := range models {
		badge := "tool support unverified"
		if slices.Contains(model.Capabilities, "tools") {
			badge = "coding tools listed"
		}
		fmt.Printf("  %d) %s  ·  %s\n", i+1, model.ID, badge)
	}
	for {
		fmt.Printf("  Enter number (default %d): ", defaultIdx+1)
		choice, readErr := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		modelIdx := defaultIdx
		if choice != "" {
			n, err := strconv.Atoi(choice)
			if err != nil || n < 1 || n > len(models) {
				if readErr != nil {
					return fmt.Errorf("model selection ended before a valid choice")
				}
				fmt.Printf("  Select a model number from 1 to %d.\n", len(models))
				continue
			}
			modelIdx = n - 1
		}
		if err := reflectconfig.SetActiveModel(context.Background(), models[modelIdx].ID); err != nil {
			return fmt.Errorf("save selected model: %w", err)
		}
		fmt.Printf("  %s"+icons.CheckBold()+" Model selected: %s%s\n", teal, models[modelIdx].ID, reset)
		report := reflectconfig.EnginePreflightReport(context.Background())
		var toolCheckFound bool
		retry := false
		for _, check := range report.Checks {
			if check.Name != "model_tools" {
				continue
			}
			toolCheckFound = true
			if string(check.Status) == "fail" {
				fmt.Printf("  %s"+icons.Alert()+" %s%s\n", red, check.Detail, reset)
				hasVerifiedAlternative := false
				for i, model := range models {
					if i != modelIdx && slices.Contains(model.Capabilities, "tools") {
						hasVerifiedAlternative = true
						break
					}
				}
				if !hasVerifiedAlternative {
					fmt.Println("  Run reflect models refresh or use /config before starting a coding task.")
					return nil
				}
				fmt.Print("  Choose another model? [Y/n]: ")
				answer, answerErr := reader.ReadString('\n')
				if answerErr != nil || strings.EqualFold(strings.TrimSpace(answer), "n") {
					fmt.Println("  Run reflect models refresh or use /config before starting a coding task.")
					return nil
				}
				defaultIdx = preferredSetupModel(models, "", defaultModel)
				retry = true
				break
			}
		}
		if retry {
			continue
		}
		if !toolCheckFound {
			return fmt.Errorf("could not verify model tool support; run reflect preflight")
		}
		fmt.Println("  Cached model metadata confirms coding tools.")
		break
	}
	fmt.Println("  Next: run reflect preflight --live, then reflect -p \"Say hello\"")

	return nil
}

func readSetupSecret(reader *bufio.Reader) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		secret, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		return strings.TrimSpace(string(secret)), err
	}
	secret, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(secret) == "" {
		return "", err
	}
	return strings.TrimSpace(secret), nil
}

func preferredSetupModel(models []reflectconfig.EngineModel, current, recommended string) int {
	for _, id := range []string{current, recommended} {
		for i, model := range models {
			if setupModelMatches(model, id) && slices.Contains(model.Capabilities, "tools") {
				return i
			}
		}
	}
	for i, model := range models {
		if slices.Contains(model.Capabilities, "tools") {
			return i
		}
	}
	for _, id := range []string{current, recommended} {
		for i, model := range models {
			if setupModelMatches(model, id) {
				return i
			}
		}
	}
	return 0
}

func setupModelMatches(model reflectconfig.EngineModel, id string) bool {
	return id != "" && (model.ID == id || model.CanonicalID == id)
}

// validateAPIKey leaves key format validation to the selected provider.
// Provider tokens may use custom prefixes or opaque formats.
func validateAPIKey(provider, key string) (string, bool) {
	if strings.TrimSpace(key) == "" {
		return "API key is empty", false
	}
	return "", true
}
