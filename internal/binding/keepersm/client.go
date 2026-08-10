package keepersm

import (
	"fmt"
	"os"
	"strings"

	ksm "github.com/keeper-security/secrets-manager-go/core"
)

// NotationClient resolves Keeper Notation queries to string values.
// Implementations must not log or return secret material through errors.
type NotationClient interface {
	GetNotationResults(notation string) ([]string, error)
}

// sdkClient wraps the official Keeper Secrets Manager Go SDK.
// It reads ambient KSM_CONFIG via the SDK (in-memory; no config file write).
type sdkClient struct {
	sm *ksm.SecretsManager
}

func newSDKClient() (NotationClient, error) {
	if strings.TrimSpace(os.Getenv("KSM_CONFIG")) == "" {
		return nil, fmt.Errorf("KSM_CONFIG is not set (base64 Keeper Secrets Manager device config required)")
	}
	// ClientOptions.Config left nil so the SDK loads KSM_CONFIG into memory storage.
	sm := ksm.NewSecretsManager(&ksm.ClientOptions{})
	if sm == nil {
		return nil, fmt.Errorf("keeper secrets manager client failed to initialize from KSM_CONFIG")
	}
	return &sdkClient{sm: sm}, nil
}

func (c *sdkClient) GetNotationResults(notation string) ([]string, error) {
	values, err := c.sm.GetNotationResults(notation)
	if err != nil {
		// Never include SDK error bodies — they may echo field content.
		return nil, fmt.Errorf("keeper-secrets-manager notation resolve failed")
	}
	return values, nil
}
