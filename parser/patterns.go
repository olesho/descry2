// patterns
package parser

import (
	"encoding/xml"
	"errors"
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

//var lineStripPre = regexp.MustCompile(`^[\t\s]`)
//var lineStripPost = regexp.MustCompile(`[\t\s]$`)

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

type Type struct {
	kind    reflect.Kind
	isArray bool
}

type Field struct {
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
	Path  string
	Data  *RegexRules

	Optional  bool `xml:"optional,attr,omitempty"`
	DontStore bool `xml:"dontstore,attr,omitempty"`
	Multiple  bool `xml:"multiple,attr,omitempty"`
	Unique    bool `xml:"unique,attr,omitempty"`

	Field []*Field
}

type CompiledField struct {
	title    string
	dataType *Type
	path     []*xpath.Expr //[]string //[]*xmlpath.Path
	data     *CompiledRegexRules
	parent   *CompiledField

	optional  bool `xml:"optional,attr,omitempty"`
	dontStore bool `xml:"dontstore,attr,omitempty"`
	multiple  bool `xml:"multiple,attr,omitempty"`
	unique    bool `xml:"unique,attr,omitempty"`

	field []*CompiledField
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

func (f *Field) Serialize() []*Field {
	result := []*Field{f}
	for _, field := range f.Field {
		result = append(result, field.Serialize()...)
	}
	return result
}

func (f *Field) FindChildField(name string) *Field {
	for _, field := range f.Field {
		if field.Title == name {
			return field
		}
	}
	return nil
}

func (f *Field) FindField(addr string) *Field {
	parts := strings.Split(addr, ".")
	result := f
	for _, p := range parts {
		result = f.FindChildField(p)
	}
	return result
}

// build "parent" dependency
func (f *CompiledField) setParent() {
	for _, field := range f.field {
		field.parent = f
		field.setParent()
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

// get relative path like: "Root.SubField1.Subfield2..."
func (f *CompiledField) RelativePath() string {
	if f.parent == nil {
		return f.title
	} else {
		return f.parent.RelativePath() + "." + f.title
	}
}

func CompileType(typeName string) (*Type, error) {
	t := &Type{}
	if strings.Contains(typeName, "[]") {
		t.isArray = true
	}
	typeName = strings.Replace(typeName, "[]", "", -1)

	switch typeName {
	case "int":
		t.kind = reflect.Int
	case "string":
		t.kind = reflect.String
	case "float64":
		t.kind = reflect.Float64
	case "array":
		t.kind = reflect.Slice
	case "struct":
		t.kind = reflect.Struct
	default:
		t.kind = reflect.Struct
		return t, errors.New("Unrecognized type " + typeName)
	}
	return t, nil
}

func (f *Field) Compile() (*CompiledField, error) {
	c := &CompiledField{}

	var err error
	c.path, err = cdataToPaths(f.Path)
	if err != nil {
		return nil, err
	}

	c.data, err = f.Data.Compile()
	if err != nil {
		return nil, err
	}
	c.field = make([]*CompiledField, 0)
	for _, field := range f.Field {
		compiledField, err := field.Compile()
		if err != nil {
			return nil, err
		}
		c.field = append(c.field, compiledField)
	}

	if f.Type == "" {
		return nil, errors.New("Failed to compile " + f.Title + ". Field missing type")
	}

	c.dataType, err = CompileType(f.Type)
	if err != nil {
		return nil, err
	}

	c.title = f.Title
	c.unique = f.Unique
	c.dontStore = f.DontStore
	c.multiple = f.Multiple
	c.optional = f.Optional

	return c, nil
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

func (f *CompiledField) Retrieve(root *html.Node) (result interface{}) {
	// is "f" has no children and so is simple type, like: int, string, float64, etc.
	if f.dataType.kind != reflect.Struct {
		// check every Path provided
		for _, query := range f.path {
			if f.dataType.isArray {
				res := make([]interface{}, 0)
				iter := query.Evaluate(htmlquery.CreateXPathNavigator(root)).(*xpath.NodeIterator)
				for iter.MoveNext() {
					// try to find each context
					bts := []byte(iter.Current().Value())

					test := f.data.Test(bts)
					if test {
						found := f.data.FindMultiple(bts)
						for _, nextVal := range found {
							cut := f.data.Clean(nextVal)
							val, err := ByteToKind(f.dataType.kind, cut)
							if err != nil {
								// cannot convert
								//fmt.Println(err)
							} else {
								if f.unique {
									unique := true
									for _, v := range res {
										// not sure if it will work???
										if v == val {
											unique = false
										}
									}
									if unique {
										res = append(res, val)
									}
								} else {
									res = append(res, val)
								}
							}
						}
					}
				}
				result = interface{}(res)
			} else {
				iter := query.Evaluate(htmlquery.CreateXPathNavigator(root)).(*xpath.NodeIterator)
				if iter.MoveNext() {
					val := []byte(iter.Current().Value())
					test := f.data.Test(val)
					if test {
						cut := f.data.Clean(val)
						found := f.data.FindOne(cut)
						var err error
						result, err = ByteToKind(f.dataType.kind, found)
						if err != nil {
							// cant convert
							//fmt.Println(err)
						}
					}
				}
			}

			// exit if at least 1 value found for Path
			if result != nil {
				return result
			}
		}
	} else {
		if f.dataType.isArray {
			res := make([]interface{}, 0)

			// only one path available works for struct
			htmlquery.FindEach(root, f.path[0].String(), func(N int, subRootIter *html.Node) {
				// try to find each context
				val := make(map[string]interface{})
				for _, child_field := range f.field {
					r := child_field.Retrieve(subRootIter)

					if r == nil && !child_field.optional {
						//result = nil
						val = nil
						break
					}

					val[child_field.title] = r
				}
				if val != nil {
					res = append(res, val)
				}
			})

			result = res
		} else {
			val := make(map[string]interface{})

			subRoot := htmlquery.FindOne(root, f.path[0].String())
			//iter := f.path[0].Evaluate(root).(*xpath.NodeIterator)
			//if iter.MoveNext() {
			if subRoot != nil {
				for _, child_field := range f.field {
					//r := child_field.Retrieve(iter.Node())
					r := child_field.Retrieve(subRoot)

					if r == nil && !child_field.optional {
						//result = nil
						val = nil
						break
					}

					val[child_field.title] = r
				}
				if val != nil {
					result = val
				}
			}
		}
	}

	return result
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
