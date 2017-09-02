// patterns
package parser

import (
	"fmt"
	//"log"
	"strings"
	"testing"

	"github.com/antchfx/xquery/html"
	"github.com/stretchr/testify/assert"
	//"github.com/stretchr/testify/require"
)

func Test_Retrieve_single(t *testing.T) {
	fmt.Println("Retrieve_single...")
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

func Test_Retrieve_multiple(t *testing.T) {
	fmt.Println("Retrieve_multiple...")
	var (
		testField1 = "Test Title"
		testField2 = "Test Field"
		testField3 = "Test Something"
	)

	var testFields = []string{testField1, testField2, testField3}

	f := &Field{
		Title:    "TestField",
		Type:     "string",
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

func TestRetrieve_singleHtmlSource(t *testing.T) {
	fmt.Println("singleHtmlSource ...")
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

func TestRetrieve_multipleHtmlSource(t *testing.T) {
	fmt.Println("multipleHtmlSource ...")
	f := &Field{
		Title:    "TestField",
		Multiple: true,
		Type:     "html",
		Path:     "//div",
		Data:     &RegexRules{},
	}
	cf, _ := f.Compile()
	n, _ := htmlquery.Parse(strings.NewReader(`<html>
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
	</html>`))
	titles := cf.Retrieve(n).([]interface{})
	if len(titles) != 6 {
		t.Error(
			"For cf.Retrieve(n) result:",
			"expected: 6 results",
			"got:", len(titles),
		)
	}
}

func TestLoadXml(t *testing.T) {
	pn := NewPatterns(nil)
	data := []byte(xmlStr)
	err := pn.LoadXml(pn.Tree, data, "test.xml")
	if err != nil {
		t.Errorf("Error loading pattern", err)
	}

	m := map[string]interface{}(*pn.Tree)["test.xml"].(*CompiledMap)

	assert.Equal(t, m.field.title, "Item")
	assert.Equal(t, m.field.path[0].String(), "//table[@class='itemlist']/tbody/tr/td[@class='title']")
	assert.Equal(t, m.field.field[0].title, "Link")
	assert.Equal(t, m.field.field[0].path[0].String(), "a[contains(@href, 'item?id=')]/@href")
	assert.Equal(t, m.field.field[1].title, "Title")
	assert.Equal(t, m.field.field[1].path[0].String(), "a[contains(@href, 'item?id=')]")
}

var xmlStr = `
		<Pattern mime="html">
			<URL>
				<Include><![CDATA[
					^https://news.ycombinator.com/jobs
				]]></Include>
			</URL>
			<Field title="Item" type="struct" multiple="true">
				<Path>
					<![CDATA[
						//table[@class='itemlist']/tbody/tr/td[@class='title']
					]]>
				</Path>
				<Field title="Link" type="string">
					<Path>
						a[contains(@href, 'item?id=')]/@href
					</Path>
				</Field>
				<Field title="Title" type="string">
					<Path>
						a[contains(@href, 'item?id=')]
					</Path>
				</Field>
			</Field>
		</Pattern>
`

var ymlStr1 = `

  field:
    title: Item
    type: struct
    path: "//table[@class='itemlist']/tbody/tr/td[@class='title']"
    data: null
    xdata: null
    optional: false
    dontstore: false
    multiple: true
    unique: false
    attr: false
    field:
    - title: Link
      type: string
      path: "a[contains(@href, 'item?id=')]/@href"
      data: null
      xdata: null
      optional: false
      dontstore: false
      multiple: false
      unique: false
      attr: false
      field: []
    - title: Title
      type: string
      path: "\n\t\t\t\ta[contains(@href, 'item?id=')]"
      data: null
      xdata: null
      optional: false
      dontstore: false
      multiple: false
      unique: false
      attr: false
      field: []
  url:
    submatch: ""
    include: "^https://news.ycombinator.com/jobs"
    exclude: ""
    remove: ""
  mime: html`

var ymlStr2 = `pattern:
  url:
    include:
      - ^https://news.ycombinator.com/jobs
      - field:
        title: Item
        type: struct
        multiple: true
        path:
          - //table[@class='itemlist']/tbody/tr/td[@class='title']
        - field:
          title: Link
          type: string
          path:
          - a[contains(@href, 'item?id=')]/@href
        - field:
          title: Title
          type: string
          path:
          - a[contains(@href, 'item?id=')]`

func TestLoadYml(t *testing.T) {
	pn := NewPatterns(nil)
	data := []byte(ymlStr1)

	err := pn.LoadYaml(pn.Tree, data, "test.yml")
	if err != nil {
		t.Errorf("Error loading pattern", err)
	}

	m := map[string]interface{}(*pn.Tree)["test.yml"].(*CompiledMap)

	assert.Equal(t, m.field.title, "Item")
	assert.Equal(t, m.field.path[0].String(), "//table[@class='itemlist']/tbody/tr/td[@class='title']")
	assert.Equal(t, m.field.field[0].title, "Link")
	assert.Equal(t, m.field.field[0].path[0].String(), "a[contains(@href, 'item?id=')]/@href")
	assert.Equal(t, m.field.field[1].title, "Title")
	assert.Equal(t, m.field.field[1].path[0].String(), "a[contains(@href, 'item?id=')]")
}
