package md

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/user"
	paths "path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	log "github.com/tauraamui/imdclude/logging"
	"github.com/teris-io/shortid"

	"github.com/tacusci/logging/v2"
)

type include struct {
	path    string
	name    string
	parent  string
	linePos int
	doc     *Document
}

type Document struct {
	path        string
	name        string
	r           io.ReadCloser
	lineContent [][]byte
	includes    []include
}

func newFromFile(fd fs.File) (*Document, error) {
	d := Document{r: fd, includes: []include{}}

	fi, err := fd.Stat()
	if err != nil {
		d.name = "N/A"
		return nil, err
	}

	d.name = fi.Name()
	return &d, nil
}

func (d *Document) ResolveIncludes(path string, fsyses ...fs.FS) error {
	if len(d.includes) == 0 {
		log.Printfln("[%s] no includes found", d.name)
		return nil
	}

	if err := d.openAllIncludes(resolveFS(path, fsyses)); err != nil {
		return err
	}

	if err := d.resolveIncludesIncludes(path, fsyses...); err != nil {
		return err
	}

	posOffset := 0
	for _, incl := range d.includes {
		if incl.doc == nil {
			continue
		}

		// inclPos := incl.linePos
		posOffset += func() int {
			conl := len(incl.doc.lineContent)
			if conl > 1 {
				conl -= 1
			}
			return conl
		}()

		log.Printfln("[%s] replacing line %d with %s's content", d.name, incl.linePos, incl.name)
		// var content []byte
		// content = append(content, d.lineContent[:inclPos-1]...)
		// content = append(content, incl.doc.lineContent...)
		// content = append(content, d.lineContent[inclPos:]...)

		// d.lineContent = content
	}

	return nil
}

func (d *Document) Write(w io.StringWriter) (int, error) {
	d.r.Close() // close original reader
	var c int
	for _, l := range d.lineContent {
		cc, err := w.WriteString(string(l))
		c += cc
		if err != nil {
			return c, err
		}
	}
	return c, nil
}

func (d *Document) Close() error {
	if d == nil {
		return nil
	}

	err := func() error {
		if r := d.r; r != nil {
			return r.Close()
		}
		return nil
	}()

	if d.includes == nil {
		return err
	}

	for _, incl := range d.includes {
		incl.doc.Close()
	}
	return err
}

func (d *Document) resolveIncludesIncludes(path string, fsyses ...fs.FS) error {
	errs := errGroup{}
	for _, incl := range d.includes {
		if incl.doc == nil {
			continue
		}
		if err := incl.doc.ResolveIncludes(path, fsyses...); err != nil {
			errs = append(errs, err)
		}
	}
	return errs.toErrOrNil()
}

func (d *Document) openAllIncludes(fsys fs.FS) error {
	errs := errGroup{}
	for i := 0; i < len(d.includes); i++ {
		ii := d.includes[i]
		log.Printfln("[%s] opening include: %s", d.name, ii.path)
		incl, err := Open(ii.path, fsys)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := incl.parse(); err != nil {
			errs = append(errs, err)
			continue
		}

		d.includes[i].doc = incl
	}

	return errs.toErrOrNil()
}

type errGroup []error

func (e errGroup) toErrOrNil() error {
	if len(e) > 0 {
		buf := strings.Builder{}
		buf.WriteString(fmt.Sprintf("%d errors occurred:\n", len(e)))
		for _, err := range e {
			buf.WriteString(fmt.Sprintf("\t* %v\n", err))
		}
		return errors.New(buf.String())
	}
	return nil
}

func (d *Document) parse() error {
	errs := errGroup{}
	readLineByLine(d.r, func(l []byte, pos int, e error) {
		if e != nil {
			errs = append(errs, e)
		}
		if path, ok := isInclude(string(l)); ok {
			d.includes = append(d.includes, include{
				path,
				paths.Base(path),
				d.name,
				pos,
				nil,
			})
		}
		d.lineContent = append(d.lineContent, l)
	})
	return errs.toErrOrNil()
}

const backupFileHeaderMagic uint16 = 0x3532

var tmpDir = func() string {
	usr, err := user.Current()
	if err != nil {
		return filepath.Join(os.TempDir(), "imdclude")
	}

	return filepath.Join(usr.HomeDir, "tmp", "imdclude")
}

func Backup(doc *Document) (id string, path string, err error) {
	tmpDir := tmpDir()
	err = os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		if !strings.Contains(err.Error(), "file exists") {
			return "", "", err
		}
	}

	tempFile, err := ioutil.TempFile(tmpDir, fmt.Sprintf("%s.*.bkup", doc.name))
	if err != nil {
		return "", "", err
	}
	defer tempFile.Close()

	header := backupFileHeader{
		magicPrefix:     backupFileHeaderMagic,
		backupTimestamp: uint32(time.Now().Unix()),
		originalPath:    doc.path,
	}

	header.write(tempFile)
	tempFile.Write(mergeLines(doc.lineContent))

	return header.id, tempFile.Name(), nil
}

func Restore(doc BackedUpDoc) error {
	if err := os.MkdirAll(paths.Dir(doc.Path), os.ModePerm); err != nil {
		return err
	}

	rf, err := os.Open(doc.Path)
	if err != nil {
		return err
	}
	rf.Truncate(0)

	if _, err := rf.Write(mergeLines(doc.Content)); err != nil {
		return err
	}

	return nil
}

type BackedUpDoc struct {
	ID      string
	Path    string
	Name    string
	Time    int
	Content [][]byte
}

func Backups(fsys ...fs.FS) ([]BackedUpDoc, error) {
	tmpDir := tmpDir()
	ts, err := os.Stat(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("unable to stat %s: %w", tmpDir, err)
	}

	if ts == nil || !ts.IsDir() {
		return nil, fmt.Errorf("backup location %s does not exist", tmpDir)
	}

	dfs := os.DirFS(tmpDir)
	if len(fsys) == 1 {
		dfs = fsys[0]
	}

	backupFile, err := fs.ReadDir(dfs, ".")
	if err != nil {
		return nil, fmt.Errorf("unable to read backups directory: %s: %w", tmpDir, err)
	}

	foundDocs := []BackedUpDoc{}
	for _, bfptr := range backupFile {
		bfile, err := dfs.Open(bfptr.Name())
		if err != nil {
			logging.Fatal(err.Error())
		}

		header := backupFileHeader{}
		header.read(bfile)

		info, err := bfptr.Info()
		if err != nil {
			logging.Fatal(err.Error())
		}
		content := make([]byte, info.Size()-int64(header.size()))
		if header.magicPrefix == backupFileHeaderMagic {
			bfile.Read(content)
			foundDocs = append(foundDocs, BackedUpDoc{
				ID:      header.id,
				Path:    header.originalPath,
				Name:    header.originalPath,
				Time:    int(header.backupTimestamp),
				Content: splitLines(content),
			})
		}
	}

	return foundDocs, nil
}

// header is 10 bytes wide
type backupFileHeader struct {
	magicPrefix     uint16
	backupTimestamp uint32
	idLength        uint32
	id              string
	pathLength      uint32
	originalPath    string
}

func (h *backupFileHeader) write(w io.Writer) {
	w.Write(uint16ToBytes(h.magicPrefix))
	w.Write(uint32ToBytes(h.backupTimestamp))
	genid := shortid.MustGenerate()
	h.id = genid
	w.Write(uint32ToBytes(uint32(len(genid))))
	w.Write([]byte(genid))
	w.Write(uint32ToBytes(uint32(len(h.originalPath))))
	w.Write([]byte(h.originalPath))
}

func (h *backupFileHeader) size() int {
	return 14 + len(h.id) + len(h.originalPath)
}

func (h *backupFileHeader) read(r io.Reader) {
	readingBuffer := make([]byte, 4)
	r.Read(readingBuffer[:2])
	h.magicPrefix = bytesToUint16(readingBuffer[:2])

	r.Read(readingBuffer)
	h.backupTimestamp = bytesToUint32(readingBuffer)

	r.Read(readingBuffer)
	h.idLength = bytesToUint32(readingBuffer)
	idBytes := make([]byte, h.idLength)
	r.Read(idBytes)
	h.id = string(idBytes)

	r.Read(readingBuffer)
	h.pathLength = bytesToUint32(readingBuffer)

	pathBytes := make([]byte, h.pathLength)
	r.Read(pathBytes)
	h.originalPath = string(pathBytes)
}

func bytesToUint16(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

func bytesToUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

func uint16ToBytes(n uint16) []byte {
	d := make([]byte, 2)
	binary.LittleEndian.PutUint16(d, n)
	return d
}

func uint32ToBytes(n uint32) []byte {
	d := make([]byte, 4)
	binary.LittleEndian.PutUint32(d, n)
	return d
}

func Open(name string, fsyses ...fs.FS) (*Document, error) {
	wd, err := os.Getwd()
	if err != nil {
		logging.Error("unable to search for files relative to CWD: %v", err)
		wd = "."
	}
	fsys := resolveFS(wd, fsyses)
	fd, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%v: path: %s", err, name)
	}

	doc, err := newFromFile(fd)
	if err != nil {
		return nil, err
	}

	doc.path = filepath.Join(wd, name)

	if err := doc.parse(); err != nil {
		return nil, err
	}

	return doc, nil
}

const includeTokenDef = `\#include \"(\S+)\"`

var includeRegexInst *regexp.Regexp

func init() {
	includeRegexInst = regexp.MustCompile(includeTokenDef)
}

func isInclude(l string) (string, bool) {
	matches := includeRegexInst.FindAllStringSubmatch(l, len(l)+1)
	if len(matches) == 0 {
		return "", false
	}
	for _, m := range matches {
		ms := len(m)
		if ms > 1 {
			return m[1], true
		}
	}
	return "", false
}

func readLineByLine(data io.Reader, eachLine func([]byte, int, error)) {
	rr := bufio.NewReader(data)

	count := 0
	lineBuffer := []byte{}
	for {
		count++
		line, isPrefix, err := rr.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			}
			eachLine(nil, count, err)
		}

		if isPrefix {
			lineBuffer = append(lineBuffer, line...)
			continue
		}

		if lineBuffer != nil {
			eachLine(append(lineBuffer, line...), count, nil)
			continue
		}

		eachLine(line, count, nil)
	}
}

func mergeLines(b [][]byte) []byte {
	dest := []byte{}
	s := len(b)
	for i, l := range b {
		if i+1 < s {
			l = append(l, []byte("\n")...)
		}
		dest = append(dest, l...)
	}
	return dest
}

func splitLines(b []byte) [][]byte {
	dest := [][]byte{}
	rr := bufio.NewReaderSize(bytes.NewBuffer(b), len(b))

	readLineByLine(rr, func(b []byte, i int, err error) {
		if err != nil {
			return
		}
		dest = append(dest, b)
	})

	return dest
}

func resolveFS(defaultRoot string, fsyses []fs.FS) fs.FS {
	if len(fsyses) > 0 {
		return fsyses[0]
	}
	return os.DirFS(defaultRoot)
}
