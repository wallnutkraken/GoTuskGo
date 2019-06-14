// Package gomarkov
// Copyright 2018 (c) Valentyn Ponomarenko.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.
/*
Generating random text: a Markov chain algorithm

Based on the program presented in the "Design and Implementation" chapter
of The Practice of Programming (Kernighan and Pike, Addison-Wesley 1999).
See also Computer Recreations, Scientific American 260, 122 - 125 (1989).

A Markov chain algorithm generates text by creating a statistical model of
potential textual suffixes for a given links of chain. Consider this text:

	One fish One fish One fish two fish red fish blue fish

Our Markov chain algorithm would arrange this text into this set of prefixes
and suffixes, or "chain": (This table assumes a prefix length of two words.)

	Link       	  Suffix

	"" ""        [One, 1]
	"" One       [fish, 3]
	One fish     [One, 2], [two, 1]
	Two fish     [red,1]
	fish red     [fish, 1]
	red fish     [blue, 1]
	blue fish    [`END`, 1]

To generate text using this table we select an initial links ("One fish", for
example), choose one of the suffixes associated with that prefix at random
with probability determined by the input statistics ("a"),
and then create a new prefix by removing the first word from the prefix
and appending the suffix (making the new prefix is "am a"). Repeat this process
until we can't find any suffixes for the current prefix or we exceed the word
limit. (The word limit is necessary as the chain table may contain cycles.)

Our version of this program reads text from standard input, parsing it into a
Markov chain, and writes generated text to standard output.
The prefix and output lengths can be specified using the -prefix and -words
flags on the command-line.
*/
package gomarkov

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"strings"
)

// Link is a Markov chain prefix of one or more words.
type Link []string

// String represented Link as a string (for use as a map key).
func (p Link) String() string {
	return strings.Join(p, " ")
}

// Shift removes the first word from the Link and appends the given word.
func (p Link) Shift(word string) {
	copy(p, p[1:])
	p[len(p)-1] = word
}

// Chain contains a map ("chain") of links to a list of suffixes/pairs.
// A Link is a string of prefixLen words joined with spaces.
// A Pair is a single word and number of appearance after Link. A Link can have multiple Pairs.
type Chain struct {
	chain       map[string]map[string]int
	linksLength int
}

// NewChain returns a new Chain with prefixes of prefixLen words.
func NewChain(linksLength int) *Chain {
	return &Chain{make(map[string]map[string]int), linksLength}
}

// SetLength sets the chain length
func (c *Chain) SetLength(length int) {
	c.linksLength = length
}

// Build reads text from the provided Reader and
// parses it into prefixes and suffixes that are stored in Chain.
func (c *Chain) Build(r io.Reader) {

	br := bufio.NewReader(r)
	l := make(Link, c.linksLength)
	for {
		var s string
		if _, err := fmt.Fscan(br, &s); err != nil {
			break
		}
		key := l.String()

		// Add key if not exist with empty map
		if _, ok := c.chain[key]; !ok {
			c.chain[key] = make(map[string]int)
		}
		c.chain[key][s] = c.chain[key][s] + 1
		l.Shift(s)
	}
}

// Feed feeds the given words into the Chain
func (c *Chain) Feed(words []string) {
	l := make(Link, c.linksLength)
	for _, s := range words {
		key := l.String()
		// Add key if not exist with empty map
		if _, ok := c.chain[key]; !ok {
			c.chain[key] = make(map[string]int)
		}
		c.chain[key][s] = c.chain[key][s] + 1
		l.Shift(s)
	}
}

// Generate returns a string of at most n words generated from Chain.
func (c *Chain) Generate(n int) string {
	l := make(Link, c.linksLength)
	var words []string
	for i := 0; i < n; i++ {
		choices := c.chain[l.String()]
		if len(choices) == 0 {
			break
		}

		choicesLen := 0
		for _, v := range choices {
			choicesLen += v
		}

		index := rand.Intn(choicesLen)

		next := ""
		for k, v := range choices {
			if (index - v) <= 0 {
				next = k
				break
			}
			index -= v
		}

		words = append(words, next)
		l.Shift(next)
	}
	return strings.Join(words, " ")
}
