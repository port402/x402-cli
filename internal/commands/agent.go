package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/port402/x402-cli/internal/a2a"
	"github.com/port402/x402-cli/internal/output"
)

var (
	agentCardURL string
	agentTimeout int
)

var agentCmd = &cobra.Command{
	Use:   "agent <url>",
	Short: "Discover A2A agent card from an endpoint",
	Long: `Discover and display A2A (Agent-to-Agent) protocol agent cards.

Tries these well-known paths in order:
  1. /.well-known/agent.json (A2A v0.1)
  2. /.well-known/agent-card.json (A2A v0.2+)
  3. /.well-known/agents.json (wildcard spec)

Examples:
  x402 agent https://api.example.com
  x402 agent https://api.example.com --json
  x402 agent https://api.example.com --card-url /custom/agent.json`,
	Args: cobra.ExactArgs(1),
	RunE: runAgent,
}

func init() {
	agentCmd.Flags().StringVar(&agentCardURL, "card-url", "", "Custom agent card path (overrides discovery)")
	agentCmd.Flags().IntVar(&agentTimeout, "timeout", 5, "Request timeout in seconds")
	rootCmd.AddCommand(agentCmd)
}

func runAgent(cmd *cobra.Command, args []string) error {
	url := args[0]
	timeout := time.Duration(agentTimeout) * time.Second

	result := a2a.Discover(cmd.Context(), url, agentCardURL, timeout)

	if GetJSONOutput() {
		return output.PrintJSON(result)
	}

	printAgentResult(result)

	if result.ExitCode != 0 {
		cmd.SilenceUsage = true
		if result.Error != "" {
			return fmt.Errorf("agent card discovery failed: %s", result.Error)
		}
		return fmt.Errorf("agent card discovery failed")
	}

	return nil
}

// printAgentResult outputs the discovery result in human-readable format.
func printAgentResult(result *a2a.Result) {
	if !result.Found {
		printNotFound(result)
		return
	}

	card := result.Card
	fmt.Printf("✓ Agent card found (%s)\n\n", result.DiscoveryPath)

	fmt.Println("Agent:")
	fmt.Printf("  Name:        %s\n", card.Name)
	printIfSet("  Version:     %s\n", card.Version)
	printIfSet("  Description: %s\n", card.Description)

	printSkills(card.Skills)
	printCapabilities(card.Capabilities)
	printProvider(card.Provider)
	printIfSet("\nDocs: %s\n", card.DocumentationURL)
}

func printNotFound(result *a2a.Result) {
	fmt.Println("⚠ No agent card found\n\nTried:")
	for _, attempt := range result.TriedPaths {
		switch {
		case attempt.Status > 0:
			fmt.Printf("  • %s (%d)\n", attempt.Path, attempt.Status)
		case attempt.Error != "":
			fmt.Printf("  • %s (error: %s)\n", attempt.Path, attempt.Error)
		default:
			fmt.Printf("  • %s (network error)\n", attempt.Path)
		}
	}

	if result.Error == "agent card requires authentication" {
		fmt.Println("\nNote: Agent card requires authentication")
	}

	fmt.Println("\nHint: Use --card-url to specify a custom location")
}

func printIfSet(format, value string) {
	if value != "" {
		fmt.Printf(format, value)
	}
}

func printSkills(skills []a2a.Skill) {
	if len(skills) == 0 {
		return
	}
	fmt.Println("\nSkills:")
	for _, skill := range skills {
		name := skill.Name
		if name == "" {
			name = skill.ID
		}
		if skill.Description != "" {
			fmt.Printf("  • %s — %s\n", name, skill.Description)
		} else {
			fmt.Printf("  • %s\n", name)
		}
	}
}

func printCapabilities(caps *a2a.Capabilities) {
	if caps == nil {
		return
	}

	var enabled []string
	if caps.Streaming {
		enabled = append(enabled, "Streaming")
	}
	if caps.PushNotifications {
		enabled = append(enabled, "Push Notifications")
	}
	if caps.StateTransitionHistory {
		enabled = append(enabled, "State History")
	}

	if len(enabled) == 0 {
		return
	}
	fmt.Println("\nCapabilities:")
	for _, cap := range enabled {
		fmt.Printf("  • %s\n", cap)
	}
}

func printProvider(provider *a2a.Provider) {
	if provider == nil || provider.Organization == "" {
		return
	}
	fmt.Printf("\nProvider: %s\n", provider.Organization)
	if provider.URL != "" {
		fmt.Printf("          %s\n", provider.URL)
	}
}
