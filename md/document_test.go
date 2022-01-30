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

			#include "mddocsdir/othermarkdowndoc.md"

			## Some sub headings
			> a nice inline quote

			#include "mddocsdir/yetanotherothermarkdowndoc.md"
		`),
	},
	"mddocsdir/othermarkdowndoc.md": &fstest.MapFile{
		Data: []byte(`
			# A child markdown document called other
		`),
	},
	"mddocsdir/yetanotherothermarkdowndoc.md": &fstest.MapFile{
		Data: []byte(`
			# A child markdown document called yet another
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
	is.Equal(err.Error(), "open doesnotexist.md: file does not exist: path: doesnotexist.md")
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
		path:    "mddocsdir/othermarkdowndoc.md",
		name:    "othermarkdowndoc.md",
		parent:  "docwithincludes.md",
		linePos: 4,
	})
	is.Equal(doc.includes[1], include{
		path:    "mddocsdir/yetanotherothermarkdowndoc.md",
		name:    "yetanotherothermarkdowndoc.md",
		parent:  "docwithincludes.md",
		linePos: 9,
	})
}

func TestIncludesAreResolved(t *testing.T) {
	is := is.New(t)

	doc, err := Open("docwithincludes.md", fsys)
	is.NoErr(err)

	is.NoErr(doc.ResolveIncludes("mddocsdir", fsys))
}
