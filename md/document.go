package md

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	paths "path"
	"regexp"
	"strings"
)

type include struct {
	path    string
	name    string
	parent  string
	linePos int
	doc     *Document
}

type Document struct {
	name        string
	r           io.Reader
	lineContent []string
	includes    []include
}

func New(name string, r io.Reader) *Document {
	return &Document{name: name, r: r, includes: []include{}}
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
	if d.includes == nil || len(d.includes) == 0 {
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

		inclPos := incl.linePos
		posOffset += func() int {
			conl := len(incl.doc.lineContent)
			if conl > 1 {
				conl -= 1
			}
			return conl
		}()

		var content []string
		content = append(content, d.lineContent[:inclPos-1]...)
		content = append(content, incl.doc.lineContent...)
		content = append(content, d.lineContent[inclPos:]...)

		d.lineContent = content
	}

	return nil
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
	readLineByLine(d.r, func(l string, pos int, e error) {
		if e != nil {
			errs = append(errs, e)
		}
		if path, ok := isInclude(l); ok {
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

func Open(name string, fsyses ...fs.FS) (*Document, error) {
	fsys := resolveFS(".", fsyses)
	fd, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%v: path: %s", err, name)
	}

	doc, err := newFromFile(fd)
	if err != nil {
		return nil, err
	}

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

func readLineByLine(data io.Reader, eachLine func(string, int, error)) {
	rr := bufio.NewReader(data)
	count := 0
	for {
		count++
		line, isPrefix, err := rr.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			}
			eachLine("", count, err)
		}

		if isPrefix {
			eachLine("", count, fmt.Errorf("line %d is too long: %v", count, err))
		}

		eachLine(string(line), count, nil)
	}
}

func resolveFS(defaultRoot string, fsyses []fs.FS) fs.FS {
	if len(fsyses) > 0 {
		return fsyses[0]
	}
	return os.DirFS(defaultRoot)
}
