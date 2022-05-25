package md

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

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
			#include "childocwithinsamedirectoryasroot.md"
		`),
	},
	"childocwithinsamedirectoryasroot.md": &fstest.MapFile{
		Data: []byte(`                     # A child document within same directory as root`),
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
	is.Equal(len(doc.includes), 4)
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
	is.Equal(doc.includes[2], include{
		path:    "mddocsdir/multilineothermarkdowndoc.md",
		name:    "multilineothermarkdowndoc.md",
		parent:  "docwithincludes.md",
		linePos: 10,
	})
	is.Equal(doc.includes[3], include{
		path:    "childocwithinsamedirectoryasroot.md",
		name:    "childocwithinsamedirectoryasroot.md",
		parent:  "docwithincludes.md",
		linePos: 13,
	})
	is.NoErr(doc.Close())
}

func TestIncludesAreResolved(t *testing.T) {
	is := is.New(t)

	doc, err := Open("docwithincludes.md", fsys)
	is.NoErr(err)

	is.NoErr(doc.ResolveIncludes("mddocsdir", fsys))
}

func TestWritingBackupDocumentHeader(t *testing.T) {
	is := is.New(t)

	now := uint32(time.Now().Unix())

	buff := bytes.Buffer{}

	bkupPath := filepath.Join(tmpDir(), "existing-path.bkup")
	header := backupFileHeader{
		magicPrefix:     4839,
		backupTimestamp: now,
		originalPath:    bkupPath,
	}

	header.write(&buff)

	newHeader := backupFileHeader{}
	newHeader.read(&buff)

	is.Equal(newHeader.magicPrefix, uint16(4839))
	is.Equal(newHeader.backupTimestamp, now)
	is.Equal(newHeader.pathLength, uint32(len(newHeader.originalPath)))
	is.Equal(newHeader.originalPath, bkupPath)
}

func TestBackupRoutine(t *testing.T) {
	is := is.New(t)

	doc := Document{
		name: "testdoc",
	}
	doc.lineContent = []byte(`
			# A regular markdown document

			#include "mddocsdir/othermarkdowndoc.md"

			## Some sub headings
			> a nice inline quote

			#include "mddocsdir/yetanotherothermarkdowndoc.md"
			#include "mddocsdir/multilineothermarkdowndoc.md"

			### Another sub header
			#include "childocwithinsamedirectoryasroot.md"
		`)

	id, filePath, err := Backup(&doc)
	is.NoErr(err)
	is.True(len(filePath) > 0)

	fs, err := os.Stat(filePath)
	is.NoErr(err)
	is.True(fs != nil)

	bkups, err := Backups()
	is.NoErr(err)
	for _, bkup := range bkups {
		if bkup.ID == id {
			is.Equal(doc.lineContent, bkup.Content)
		}
	}
}
