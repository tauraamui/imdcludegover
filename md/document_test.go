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
	is.Equal(len(doc.includes), 1)
}
