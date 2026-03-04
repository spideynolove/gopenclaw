package security

import (
	"fmt"
	"regexp"
	"strings"
)

type Sanitizer struct {
	denyList []*regexp.Regexp
}

func New() *Sanitizer {
	return &Sanitizer{
		denyList: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\b(DROP|DELETE|INSERT|UPDATE|EXEC|EXECUTE|UNION|SELECT|ALTER)\s+)`),
			regexp.MustCompile(`(?i)(<\s*script|javascript:|onerror=|onload=)`),
			regexp.MustCompile(`(?i)(--.*$|/\*.*\*/)`),
			regexp.MustCompile(`(?i)(xp_|sp_|;\s*DROP|;\s*DELETE)`),
			regexp.MustCompile(`(?i)('\s*(OR|AND)\s*'|'\s*=\s*')`),
		},
	}
}

func (s *Sanitizer) IsSuspicious(content string) bool {
	for _, pattern := range s.denyList {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func (s *Sanitizer) ValidateUserMessage(content string) error {
	if content == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	if s.IsSuspicious(content) {
		return fmt.Errorf("message contains potentially malicious content")
	}

	if len(content) > 10000 {
		return fmt.Errorf("message exceeds maximum length of 10000 characters")
	}

	return nil
}

func (s *Sanitizer) ValidateSystemPrompt(prompt string) error {
	if prompt == "" {
		return fmt.Errorf("system prompt cannot be empty")
	}

	if len(prompt) > 50000 {
		return fmt.Errorf("system prompt exceeds maximum length of 50000 characters")
	}

	return nil
}

func (s *Sanitizer) EnsurePromptImmutability(userContent, systemPrompt string) (string, error) {
	if strings.Contains(systemPrompt, "{{USER_INPUT}}") {
		return userContent, nil
	}

	if strings.Contains(userContent, "[SYSTEM]") ||
		strings.Contains(userContent, "[PROMPT]") ||
		strings.Contains(userContent, "Ignore previous instructions") {
		return "", fmt.Errorf("user message attempts to override system prompt")
	}

	return userContent, nil
}

type BlastRadiusPolicy struct {
	WorkspaceOnly bool
}

func (b *BlastRadiusPolicy) AllowsSystemPath() bool {
	return !b.WorkspaceOnly
}

func NewBlastRadiusPolicy(workspaceOnly bool) *BlastRadiusPolicy {
	return &BlastRadiusPolicy{
		WorkspaceOnly: workspaceOnly,
	}
}
