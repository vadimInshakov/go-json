package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

type opType struct {
	Op   string
	Code string
}

func createOpType(op, code string) opType {
	return opType{
		Op:   op,
		Code: code,
	}
}

func _main() error {
	tmpl, err := template.New("").Parse(`// Code generated by internal/cmd/generator. DO NOT EDIT!
package encoder

import (
  "strings"
)

type CodeType int

const (
{{- range $index, $type := .CodeTypes }}
  Code{{ $type }} CodeType = {{ $index }}
{{- end }}
)

var opTypeStrings = [{{ .OpLen }}]string{
{{- range $type := .OpTypes }}
    "{{ $type.Op }}",
{{- end }}
}

type OpType uint16

const (
{{- range $index, $type := .OpTypes }}
  Op{{ $type.Op }} OpType = {{ $index }}
{{- end }}
)

func (t OpType) String() string {
  if int(t) >= {{ .OpLen }} {
    return ""
  }
  return opTypeStrings[int(t)]
}

func (t OpType) CodeType() CodeType {
  if strings.Contains(t.String(), "Struct") {
    if strings.Contains(t.String(), "End") {
      return CodeStructEnd
    }
    return CodeStructField
  }
  switch t {
  case OpArray, OpArrayPtr:
    return CodeArrayHead
  case OpArrayElem:
    return CodeArrayElem
  case OpSlice, OpSlicePtr:
    return CodeSliceHead
  case OpSliceElem:
    return CodeSliceElem
  case OpMap, OpMapPtr:
    return CodeMapHead
  case OpMapKey:
    return CodeMapKey
  case OpMapValue:
    return CodeMapValue
  case OpMapEnd:
    return CodeMapEnd
  }

  return CodeOp
}

func (t OpType) HeadToPtrHead() OpType {
  if strings.Index(t.String(), "PtrHead") > 0 {
    return t
  }

  idx := strings.Index(t.String(), "Head")
  if idx == -1 {
    return t
  }
  suffix := "PtrHead"+t.String()[idx+len("Head"):]

  const toPtrOffset = 2
  if strings.Contains(OpType(int(t) + toPtrOffset).String(), suffix) {
    return OpType(int(t) + toPtrOffset)
  }
  return t
}

func (t OpType) HeadToOmitEmptyHead() OpType {
  const toOmitEmptyOffset = 1
  if strings.Contains(OpType(int(t) + toOmitEmptyOffset).String(), "OmitEmpty") {
    return OpType(int(t) + toOmitEmptyOffset)
  }

  return t
}

func (t OpType) PtrHeadToHead() OpType {
  idx := strings.Index(t.String(), "Ptr")
  if idx == -1 {
    return t
  }
  suffix := t.String()[idx+len("Ptr"):]

  const toPtrOffset = 2
  if strings.Contains(OpType(int(t) - toPtrOffset).String(), suffix) {
    return OpType(int(t) - toPtrOffset)
  }
  return t
}

func (t OpType) FieldToEnd() OpType {
  idx := strings.Index(t.String(), "Field")
  if idx == -1 {
    return t
  }
  suffix := t.String()[idx+len("Field"):]
  if suffix == "" || suffix == "OmitEmpty" {
    return t
  }
  const toEndOffset = 2
  if strings.Contains(OpType(int(t) + toEndOffset).String(), "End"+suffix) {
    return OpType(int(t) + toEndOffset)
  }
  return t
}

func (t OpType) FieldToOmitEmptyField() OpType {
  const toOmitEmptyOffset = 1
  if strings.Contains(OpType(int(t) + toOmitEmptyOffset).String(), "OmitEmpty") {
    return OpType(int(t) + toOmitEmptyOffset)
  }
  return t
}
`)
	if err != nil {
		return err
	}
	codeTypes := []string{
		"Op",
		"ArrayHead",
		"ArrayElem",
		"SliceHead",
		"SliceElem",
		"MapHead",
		"MapKey",
		"MapValue",
		"MapEnd",
		"Recursive",
		"StructField",
		"StructEnd",
	}
	primitiveTypes := []string{
		"int", "uint", "float32", "float64", "bool", "string", "bytes", "number",
		"array", "map", "slice", "struct", "MarshalJSON", "MarshalText",
		"intString", "uintString", "float32String", "float64String", "boolString", "stringString", "numberString",
		"intPtr", "uintPtr", "float32Ptr", "float64Ptr", "boolPtr", "stringPtr", "bytesPtr", "numberPtr",
		"arrayPtr", "mapPtr", "slicePtr", "marshalJSONPtr", "marshalTextPtr", "interfacePtr",
		"intPtrString", "uintPtrString", "float32PtrString", "float64PtrString", "boolPtrString", "stringPtrString", "numberPtrString",
	}
	primitiveTypesUpper := []string{}
	for _, typ := range primitiveTypes {
		primitiveTypesUpper = append(primitiveTypesUpper, strings.ToUpper(string(typ[0]))+typ[1:])
	}
	opTypes := []opType{
		createOpType("End", "Op"),
		createOpType("Interface", "Op"),
		createOpType("Ptr", "Op"),
		createOpType("SliceElem", "SliceElem"),
		createOpType("SliceEnd", "Op"),
		createOpType("ArrayElem", "ArrayElem"),
		createOpType("ArrayEnd", "Op"),
		createOpType("MapKey", "MapKey"),
		createOpType("MapValue", "MapValue"),
		createOpType("MapEnd", "Op"),
		createOpType("Recursive", "Op"),
		createOpType("RecursivePtr", "Op"),
		createOpType("RecursiveEnd", "Op"),
		createOpType("StructAnonymousEnd", "StructEnd"),
	}
	for _, typ := range primitiveTypesUpper {
		typ := typ
		opTypes = append(opTypes, createOpType(typ, "Op"))
	}
	for _, typ := range append(primitiveTypesUpper, "") {
		for _, ptrOrNot := range []string{"", "Ptr"} {
			for _, opt := range []string{"", "OmitEmpty"} {
				ptrOrNot := ptrOrNot
				opt := opt
				typ := typ

				op := fmt.Sprintf(
					"Struct%sHead%s%s",
					ptrOrNot,
					opt,
					typ,
				)
				opTypes = append(opTypes, opType{
					Op:   op,
					Code: "StructField",
				})
			}
		}
	}
	for _, typ := range append(primitiveTypesUpper, "") {
		for _, opt := range []string{"", "OmitEmpty"} {
			opt := opt
			typ := typ

			op := fmt.Sprintf(
				"StructField%s%s",
				opt,
				typ,
			)
			opTypes = append(opTypes, opType{
				Op:   op,
				Code: "StructField",
			})
		}
		for _, opt := range []string{"", "OmitEmpty"} {
			opt := opt
			typ := typ

			op := fmt.Sprintf(
				"StructEnd%s%s",
				opt,
				typ,
			)
			opTypes = append(opTypes, opType{
				Op:   op,
				Code: "StructEnd",
			})
		}
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, struct {
		CodeTypes []string
		OpTypes   []opType
		OpLen     int
	}{
		CodeTypes: codeTypes,
		OpTypes:   opTypes,
		OpLen:     len(opTypes),
	}); err != nil {
		return err
	}
	path := filepath.Join(repoRoot(), "internal", "encoder", "optype.go")
	buf, err := format.Source(b.Bytes())
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, buf, 0644)
}

func generateVM() error {
	file, err := ioutil.ReadFile("vm.go.tmpl")
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", string(file), parser.ParseComments)
	if err != nil {
		return err
	}
	for _, pkg := range []string{"vm", "vm_indent", "vm_escaped", "vm_escaped_indent"} {
		f.Name.Name = pkg
		var buf bytes.Buffer
		printer.Fprint(&buf, fset, f)
		path := filepath.Join(repoRoot(), "internal", "encoder", pkg, "vm.go")
		source, err := format.Source(buf.Bytes())
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(path, source, 0644); err != nil {
			return err
		}
	}
	return nil
}

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	relativePathFromRepoRoot := filepath.Join("internal", "cmd", "generator")
	return strings.TrimSuffix(filepath.Dir(file), relativePathFromRepoRoot)
}

//go:generate go run main.go
func main() {
	if err := generateVM(); err != nil {
		panic(err)
	}
	if err := _main(); err != nil {
		panic(err)
	}
}
