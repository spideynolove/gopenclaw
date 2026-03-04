package tools_test

import (
	"testing"

	"github.com/spideynolove/gopenclaw/core"
	"github.com/spideynolove/gopenclaw/internal/tools"
)

func TestDenyOverridesAllow(t *testing.T) {
	chain := tools.PolicyChain{
		Global: core.Policy{Allow: []string{"web_search"}},
		Agent:  core.Policy{Deny: []string{"web_search"}},
	}
	if chain.Permitted("web_search") {
		t.Error("deny should override allow")
	}
}

func TestAllowedTool(t *testing.T) {
	chain := tools.PolicyChain{
		Global: core.Policy{Allow: []string{"memory_write"}},
	}
	if !chain.Permitted("memory_write") {
		t.Error("memory_write should be permitted")
	}
}

func TestUnlistedToolDenied(t *testing.T) {
	chain := tools.PolicyChain{
		Global: core.Policy{Allow: []string{"memory_write"}},
	}
	if chain.Permitted("unknown_tool") {
		t.Error("unlisted tool should be denied")
	}
}

func TestMultipleLevelsAllow(t *testing.T) {
	chain := tools.PolicyChain{
		Global:   core.Policy{Allow: []string{"web_search"}},
		Provider: core.Policy{Allow: []string{"memory_write"}},
		Agent:    core.Policy{Allow: []string{"math_eval"}},
	}
	if !chain.Permitted("web_search") {
		t.Error("web_search should be permitted via global")
	}
	if !chain.Permitted("memory_write") {
		t.Error("memory_write should be permitted via provider")
	}
	if !chain.Permitted("math_eval") {
		t.Error("math_eval should be permitted via agent")
	}
}

func TestDenyAtHigherLevel(t *testing.T) {
	chain := tools.PolicyChain{
		Global:  core.Policy{Allow: []string{"web_search"}},
		Session: core.Policy{Deny: []string{"web_search"}},
	}
	if chain.Permitted("web_search") {
		t.Error("deny at session level should override global allow")
	}
}

func TestEmptyPolicyDeniesAll(t *testing.T) {
	chain := tools.PolicyChain{}
	if chain.Permitted("any_tool") {
		t.Error("empty policy should deny all tools")
	}
}
