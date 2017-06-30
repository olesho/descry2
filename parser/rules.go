// patterns
package parser

import (
	"errors"
	"regexp"
)

type RegexRules struct {
	Submatch string
	Include  string
	Exclude  string
	Remove   string
}

type CompiledRegexRules struct {
	Submatch *regexp.Regexp
	Include  []*regexp.Regexp
	Exclude  []*regexp.Regexp
	Remove   []*regexp.Regexp
}

func (p *RegexRules) Compile() (*CompiledRegexRules, error) {
	c := &CompiledRegexRules{}
	var err error
	if p != nil {
		if p.Submatch != "" {
			c.Submatch, err = regexp.Compile(p.Submatch)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Submatch' expression: " + p.Submatch)
				return nil, e
			}
		}

		c.Exclude, err = cdataToRegex(p.Exclude)
		if err != nil {
			e := errors.New("Regex 'Exclude' error:" + err.Error())
			return nil, e
		}

		c.Include, err = cdataToRegex(p.Include)
		if err != nil {
			e := errors.New("Regex 'Include' error:" + err.Error())
			return nil, e
		}

		c.Remove, err = cdataToRegex(p.Remove)
		if err != nil {
			e := errors.New("Regex 'Remove' error:" + err.Error())
			return nil, e
		}
	}
	return c, nil
}

func (p *CompiledRegexRules) Test(s []byte) bool {
	if p != nil {
		// URL must apply to at least one "include" patern
		count := len(p.Include)

		if count > 0 {
			for _, r := range p.Include {
				if r.Match(s) {
					break
				} else {
					count--
				}
			}
			if count == 0 {
				return false
			}
		}

		for _, r := range p.Exclude {
			if r.Match(s) {
				return false
			}
		}
	}
	return true
}

func (p *CompiledRegexRules) FindOne(s []byte) []byte {
	if p != nil {
		if p.Submatch != nil {
			res := p.Submatch.FindSubmatch(s)
			if len(res) > 0 {
				return res[1]
			}
			return []byte{}
		}
	}
	return s
}

func (p *CompiledRegexRules) FindMultiple(s []byte) [][]byte {
	if p != nil {
		if p.Submatch != nil {
			found := p.Submatch.FindAllSubmatch(s, -1)
			res := make([][]byte, len(found))
			for i, pair := range found {
				res[i] = pair[1]
			}
			return res
		}
	}
	return [][]byte{s}
}

func (p *CompiledRegexRules) Clean(s []byte) []byte {
	if p != nil {
		for _, r := range p.Remove {
			if r != nil {
				s = r.ReplaceAll(s, []byte{})
			}
		}
	}
	return s
}
