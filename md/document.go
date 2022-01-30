package md

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
)

type include struct {
	path    string
	parent  string
	linePos int
}

type Document struct {
	name        string
	r           io.Reader
	lineContent []string
	includes    []include
}

func New(name string, r io.Reader) *Document {
	return &Document{name: name, r: r}
}

func newFromFile(fd fs.File) (*Document, error) {
	d := Document{r: fd}

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
		return fmt.Errorf("document %s contains no includes", d.name)
	}
	return nil
}

type errGroup []error

func (e errGroup) toErrOrNil() error {
	if len(e) > 0 {
		fe := errors.New("")
		for _, err := range e {
			fe = fmt.Errorf("%v: %v", fe, err)
		}
		return fe
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
				d.name,
				pos,
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
		return nil, err
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
