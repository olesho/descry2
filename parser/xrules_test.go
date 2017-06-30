// patterns
package parser

import (
	//"bytes"
	//"fmt"
	"strings"
	"testing"

	"github.com/antchfx/xquery/html"
	"golang.org/x/net/html"
)

func Test_Clean(t *testing.T) {
	r := &XpathRules{
		Remove: `
		//p
		//h3
		`,
	}

	rc, _ := r.Compile()

	n, _ := htmlquery.Parse(strings.NewReader(`
	<html>
		<head>
			<title>Test</title>
		</head>
		<body>
			<h1>Test Sample</h1>
			<h2>This should be left</h2>
			<p>This should be filtered out</p>
			<h2>This should be left too</h2>
			<p>This should be removed</p>
			<h3>Leave this</h3>
		</body>
	</html>
	`))

	cleanNode := rc.Clean(n)
	iterateNodes(cleanNode, func(next *html.Node) {
		if next.Data == "p" || next.Data == "h3" {
			t.Error("xrules.Clean func not working properly")
		}
	})

	/*
		var b []byte
		buf := bytes.NewBuffer(b)
		html.Render(buf, cleanNode)
		fmt.Println(buf.String())
	*/
}

func Test_Include(t *testing.T) {
	r := &XpathRules{
		Include: `
		//div/p
		`,
	}

	rc, _ := r.Compile()

	n, _ := htmlquery.Parse(strings.NewReader(`
	<html>
		<head>
			<title>Test</title>
		</head>
		<body>
			<div>
				<h1>Test Sample</h1>
			</div>
			<div>
				<h2>This should be left</h2>
			</div>
			<div>
				<p>This should be filtered out</p>
			</div>
			<div>
				<h2>This should be left too</h2>
			</div>
			<div>
				<p>This should be removed</p>
			</div>
			<div>
				<h3>Leave this</h3>
			</div>
		</body>
	</html>
	`))

	if !rc.Test(htmlquery.CreateXPathNavigator(n)) {
		t.Error("xrules.Include func not working properly")
	}
}

func Test_Exclude(t *testing.T) {
	r := &XpathRules{
		Exclude: `
		//div/p
		`,
	}

	rc, _ := r.Compile()

	n, _ := htmlquery.Parse(strings.NewReader(`
	<html>
		<head>
			<title>Test</title>
		</head>
		<body>
			<div>
				<h1>Test Sample</h1>
			</div>
			<div>
				<h2>This should be left</h2>
			</div>
			<div>
				<p>This should be filtered out</p>
			</div>
			<div>
				<h2>This should be left too</h2>
			</div>
			<div>
				<p>This should be removed</p>
			</div>
			<div>
				<h3>Leave this</h3>
			</div>
		</body>
	</html>
	`))

	if rc.Test(htmlquery.CreateXPathNavigator(n)) {
		t.Error("xrules.Exclude func not working properly")
	}
}

/*
func Test_iterateNodes(t *testing.T) {
	n, _ := htmlquery.Parse(strings.NewReader(`
	<html>
		<head>
			<title>Test</title>
		</head>
		<body>
			<h1>Test Sample</h1>
			<br>This should be left</br>
			<p>This should be filtered out</p>
			<br>This should be left too</br>
			<p>This should be removed</p>
		</body>
	</html>
	`))

	iterateNodes(n, func(next *html.Node) {
		fmt.Println(next.Type, next.Data)
	})
}
*/
