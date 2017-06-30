// patterns
package parser

import (
	"errors"
	//"fmt"

	"github.com/antchfx/xpath"
	"github.com/antchfx/xquery/html"
	"golang.org/x/net/html"
)

type XpathRules struct {
	Include string
	Exclude string
	Remove  string
}

type CompiledXpathRules struct {
	Include []*xpath.Expr
	Exclude []*xpath.Expr
	Remove  []*xpath.Expr
}

// brings to html cdata to compiled form
func (p *XpathRules) Compile() (*CompiledXpathRules, error) {
	c := &CompiledXpathRules{}
	var err error
	if p != nil {
		c.Exclude, err = cdataToPaths(p.Exclude)
		if err != nil {
			e := errors.New("XPath 'Exclude' error:" + err.Error())
			return nil, e
		}

		c.Include, err = cdataToPaths(p.Include)
		if err != nil {
			e := errors.New("XPath 'Include' error:" + err.Error())
			return nil, e
		}

		c.Remove, err = cdataToPaths(p.Remove)
		if err != nil {
			e := errors.New("XPath 'Remove' error:" + err.Error())
			return nil, e
		}
	}
	return c, nil
}

// test if complies to defined rules
func (p *CompiledXpathRules) Test(s xpath.NodeNavigator) bool {
	if p != nil {
		// URL must apply to at least one "include" patern
		count := len(p.Include)

		if count > 0 {
			for _, r := range p.Include {
				if r.Select(s).MoveNext() {
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
			if r.Select(s).MoveNext() {
				return false
			}
		}
	}
	return true
}

func (p *CompiledXpathRules) Clean(s *html.Node) *html.Node {
	var list []*html.Node
	if p != nil {
		for _, r := range p.Remove {
			if r != nil {
				htmlquery.FindEach(s, r.String(), func(n int, root *html.Node) {
					//root.Parent.RemoveChild(root)
					list = append(list, root)
				})
			}
		}
	}

	for _, toRemove := range list {
		iterateNodes(s, func(root *html.Node) {
			if toRemove == root {
				root.Parent.RemoveChild(root)
			}
		})
	}

	return s
}

func iterateNodes(r *html.Node, cb func(n *html.Node)) {
	for c := r.FirstChild; c != nil; c = c.NextSibling {
		cb(c)
		iterateNodes(c, cb)
	}
}
