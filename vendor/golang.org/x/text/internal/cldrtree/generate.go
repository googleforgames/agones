// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cldrtree

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/text/internal/gen"
)

func generate(b *Builder, t *Tree, w *gen.CodeWriter) {
	fmt.Fprintln(w, `import "golang.org/x/text/internal/cldrtree"`)
	fmt.Fprintln(w)

	fmt.Fprintf(w, "var tree = &cldrtree.Tree{locales, indices, buckets}\n\n")

	w.WriteComment("Path values:\n" + b.stats())
	fmt.Fprintln(w)

	// Generate enum types.
	for _, e := range b.enums {
		// Build enum types.
		w.WriteComment("%s specifies a property of a CLDR field.", e.name)
		fmt.Fprintf(w, "type %s int\n", e.name)
		fmt.Fprintln(w, "const (")
		fmt.Fprintf(w, "%v %v = iota\n", toCamel(e.keys[0]), e.name)
		for _, s := range e.keys[1:] {
			fmt.Fprintln(w, toCamel(s))
		}
		fmt.Fprintln(w, ")")
	}

	w.WriteVar("locales", t.Locales)
	w.WriteVar("indices", t.Indices)

	// Generate string buckets.
	fmt.Fprintln(w, "var buckets = []string{")
	for i := range t.Buckets {
		fmt.Fprintf(w, "bucket%d,\n", i)
	}
	fmt.Fprint(w, "}\n\n")
	w.Size += int(reflect.TypeOf("").Size()) * len(t.Buckets)

	// Generate string buckets.
	for i, bucket := range t.Buckets {
		w.WriteVar(fmt.Sprint("bucket", i), bucket)
	}
}

func toCamel(s string) string {
	p := strings.Split(s, "-")
	for i, s := range p[1:] {
		p[i+1] = strings.Title(s)
	}
	return strings.Join(p, "")
}

func (b *Builder) stats() string {
	w := &bytes.Buffer{}

	b.rootMeta.validate()
	for _, es := range b.enums {
		fmt.Fprintf(w, "<%s>\n", es.name)
		for _, s := range es.keys {
			fmt.Fprintf(w, "  - %s\n", s)
		}
	}
	fmt.Fprintln(w)
	printEnums(w, b.rootMeta.typeInfo, 0)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Nr elem:           ", len(b.strToBucket))
	fmt.Fprintln(w, "uniqued size:      ", b.size)
	fmt.Fprintln(w, "total string size: ", b.sizeAll)
	fmt.Fprintln(w, "bucket waste:      ", b.bucketWaste)

	return w.String()
}

func printEnums(w io.Writer, s *typeInfo, indent int) {
	idStr := strings.Repeat("  ", indent) + "- "
	e := s.enum
	if e == nil {
		if len(s.entries) > 0 {
			panic(fmt.Errorf("has entries but no enum values: %#v", s.entries))
		}
		return
	}
	if e.name != "" {
		fmt.Fprintf(w, "%s<%s>\n", idStr, e.name)
	} else {
		for i := 0; i < len(e.keys); i++ {
			fmt.Fprint(w, idStr)
			// fmt.Fprint(w, "- ")
			k := e.keys[i]
			if u, err := strconv.ParseUint(k, 10, 16); err == nil {
				fmt.Fprintf(w, "%s", k)
				// Skip contiguous integers
				var v, last uint64
				for i++; i < len(e.keys); i++ {
					k = e.keys[i]
					if v, err = strconv.ParseUint(k, 10, 16); err != nil {
						break
					}
					last = v
				}
				if u < last {
					fmt.Fprintf(w, `..%d`, last)
				}
				fmt.Fprintln(w)
				if err != nil {
					fmt.Fprintf(w, "%s%s\n", idStr, k)
				}
			} else if k == "" {
				fmt.Fprintln(w, `""`)
			} else {
				fmt.Fprintf(w, "%s\n", k)
			}
			if !s.sharedKeys() {
				if e := s.entries[enumIndex(i)]; e != nil {
					printEnums(w, e, indent+1)
				}
			}
		}
	}
	if s.sharedKeys() {
		for _, v := range s.entries {
			printEnums(w, v, indent+1)
			break
		}
	}
}
