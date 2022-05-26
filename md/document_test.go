package md

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/matryer/is"
	"github.com/stretchr/testify/assert"
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

func TestSplitLines(t *testing.T) {
	is := is.New(t)

	lines := []byte(
		"First line\nSecond line\nThird line",
	)

	linesSplit := splitLines(lines)
	is.Equal(len(linesSplit), 3)

	is = is.NewRelaxed(t)
	is.Equal(string(linesSplit[0]), "First line")
	is.Equal(string(linesSplit[1]), "Second line")
	is.Equal(string(linesSplit[2]), "Third line")
}

func TestMergeLines(t *testing.T) {
	is := is.New(t)

	splitLines := [][]byte{
		[]byte("First line"),
		[]byte("Second line"),
		[]byte("Third line"),
	}

	mergedLines := mergeLines(splitLines)
	is.Equal(string(mergedLines), "First line\nSecond line\nThird line")
}

func setupTestTempDir() func() error {
	dirpath := filepath.Join(os.TempDir(), "imdclude", "tests")

	oldTempRef := tmpDir
	tmpDir = func() string {
		return dirpath
	}

	return func() error {
		tmpDir = oldTempRef
		return os.RemoveAll(dirpath)
	}
}

func writeTestMD(path string, content []byte) (func(), error) {
	testFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil, err
	}
	closeFileFunc := func() { testFile.Close() }

	_, err = testFile.Write(content)
	if err != nil {
		return closeFileFunc, err
	}

	return closeFileFunc, nil
}

func createTestMarkdownFile(is *is.I, name string, content []byte) {
	close, err := writeTestMD(name, content)
	is.NoErr(err)
	if close != nil {
		close()
	}
}

func TestResolveIncludesSuccess(t *testing.T) {
	t.Skip()
	is := is.New(t)

	resetTmpDir := setupTestTempDir()
	defer resetTmpDir()

	tmpDir := tmpDir()
	is.NoErr(os.MkdirAll(tmpDir, os.ModePerm))

	readmeFilePath := filepath.Join(tmpDir, "README.md")
	createTestMarkdownFile(is, readmeFilePath, []byte(
		`# A regular markdown document

		#include "othermarkdowndoc.md"

		## Some sub headings
		> a nice inline quote`,
	))

	createTestMarkdownFile(is, filepath.Join(tmpDir, "othermarkdowndoc.md"), []byte(
		`# Another markdown document, called other 

		## This is a different subheading
		A non-formatted normal line of text!`,
	))

	fd, err := os.Open(readmeFilePath)
	is.NoErr(err)
	defer fd.Close()

	doc, err := newFromFile(fd)
	is.NoErr(err)
	defer doc.Close()

	doc.path = readmeFilePath
	is.NoErr(doc.parse())
	is.Equal(len(doc.includes), 1)
	is.Equal(doc.includes[0].linePos, 3)

	is.NoErr(doc.ResolveIncludes(tmpDir))
	is.Equal(string(mergeLines(doc.lineContent)),
		`# A regular markdown document

		# Another markdown document, called other

		## This is a different subheading
		A non-formatted normal line of text!

		## Some sub headings
		> a nice inline quote`,
	)
}

func TestBackupRoutine(t *testing.T) {
	t.Skip()
	is := is.New(t)

	resetTmpDir := setupTestTempDir()
	defer resetTmpDir()

	doc := Document{
		name: "testdoc",
	}

	doc.lineContent = splitLines(
		[]byte(
			`# A regular markdown document

			#include "mddocsdir/othermarkdowndoc.md"

			## Some sub headings
			> a nice inline quote

			#include "mddocsdir/yetanotherothermarkdowndoc.md"
			#include "mddocsdir/multilineothermarkdowndoc.md"

			### Another sub header
			#include "childocwithinsamedirectoryasroot.md"`,
		),
	)

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
			assert.EqualValues(t, doc.lineContent, bkup.Content)
			// is.Equal(doc.lineContent, bkup.Content)
		}
	}
}
