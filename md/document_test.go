package md

import (
	"testing"
	"testing/fstest"

	"github.com/matryer/is"
)

var fsys = fstest.MapFS{
	"emptydoc.md": &fstest.MapFile{},
	"docwithincludes.md": &fstest.MapFile{
		Data: []byte(`
			# A regular markdown document

			#include "othermarkdowndoc.md"

			## Some sub headings
			> a nice inline quote

			#include "yetanotherothermarkdowndoc.md"
		`),
	},
}

func TestOpenEmptyRootMDDoc(t *testing.T) {
	is := is.New(t)
	_, err := Open("emptydoc.md", fsys)
	is.NoErr(err)
}

func TestOpenNonExistantDoc(t *testing.T) {
	is := is.New(t)
	_, err := Open("doesnotexist.md", fsys)
	is.True(err != nil)
	is.Equal(err.Error(), "open doesnotexist.md: file does not exist")
}

func TestResolveIncludesOfEmptyDoc(t *testing.T) {
	is := is.New(t)

	doc, err := Open("emptydoc.md", fsys)
	is.NoErr(err)
	err = doc.ResolveIncludes("")
	is.True(err != nil)
	is.Equal(err.Error(), "document emptydoc.md contains no includes")
}

func TestIncludesAreFoundInDocumentWithIncludes(t *testing.T) {
	is := is.New(t)

	doc, err := Open("docwithincludes.md", fsys)
	is.NoErr(err)
	is.Equal(len(doc.includes), 2)
	is.Equal(doc.includes[0], include{
		path:    "othermarkdowndoc.md",
		parent:  "docwithincludes.md",
		linePos: 4,
	})
	is.Equal(doc.includes[1], include{
		path:    "yetanotherothermarkdowndoc.md",
		parent:  "docwithincludes.md",
		linePos: 9,
	})
}
