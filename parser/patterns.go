// patterns
package parser

import (
	"encoding/xml"
	"errors"
	//"fmt"
	"io"
	"io/ioutil"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/antchfx/xpath"
	"github.com/antchfx/xquery/html"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v2"
	//"gopkg.in/xmlpath.v2"
)

var lineSplit = regexp.MustCompile(`\n+`)

type Type struct {
	kind reflect.Kind
	//isArray bool
	isHtml bool
}

type PatternNode map[string]interface{}

type Patterns struct {
	Tree *PatternNode
	Log  *log.Logger
}

func NewPatterns(log *log.Logger) *Patterns {
	return &Patterns{
		Tree: &PatternNode{}, //make(map[string]interface{}),
		//Maps: make(map[string]*Map),
		Log: log,
	}
}

func (f *Map) UnmarshalXml(data []byte) error {
	return xml.Unmarshal(data, &f)
}

func (f *Map) UnmarshalYaml(data []byte) error {
	return yaml.Unmarshal(data, &f)
}

func (f *Map) Marshal() ([]byte, error) {
	return xml.Marshal(&f)
}

func CompileType(typeName string) (*Type, error) {
	t := &Type{}

	switch typeName {
	case "int":
		t.kind = reflect.Int
	case "string":
		t.kind = reflect.String
	case "float64":
		t.kind = reflect.Float64
	case "struct":
		t.kind = reflect.Struct
	case "html":
		t.kind = reflect.String
		t.isHtml = true
	default:
		t.kind = reflect.Struct
		//return t, errors.New("Unrecognized type " + typeName)
	}
	return t, nil
}

type Map struct {
	//	Title   string `xml:"title,attr"`
	Storage string `xml:"storage,attr,omitempty"`
	Field   *Field
	URL     *RegexRules
	Mime    string `xml:"mime,attr"`
}

type CompiledMap struct {
	title   string
	storage string
	field   *CompiledField
	url     *CompiledRegexRules
}

func hasExt(fileName, ext string) bool {
	parts := strings.Split(fileName, ".")
	return parts[len(parts)-1] == ext
}

func (p *Patterns) LoadTree(el *PatternNode, path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, f := range files {
		itemName := f.Name()
		if f.IsDir() {
			new_el := &PatternNode{}
			err := p.LoadTree(new_el, path+"/"+itemName)
			if err != nil {
				p.Log.Println(err)
			}
			map[string]interface{}(*el)[itemName] = new_el
		} else {
			data, err := ioutil.ReadFile(path + "/" + itemName)
			if err != nil {
				p.Log.Println(err)
			} else {
				next_pattern := &Map{}
				if hasExt(itemName, "xml") {
					err := next_pattern.UnmarshalXml(data)
					if err != nil {
						p.Log.Println("Pattern "+path+"/"+itemName+" compilation error", err)
					} else {
						// set default type for root element
						if next_pattern.Field.Type == "" {
							next_pattern.Field.Type = "struct"
						}
						compiledPattern, err := next_pattern.Compile()
						if err != nil {
							p.Log.Println("Pattern compilation error: ", err)
						} else {
							map[string]interface{}(*el)[itemName] = compiledPattern
						}
					}
				} else if hasExt(itemName, "yaml") {
					err := next_pattern.UnmarshalYaml(data)
					if err != nil {
						p.Log.Println("Pattern "+path+"/"+itemName+" compilation error", err)
					} else {
						// set default type for root element
						if next_pattern.Field.Type == "" {
							next_pattern.Field.Type = "struct"
						}
						compiledPattern, err := next_pattern.Compile()
						if err != nil {
							p.Log.Println("Pattern compilation error: ", err)
						} else {
							map[string]interface{}(*el)[itemName] = compiledPattern
						}
					}
				}
			}
		}
	}
	return nil
}

func (p *Map) Compile() (*CompiledMap, error) { //(interface{}, error) {
	if p.Mime == "html" {
		m := &CompiledMap{}
		var err error
		m.url, err = p.URL.Compile()
		if err != nil {
			return nil, err
		}

		m.field, err = p.Field.Compile()
		if err != nil {
			return nil, err
		}
		return m, nil
	}
	return nil, nil
}

func ByteToKind(t reflect.Kind, data []byte) (interface{}, error) {
	switch t {
	case reflect.Int:
		val, err := strconv.Atoi(string(data))
		if err != nil {
			return nil, err
		}
		return int(val), nil
	case reflect.String:
		return string(data), nil
	case reflect.Float64:
		//val, err := strconv.FormatFloat(string(data), )
		//return float64(data)
	case reflect.Slice:
		//		return []interface{}(data)
	case reflect.Struct:
		//		return interface{}(data)
	}
	return nil, nil
}

func (p *CompiledMap) ApplyHtml(url string, context *html.Node) interface{} {
	// source URL should be either empty or fit current URL pattern

	if url != "" {
		if !p.url.Test([]byte(url)) {
			return nil
		}
	}

	// retrieve data for root field
	data := p.field.Retrieve(context)
	if data != nil {
		n := make(map[string]interface{})
		n[p.field.title] = data
		return n
	}

	return nil
}

func (p *Patterns) Apply(url string, content io.Reader) (map[string]interface{}, error) {
	//data, err := xmlpath.ParseHTML(content)
	data, err := htmlquery.Parse(content)
	if err != nil {
		return nil, err
	}

	return p.Tree.ApplyPatterns(url, data), nil
}

// Applies XML patterns to input (URL "address" and HTML "content").
// Returns map with result data.
func (pn *PatternNode) ApplyPatterns(url string, data *html.Node /*content io.Reader*/) map[string]interface{} {
	var el map[string]interface{}
	for key, val := range *pn {
		if pattern, ok := val.(*CompiledMap); ok {
			if res := pattern.ApplyHtml(url, data); res != nil {
				if el == nil {
					el = make(map[string]interface{})
				}
				el[key] = res
			}
		} else if subPattern, ok := val.(*PatternNode); ok {
			if res := subPattern.ApplyPatterns(url, data); res != nil {
				if el == nil {
					el = make(map[string]interface{})
				}
				el[key] = res
			}
			//ps.Log.IsError("Error matching pattern", err)
		}
	}

	return el
}

func (pn *PatternNode) ListPatterns() []string {
	res := []string{}
	for key, val := range *pn {
		if _, ok := val.(*Map); ok {
			res = append(res, key)
		} else if subPattern, ok := val.(*PatternNode); ok {
			children := subPattern.ListPatterns()
			for _, childName := range children {
				res = append(res, key+"/"+childName)
			}
		}
	}

	return res
}

// CDATA to xml paths
func cdataToPaths(data string) ([]*xpath.Expr, error) {
	paths := make([]*xpath.Expr, 0)
	lines := lineSplit.Split(data, -1)
	for _, x := range lines {
		x := strings.TrimSpace(x)
		if len(x) > 0 {
			query, err := xpath.Compile(x)
			/*
				query, err := xmlpath.Compile(x)*/
			if err != nil {
				e := errors.New(err.Error() + "\n Path: " + x)
				return nil, e
			}

			paths = append(paths, query)
		}
	}
	return paths, nil
}

// CDATA to regex rules
func cdataToRegex(data string) ([]*regexp.Regexp, error) {
	paths := make([]*regexp.Regexp, 0)
	lines := lineSplit.Split(data, -1)
	for _, x := range lines {
		x := strings.TrimSpace(x)
		if len(x) > 0 {
			query, err := regexp.Compile(x)
			if err != nil {
				e := errors.New(err.Error() + "\n Path: " + x)
				return nil, e
			}
			paths = append(paths, query)
		}
	}
	return paths, nil
}
