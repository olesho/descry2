// patterns
package parser

import (
	"encoding/xml"
	"errors"
	//"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/olesho/descry"
	//"bitbucket.org/olesho/scrub/logging"
	"gopkg.in/xmlpath.v2"
)

type RegexRules struct {
	RegexSubmatch string
	RegexInclude  []string
	RegexExclude  []string
	RegexRemove   []string

	regexSubmatchCompiled *regexp.Regexp
	regexIncludeCompiled  []*regexp.Regexp
	regexExcludeCompiled  []*regexp.Regexp
	regexRemoveCompiled   []*regexp.Regexp
}

type Type struct {
	Name string `xml:"name,attr"`

	kind    reflect.Kind
	isArray bool
}

type Field struct {
	Title     string `xml:"title,attr"`
	Type      *Type
	XPath     []string
	DataRules *RegexRules

	Optional  bool `xml:"optional,attr,omitempty"`
	DontStore bool `xml:"dontstore,attr,omitempty"`
	Multiple  bool `xml:"multiple,attr,omitempty"`
	Unique    bool `xml:"unique,attr,omitempty"`

	Field []*Field

	xquery []*xmlpath.Path `xml:"-"`
	parent *Field          `xml:"-"`
}

type PatternNode map[string]interface{}

type Patterns struct {
	HtmlPatternTree *PatternNode

	//HtmlMaps map[string]*HtmlMap
	//JsonMaps map[string]*JsonMap
	Log *descry.Logger
}

func NewPatterns(log *descry.Logger) *Patterns {
	//	log := logging.NewLogger()
	//	log.Level = logging.LEVEL_DEBUG
	return &Patterns{
		HtmlPatternTree: &PatternNode{}, //make(map[string]interface{}),
		//HtmlMaps: make(map[string]*HtmlMap),
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
func (f *Field) setParent() {
	for _, field := range f.Field {
		field.parent = f
		field.setParent()
	}
}

func (f *HtmlMap) Unmarshal(data []byte) error {
	err := xml.Unmarshal(data, &f)
	f.Field.setParent()
	return err
}

func (f *HtmlMap) Marshal() ([]byte, error) {
	return xml.Marshal(&f)
}

// get relative path like: "Root.SubField1.Subfield2..."
func (f *Field) Path() string {
	if f.parent == nil {
		return f.Title
	} else {
		return f.parent.Path() + "." + f.Title
	}
}

func (t *Type) Compile() error {
	if strings.Contains(t.Name, "[]") {
		t.isArray = true
	}
	typeName := strings.Replace(t.Name, "[]", "", -1)

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
		return errors.New("Unrecognized type " + typeName)
	}
	return nil
}

func (f *Field) Compile() error {
	//if f != nil {
	for _, x := range f.XPath {
		query, err := xmlpath.Compile(x)
		if err != nil {
			e := errors.New(err.Error() + "\n Path: " + x)
			return e
		}
		f.xquery = append(f.xquery, query)
	}

	f.DataRules.Compile()
	for _, c := range f.Field {
		err := c.Compile()
		if err != nil {
			return err
		}
	}

	if f.Type == nil {
		return errors.New("Failed to compile " + f.Title + ". Field missing type")
	}

	err := f.Type.Compile()
	if err != nil {
		return err
	}
	//}
	return nil
}

type HtmlMap struct {
	Title    string `xml:"title,attr"`
	Storage  string `xml:"storage,attr,omitempty"`
	Field    *Field
	URLRules *RegexRules
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
			p.LoadTree(new_el, path+"/"+itemName)
			map[string]interface{}(*el)[itemName] = new_el
		} else {
			data, err := ioutil.ReadFile(path + "/" + itemName)
			if err != nil {
				p.Log.IsError("", err)
			} else {
				next_pattern := &HtmlMap{}
				next_pattern.Unmarshal(data)

				// set default type for root element
				if next_pattern.Field.Type == nil {
					next_pattern.Field.Type = &Type{Name: "struct"}
				}
				err = next_pattern.Compile()
				if err != nil {
					p.Log.IsError("Pattern compilation error: ", err)
				} else {
					map[string]interface{}(*el)[itemName] = next_pattern
				}
			}
		}
	}
	return nil
}

/*
func (p *Patterns) LoadAppend(dir string, root bool) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			err := p.LoadAppend(dir+"/"+f.Name(), false)
			if err != nil {
				p.Log.IsError("", err)
			}
		} else {
			parts := strings.Split(f.Name(), ".")
			ext := parts[len(parts)-1]
			data, err := ioutil.ReadFile(dir + "/" + f.Name())
			if err != nil {
				p.Log.IsError("", err)
			} else {
				var key string
				if root {
					key = f.Name()
				} else {
					key = dir + "/" + f.Name()
				}
				switch ext {
				case "xml":
					next_pattern := &HtmlMap{}
					next_pattern.Unmarshal(data)

					// set default type for root element
					if next_pattern.Field.Type == nil {
						next_pattern.Field.Type = &Type{Name: "struct"}
					}
					err = next_pattern.Compile()
					if err != nil {
						p.Log.IsError("Pattern compilation error: ", err)
					} else {
						p.HtmlMaps[key] = next_pattern
					}
				}
			}
		}
	}
	return nil
}
*/

/*
func (ps *Patterns) AddPattern(title string, xmldata []byte) error {
	err := ps.HtmlMaps[title].Unmarshal(xmldata)
	if err != nil {
		return err
	}

	err = ps.HtmlMaps[title].URLRules.Compile()
	if err != nil {
		e := errors.New(err.Error() + "\n Pattern: " + title)
		return e
	}

	return ps.HtmlMaps[title].Field.Compile()
}
*/

func (p *HtmlMap) Compile() error {
	err := p.URLRules.Compile()
	if err != nil {
		return err
	}

	return p.Field.Compile()
}

func (p *RegexRules) Test(s []byte) bool {
	if p != nil {
		// URL must apply to at least one "include" patern
		count := len(p.regexIncludeCompiled)

		if count > 0 {
			for _, r := range p.regexIncludeCompiled {
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

		for _, r := range p.regexExcludeCompiled {
			if r.Match(s) {
				return false
			}
		}
	}
	return true
}

func (p *RegexRules) Clean(s []byte) []byte {
	if p != nil {
		if p.regexSubmatchCompiled != nil {
			s = p.regexSubmatchCompiled.Find(s)
		}
		for _, r := range p.regexRemoveCompiled {
			s = r.ReplaceAll(s, []byte{})
		}
	}
	return s
}

func (p *RegexRules) Compile() error {
	if p != nil {
		if p.RegexSubmatch != "" {
			text, err := strconv.Unquote(`"` + p.RegexSubmatch + `"`)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Submatch' expression: " + p.RegexSubmatch)
				return e
			}

			p.regexSubmatchCompiled, err = regexp.Compile(text)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Submatch' expression: " + p.RegexSubmatch)
				return e
			}
		}
		p.regexExcludeCompiled = make([]*regexp.Regexp, len(p.RegexExclude))
		for i, r := range p.RegexExclude {
			text, err := strconv.Unquote(`"` + r + `"`)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Exclude' expression: " + r)
				return e
			}

			p.regexExcludeCompiled[i], err = regexp.Compile(text)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Exclude' expression: " + r)
				return e
			}
		}
		p.regexIncludeCompiled = make([]*regexp.Regexp, len(p.RegexInclude))
		for i, r := range p.RegexInclude {
			text, err := strconv.Unquote(`"` + r + `"`)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Include' expression: " + r)
				return e
			}

			p.regexIncludeCompiled[i], err = regexp.Compile(text)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Include' expression: " + r)
				return e
			}
		}
		p.regexRemoveCompiled = make([]*regexp.Regexp, len(p.RegexRemove))
		for i, r := range p.RegexRemove {
			text, err := strconv.Unquote(`"` + r + `"`)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Remove' expression: " + r)
				return e
			}

			p.regexRemoveCompiled[i], err = regexp.Compile(text)
			if err != nil {
				e := errors.New(err.Error() + "\n Regex 'Remove' expression: " + r)
				return e
			}
		}
	}
	return nil
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

func (f *Field) Retrieve(root *xmlpath.Node) (result interface{}) {
	// is "f" has no children and so is simple type, like: int, string, float64, etc.
	if f.Type.kind != reflect.Struct {
		// check every XPath provided
		for _, query := range f.xquery {
			if f.Type.isArray {
				res := make([]interface{}, 0)
				iter := query.Iter(root)
				for iter.Next() {
					// try to find each context
					bts := iter.Node().Bytes()
					test := f.DataRules.Test(bts)
					if test {
						bts := f.DataRules.Clean(bts)
						val, err := ByteToKind(f.Type.kind, bts)
						if err != nil {
							//fmt.Println(err)
						} else {
							if f.Unique {
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
				result = interface{}(res)
			} else {
				val, ok := query.Bytes(root)
				if ok {
					var err error
					result, err = ByteToKind(f.Type.kind, val)
					if err != nil {
						//fmt.Println(err)
					}
				}

			}

			// exit if at least 1 value found for XPath
			if result != nil {
				return result
			}
		}
	} else {
		if f.Type.isArray {
			res := make([]interface{}, 0)
			// only one path available works for struct
			iter := f.xquery[0].Iter(root)
			for iter.Next() {
				// try to find each context
				context := iter.Node()
				val := make(map[string]interface{})
				for _, child_field := range f.Field {
					r := child_field.Retrieve(context)

					if r == nil && !child_field.Optional {
						//result = nil
						val = nil
						break
					}

					val[child_field.Title] = r
				}
				if val != nil {
					res = append(res, val)
				}
			}
			result = res
		} else {
			val := make(map[string]interface{})
			iter := f.xquery[0].Iter(root)
			if iter.Next() {
				for _, child_field := range f.Field {
					r := child_field.Retrieve(iter.Node())

					if r == nil && !child_field.Optional {
						//result = nil
						val = nil
						break
					}

					val[child_field.Title] = r
				}
				if val != nil {
					result = val
				}
			}
		}
	}

	return result
}

func (p *HtmlMap) ApplyHtml(url string, context *xmlpath.Node) interface{} {
	if !p.URLRules.Test([]byte(url)) {
		return nil
	}

	// retrieve data for root field
	data := p.Field.Retrieve(context)
	if data != nil {
		n := make(map[string]interface{})
		n[p.Field.Title] = data
		return n
	}

	return nil
}

func (p *Patterns) Apply(url string, content io.Reader) (map[string]interface{}, error) {
	data, err := xmlpath.ParseHTML(content)
	if err != nil {
		return nil, err
	}
	return p.HtmlPatternTree.ApplyHtmlPatterns(url, data), nil
}

// Applies XML patterns to input (URL "address" and HTML "content").
// Returns map with result data.
func (pn *PatternNode) ApplyHtmlPatterns(url string, data *xmlpath.Node /*content io.Reader*/) map[string]interface{} {
	var el map[string]interface{}
	for key, val := range *pn {
		if pattern, ok := val.(*HtmlMap); ok {
			if res := pattern.ApplyHtml(url, data); res != nil {
				if el == nil {
					el = make(map[string]interface{})
				}
				el[key] = res
			}
		} else if subPattern, ok := val.(*PatternNode); ok {
			if res := subPattern.ApplyHtmlPatterns(url, data); res != nil {
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
