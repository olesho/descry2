// patterns
package parser

import (
	"bytes"
	"errors"
	//"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/antchfx/xpath"
	"github.com/antchfx/xquery/html"
	"golang.org/x/net/html"
)

// Field struct declaration: <Field>...</Field>
type Field struct {
	// field title
	Title string `xml:"title,attr"`

	// field type: int, string, float64, struct, html
	Type string `xml:"type,attr"`

	// xpath expression to find field
	Path string

	// data filter or transformaion based on regex expressions; see rules.go
	Data *RegexRules

	// data filter based on xpath expressions
	XData *XpathRules

	// optional field
	Optional bool `xml:"optional,attr,omitempty"`

	// deprecated
	DontStore bool `xml:"dontstore,attr,omitempty"`

	//
	Multiple bool `xml:"multiple,attr,omitempty"`

	// only unique fields
	Unique bool `xml:"unique,attr,omitempty"`

	// preserve HTML attributes (only works if field type="html")
	Attr bool `xml:"attr,attr,omitempty"`

	// sub-fields declaration
	Field []*Field
}

type CompiledField struct {
	title    string
	dataType *Type
	path     []*xpath.Expr //[]string //[]*xmlpath.Path
	xdata    *CompiledXpathRules
	data     *CompiledRegexRules
	parent   *CompiledField

	optional  bool `xml:"optional,attr,omitempty"`
	dontStore bool `xml:"dontstore,attr,omitempty"`
	multiple  bool `xml:"multiple,attr,omitempty"`
	unique    bool `xml:"unique,attr,omitempty"`
	attr      bool `xml:"attr,attr,omitempty"`

	field []*CompiledField
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

	c.xdata, err = f.XData.Compile()
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
	c.attr = f.Attr

	return c, nil
}

// get relative path like: "Root.SubField1.Subfield2..."
func (f *CompiledField) RelativePath() string {
	if f.parent == nil {
		return f.title
	} else {
		return f.parent.RelativePath() + "." + f.title
	}
}

func (f *CompiledField) Retrieve(root *html.Node) (result interface{}) {
	// is "f" has no children and so is simple type, like: int, string, float64, etc.
	if f.dataType.kind != reflect.Struct {
		// check every Path provided
		for _, query := range f.path {
			if f.multiple {
				if f.dataType.isHtml {
					res := make([]interface{}, 0)
					var err error
					htmlquery.FindEach(root, query.String(), func(n int, next *html.Node) {
						// test include/exclude
						if f.xdata.Test(htmlquery.CreateXPathNavigator(next)) {
							// clean off unnecessary nodes
							next = f.xdata.Clean(next)

							var buf bytes.Buffer
							w := io.Writer(&buf)

							if f.attr {
								err = html.Render(w, next)
							} else {
								err = Render(w, next)
							}
							if err != nil {
								// cant render
								//fmt.Println(err)
							}

							val, err := ByteToKind(f.dataType.kind, buf.Bytes())
							if err != nil {
								// cant convert
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
					})
					result = interface{}(res)
				} else {
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
				}

			} else {
				var err error
				if f.dataType.isHtml {
					singleNode := htmlquery.FindOne(root, query.String())

					// test include/exclude
					if f.xdata.Test(htmlquery.CreateXPathNavigator(singleNode)) {
						// clean off unnecessary nodes
						singleNode = f.xdata.Clean(singleNode)

						var buf bytes.Buffer
						w := io.Writer(&buf)

						if f.attr {
							err = html.Render(w, singleNode)
						} else {
							err = Render(w, singleNode)
						}
						if err != nil {
							// cant render
							//fmt.Println(err)
						}

						result, err = ByteToKind(f.dataType.kind, buf.Bytes())
						if err != nil {
							// cant convert
							//fmt.Println(err)
						}
					}
				} else {
					// create iterator from "query" xpath within "root"
					iter := query.Evaluate(htmlquery.CreateXPathNavigator(root)).(*xpath.NodeIterator)
					if iter.MoveNext() {
						val := []byte(iter.Current().Value())
						test := f.data.Test(val)
						if test {
							cut := f.data.Clean(val)
							found := f.data.FindOne(cut)
							result, err = ByteToKind(f.dataType.kind, found)
							if err != nil {
								// cant convert
								//fmt.Println(err)
							}
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
		if f.multiple {
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
