package a2a

// AgentCard represents an A2A protocol agent card.
// Based on A2A spec: https://a2a-protocol.org/latest/specification/
type AgentCard struct {
	// Required fields
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	// Skills
	Skills []Skill `json:"skills,omitempty"`

	// Capabilities
	Capabilities *Capabilities `json:"capabilities,omitempty"`

	// Optional fields
	ProtocolVersion  string    `json:"protocolVersion,omitempty"`
	Provider         *Provider `json:"provider,omitempty"`
	DocumentationURL string    `json:"documentationUrl,omitempty"`
	IconURL          string    `json:"iconUrl,omitempty"`

	// Input/output modes
	DefaultInputModes  []string `json:"defaultInputModes,omitempty"`
	DefaultOutputModes []string `json:"defaultOutputModes,omitempty"`

	// Interfaces
	SupportedInterfaces []Interface `json:"supportedInterfaces,omitempty"`

	// URL (endpoint where the A2A service can be reached)
	URL string `json:"url,omitempty"`
}

// Skill represents an agent skill/capability.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// Capabilities describes agent capabilities.
type Capabilities struct {
	Streaming              bool `json:"streaming,omitempty"`
	PushNotifications      bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
}

// Provider contains agent provider information.
type Provider struct {
	Organization string `json:"organization,omitempty"`
	URL          string `json:"url,omitempty"`
}

// Interface represents a supported protocol interface.
type Interface struct {
	Protocol string `json:"protocol,omitempty"`
	URL      string `json:"url,omitempty"`
}

// Result is the complete result of agent card discovery.
type Result struct {
	URL           string        `json:"url"`                     // Original URL provided
	BaseURL       string        `json:"baseUrl"`                 // Extracted base URL
	Found         bool          `json:"found"`                   // Whether a card was found
	DiscoveryPath string        `json:"discoveryPath,omitempty"` // Path where card was found
	Card          *AgentCard    `json:"card,omitempty"`          // The discovered card
	TriedPaths    []PathAttempt `json:"triedPaths,omitempty"`    // All paths attempted
	ExitCode      int           `json:"exitCode"`                // Exit code for CLI
	Error         string        `json:"error,omitempty"`         // Error message if any
}

// PathAttempt records a discovery attempt at a specific path.
type PathAttempt struct {
	Path   string `json:"path"`
	Status int    `json:"status"`          // HTTP status code (0 if network error)
	Error  string `json:"error,omitempty"` // Error message if any
}
