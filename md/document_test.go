package md

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/matryer/is"
)

var fsys = fstest.MapFS{
	"emptydoc.md": &fstest.MapFile{},
	"singlelinedocwithinclude.md": &fstest.MapFile{
		Data: []byte(`#include "mddocsdir/multilineothermarkdowndoc.md"`),
	},
	"docwithincludes.md": &fstest.MapFile{
		Data: []byte(`
			# A regular markdown document

			#include "mddocsdir/othermarkdowndoc.md"

			## Some sub headings
			> a nice inline quote

			#include "mddocsdir/yetanotherothermarkdowndoc.md"
			#include "mddocsdir/multilineothermarkdowndoc.md"

			### Another sub header
		`),
	},
	"mddocsdir/othermarkdowndoc.md": &fstest.MapFile{
		Data: []byte(`                     # A child markdown document called other`),
	},
	"mddocsdir/yetanotherothermarkdowndoc.md": &fstest.MapFile{
		Data: []byte(`                     # A child markdown document called yet another`),
	},
	"mddocsdir/multilineothermarkdowndoc.md": &fstest.MapFile{
		Data: []byte(
			`
			# First line of multiline markdown.
			# Second line of multiline markdown.
			#include "mddocsdir/yetanotherothermarkdowndoc.md"
			`,
		),
	},
}

func TestOpenEmptyRootMDDoc(t *testing.T) {
	is := is.New(t)
	_, err := Open("emptydoc.md", fsys)
	is.NoErr(err)
}

func TestClosingOpenEmptyRootMDDoc(t *testing.T) {
	is := is.New(t)
	doc, err := Open("emptydoc.md", fsys)
	is.NoErr(err)
	is.NoErr(doc.Close())
}

func TestOpenNonExistantDoc(t *testing.T) {
	is := is.New(t)
	_, err := Open("doesnotexist.md", fsys)
	is.True(err != nil)
	is.Equal(err.Error(), "open doesnotexist.md: file does not exist: path: doesnotexist.md")
}

func TestIncludesAreFoundInDocumentWithIncludes(t *testing.T) {
	is := is.New(t)

	doc, err := Open("docwithincludes.md", fsys)
	is.NoErr(err)
	is.Equal(len(doc.includes), 3)
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
	is.NoErr(doc.Close())
}

func TestIncludesAreResolved(t *testing.T) {
	is := is.New(t)

	doc, err := Open("docwithincludes.md", fsys)
	is.NoErr(err)

	is.NoErr(doc.ResolveIncludes("mddocsdir", fsys))
}

func TestSingleLineMDWithIncludeIsResolved(t *testing.T) {
	is := is.New(t)

	doc, err := Open("singlelinedocwithinclude.md", fsys)
	is.NoErr(err)

	is.NoErr(doc.ResolveIncludes("mddocsdir", fsys))
	for i, l := range doc.lineContent {
		fmt.Printf("%d: %s\n", i, l)
	}
}
