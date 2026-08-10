package broker

// Claims are the Cursor workload identity fields used for authorization.
// Email is intentionally omitted from policy matching.
type Claims struct {
	Subject      string
	Audience     string
	Issuer       string
	CloudAgentID string
	AgentRuntime string
	RepoURL      string
	RepoURLs     []string
	RepoCount    int
	OwnerUserID  string
	TeamID       string
	JTI          string
}
