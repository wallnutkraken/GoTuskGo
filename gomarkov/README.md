# Markov Chain Algorithm [![License](https://img.shields.io/dub/l/vibe-d.svg)](https://opensource.org/licenses/MIT) [![Go Report Card](https://goreportcard.com/badge/github.com/P-A-R-U-S/Go-Markov-Chain)](https://goreportcard.com/report/github.com/P-A-R-U-S/Go-Markov-Chain) [![Travis-CI](https://travis-ci.org/P-A-R-U-S/Go-Markov-Chain.svg?branch=master)](https://travis-ci.org/P-A-R-U-S/Go-Markov-Chain) [![Get help on Codementor](https://cdn.codementor.io/badges/get_help_github.svg)](https://www.codementor.io/parus) 

A Markov chain algorithm optimized by memory consumption generates text by creating a statistical model of potential textual Suffixes for a given Links.

Generating random text: a Markov chain algorithm

Based on the program presented in the "Design and Implementation" chapter of The Practice of Programming (Kernighan and Pike, Addison-Wesley 1999).
See also Computer Recreations, Scientific American 260, 122 - 125 (1989).

A Markov chain algorithm generates text by creating a statistical model of potential textual suffixes for a given links of chain. Consider this text:

	One fish one fish one fish two fish red fish blue fish

Our Markov chain algorithm would arrange this text into this set of links and suffixes, or "chain": (This table assumes a link length of two words.)

    Link            Suffix
    ========        ==================
    "" ""           [One, 1]
    "" One          [fish, 3] 
    One fish        [One, 2], [two, 1]
    fish one        [fish,2] 
    fish two        [fish,1] 
    Two fish        [red,1]
    fish red        [fish, 1]
    red fish        [blue, 1]
    blue fish       [`END`, 1]

To generate text using this table we select an initial links ("One fish", for example), choose one of the suffixes associated with that prefix at random with probability determined by the input statistics ("One, 2", "fish, 1"), and then create a new links by removing the first word from the link and appending the suffix (making the new link is "fish one" and second "fish two"). Repeat this process until we can't find any suffixes for the current link or we exceed the word limit. (The word limit is necessary as the chain table may contain cycles.)

Current optimized algorithm support similar logic described at following example: https://golang.org/doc/codewalk/markov/, but optimized for memory consumption when base text contains lot of identical suffixes for the same link. 
E.g. in example above following pair of link and suffexes below would contain 2 same suffixes for "one" and 1 for "fish"

```
    Link            Suffix
    ========        ==================
    ...             ...
    One fish        one, one, fish
```
but in optimized version pairs of link and related suffixes would be created in the following way when each suffix contains number of appearances with the appropriate link.

```
    Link            Suffix
    ========        ==================
    ...             ...
    One fish        [One, 2], [two, 1]
```    

Note: keep in mind if your text not long or you're suggesting it is not contains lot of duplicated pair of link and suffixes initial version of algorithm might be more optimized from performance prospectives.

Enjoy coding!!
    
## Example

```GO
    c := NewChain(2) // Words per Links 
    c.Build(strings.NewReader("One fish two fish red fish blue fish))"
    text := c.Generate(250) // Generate text. 
    fmt.Println(text) 
```

### Contributing

We welcome pull requests, bug fixes and issue reports. 

Before proposing a change, please discuss your change by raising an issue.

### Authors

* **Valentyn Ponomarenko** - *Initial work* - [P-A-R-U-S](https://github.com/P-A-R-U-S)

### License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details