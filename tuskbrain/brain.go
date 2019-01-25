// Package tuskbrain contains the core of GoTuskGo, the markov chain object and its related functions
package tuskbrain

import (
	"github.com/wallnutkraken/gotuskgo/gomarkov"
	"github.com/wallnutkraken/gotuskgo/stringer"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
)

// Brain contains the GoTuskBot brain and associated generation functions
type Brain struct {
	chain  *gomarkov.Chain
	config settings.Brain
}

// New creates a new instance of the TUSK brain
func New(brainSettings settings.Brain) Brain {
	return Brain{
		chain:  gomarkov.NewChain(brainSettings.ChainLength),
		config: brainSettings,
	}
}

// Feed feeds the given messages to the bot markov chain
func (b Brain) Feed(messages ...string) {
	for _, msg := range messages {
		b.chain.Feed(stringer.SplitMultiple(msg, b.config.SplitChars))
	}
}

// Generate creates a new string from the bot brain
func (b Brain) Generate() string {
	return b.chain.Generate(b.config.MaxGeneratedLength)
}
