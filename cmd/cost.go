package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/engine/cost"
	analytics "github.com/GrayCodeAI/reflect/internal/observability"
	"github.com/GrayCodeAI/reflect/internal/storage"
	"github.com/GrayCodeAI/reflect/internal/usage"
	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "[Experimental] Analyze and optimize LLM API spend",
	Long: `cost provides analysis and optimization recommendations for LLM API
usage. It reads the per-generation cost ledger the engine appends to on
every model completion (cost.jsonl in the state directory) and reports
spend breakdowns by model and task type, waste detection, and model
routing suggestions.

Subcommands:
  analyze   Run a full cost optimization analysis
  summary   Show a quick spend summary

Both subcommands accept:
  --session <id>   restrict the report to one session
  --since <p>      window: 24h, 7d, 30d, or all (default)`,
}

var (
	costAnalyzeJSON bool
	costSummaryJSON bool
	costSession     string // --session: filter to a single session id
	costSince       string // --since: 24h, 7d, 30d, or all
)

func init() {
	costAnalyzeCmd.Flags().BoolVar(&costAnalyzeJSON, "json", false, "output the report as JSON")
	costSummaryCmd.Flags().BoolVar(&costSummaryJSON, "json", false, "output the summary as JSON")
	costAnalyzeCmd.Flags().StringVar(&costSession, "session", "", "restrict the report to one session id")
	costAnalyzeCmd.Flags().StringVar(&costSince, "since", "all", "spend window: 24h, 7d, 30d, or all")
	costSummaryCmd.Flags().StringVar(&costSession, "session", "", "restrict the report to one session id")
	costSummaryCmd.Flags().StringVar(&costSince, "since", "all", "spend window: 24h, 7d, 30d, or all")
	costCmd.AddCommand(costAnalyzeCmd)
	costCmd.AddCommand(costSummaryCmd)
}

// parseCostSince converts --since into a unix-ms lower bound (0 = no bound).
// It accepts the usage windows (24h, 7d, 30d) plus "all"/"" for the entire
// ledger, which is the default for cost analysis.
func parseCostSince(s string) (int64, error) {
	switch strings.TrimSpace(s) {
	case "", "all":
		return 0, nil
	case "24h", "7d", "30d":
		sinceMS, _, err := usage.ParsePeriod(s)
		return sinceMS, err
	default:
		return 0, fmt.Errorf("cost: invalid --since %q (want 24h, 7d, 30d, or all)", s)
	}
}

// loadCostEntries reads the generation cost ledger and applies the --session
// and --since filters. A missing ledger is empty history, not an error.
func loadCostEntries() ([]analytics.CostEntry, error) {
	entries, err := cost.LoadCostHistory()
	if err != nil {
		return nil, fmt.Errorf("cost: read ledger: %w", err)
	}
	sinceMS, err := parseCostSince(costSince)
	if err != nil {
		return nil, err
	}
	out := make([]analytics.CostEntry, 0, len(entries))
	for _, e := range entries {
		if costSession != "" && e.SessionID != costSession {
			continue
		}
		if sinceMS > 0 && e.Timestamp.UnixMilli() < sinceMS {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// costScopeLabel describes the active --session/--since filters.
func costScopeLabel() string {
	window := strings.TrimSpace(costSince)
	if window == "" {
		window = "all"
	}
	if costSession != "" {
		return fmt.Sprintf("session %s, %s", costSession, window)
	}
	return window
}

// printCostEmptyState explains an empty ledger instead of failing.
func printCostEmptyState(cmd *cobra.Command) {
	cmd.Println(auditTint("No cost data recorded yet.", textMuted))
	cmd.Println(auditTint("The engine appends one entry per generation to "+filepath.Join(storage.StateDir(), "cost.jsonl")+".", textMuted))
}

// sortedModelSpend orders a by-model report by total cost, most expensive
// first (mirroring FormatOptimizationReport's ordering).
func sortedModelSpend(byModel map[string]*analytics.ModelSpend) []*analytics.ModelSpend {
	models := make([]*analytics.ModelSpend, 0, len(byModel))
	for _, ms := range byModel {
		models = append(models, ms)
	}
	sort.Slice(models, func(i, j int) bool { return models[i].TotalCost > models[j].TotalCost })
	return models
}

var costAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Run a full cost optimization analysis",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := loadCostEntries()
		if err != nil {
			return err
		}
		report := analytics.Analyze(entries)

		if costAnalyzeJSON {
			out, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling report: %w", err)
			}
			cmd.Println(string(out))
			return nil
		}

		if report.TotalSpend == 0 {
			printCostEmptyState(cmd)
			return nil
		}

		cmd.Println(auditTint("Cost analysis ("+costScopeLabel()+")", reflectColor))
		cmd.Println()
		cmd.Print(analytics.FormatOptimizationReport(report))
		return nil
	},
}

var costSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show a quick spend summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := loadCostEntries()
		if err != nil {
			return err
		}
		report := analytics.Analyze(entries)

		if costSummaryJSON {
			out, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling report: %w", err)
			}
			cmd.Println(string(out))
			return nil
		}

		if report.TotalSpend == 0 {
			printCostEmptyState(cmd)
			return nil
		}

		cmd.Println(auditTint("Cost summary ("+costScopeLabel()+")", reflectColor))
		cmd.Println()
		cmd.Println(auditTint("Total spend:      ", textMuted) + auditTint(fmt.Sprintf("$%.4f", report.TotalSpend), textPrimary))
		cmd.Println(auditTint("Productive spend: ", textMuted) + auditTint(fmt.Sprintf("$%.4f", report.ProductiveSpend), textPrimary))
		cmd.Println(auditTint("Wasted spend:     ", textMuted) + auditTint(fmt.Sprintf("$%.4f", report.WastedSpend), errorCoral))
		cmd.Println(auditTint("Yield rate:       ", textMuted) + auditTint(fmt.Sprintf("%.1f%%", report.YieldRate*100), textPrimary))

		if len(report.ByModel) > 0 {
			cmd.Println()
			cmd.Println(auditTint("Spend by model:", textMuted))
			for _, ms := range sortedModelSpend(report.ByModel) {
				cmd.Println(auditTint(fmt.Sprintf("  %-28s %5d calls  $%.4f", truncateModel(ms.Model), ms.Calls, ms.TotalCost), textPrimary))
			}
		}

		if len(report.Recommendations) > 0 {
			cmd.Println()
			cmd.Println(auditTint("Top recommendation:", textPrimary))
			rec := report.Recommendations[0]
			cmd.Println(auditTint(fmt.Sprintf("  [%s] %s (est. savings: $%.4f)", rec.Type, rec.Description, rec.Savings), textPrimary))
		}
		return nil
	},
}
