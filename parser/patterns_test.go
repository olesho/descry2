// patterns
package parser

import (
	//"fmt"
	//"log"
	"strings"
	"testing"

	"github.com/antchfx/xquery/html"
)

func Test_Retrieve_single(t *testing.T) {
	var testField1 = "Test Title"

	f := &Field{
		Title: "TestField",
		Type:  "string",
		Path:  "//head",
		Data: &RegexRules{
			Submatch: "",
			Include:  "",
			Exclude:  "",
			Remove: `
			^[\x20\x09\x0D\x0A]+
			[\x20\x09\x0D\x0A]+$
			`,
		},
	}
	cf, _ := f.Compile()
	n, _ := htmlquery.Parse(strings.NewReader(`
	<html>
		<head>
			<title>` + testField1 + `</title>
		</head>
	</html>
	`))
	title := cf.Retrieve(n)
	if title != testField1 {
		t.Error(
			"For cf.Retrieve(n) result:",
			"expected", testField1,
			"got", title,
		)
	}
}

func TestRetrieve_singleHtmlSource(t *testing.T) {
	var testField1 = `<head>
	<title>testField1</title>
</head>`

	f := &Field{
		Title: "TestField",
		Type:  "html",
		Path:  "//head",
		Data: &RegexRules{
			Submatch: "",
			Include:  "",
			Exclude:  "",
			Remove: `
			^[\x20\x09\x0D\x0A]+
			[\x20\x09\x0D\x0A]+$
			`,
		},
	}
	cf, _ := f.Compile()
	n, _ := htmlquery.Parse(strings.NewReader(`
	<html>` + testField1 + `</html>
	`))
	title := cf.Retrieve(n)
	if title != testField1 {
		t.Error(
			"For cf.Retrieve(n) result:",
			"expected", testField1,
			"got", title,
		)
	}
}

func Test_Retrieve_multiple(t *testing.T) {
	var (
		testField1 = "Test Title"
		testField2 = "Test Field"
		testField3 = "Test Something"
	)

	var testFields = []string{testField1, testField2, testField3}

	f := &Field{
		Title:    "TestField",
		Type:     "[]string",
		Path:     "//body/ul/li",
		Multiple: true,
		Data: &RegexRules{
			Submatch: "",
			Include:  "",
			Exclude:  "",
			Remove: `
			^[\x20\x09\x0D\x0A]+
			[\x20\x09\x0D\x0A]+$
			`,
		},
	}
	cf, _ := f.Compile()
	n, _ := htmlquery.Parse(strings.NewReader(`
	<html>
		<head></head>
		<body>
			<ul>
				<li>` + testField1 + `</li>
				<li>` + testField2 + `</li>
				<li>` + testField3 + `</li>
			</ul>
		</body>
	</html>
	`))
	titles := cf.Retrieve(n).([]interface{})
	for n, title := range titles {
		if title != testFields[n] {
			t.Error(
				"For cf.Retrieve(n) result #", n,
				"expected", testFields[n],
				"got", title,
			)
		}
	}
}
