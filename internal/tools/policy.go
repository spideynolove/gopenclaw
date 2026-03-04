package tools

import "github.com/spideynolove/gopenclaw/core"

type PolicyChain struct {
	Global   core.Policy
	Provider core.Policy
	Agent    core.Policy
	Session  core.Policy
	Sandbox  core.Policy
}

func (c PolicyChain) Permitted(toolName string) bool {
	levels := []core.Policy{c.Global, c.Provider, c.Agent, c.Session, c.Sandbox}
	allowed := false
	for _, p := range levels {
		for _, d := range p.Deny {
			if d == toolName {
				return false
			}
		}
		for _, a := range p.Allow {
			if a == toolName {
				allowed = true
			}
		}
	}
	return allowed
}
