package md

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
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

func (d *Document) parse() {
	readLineByLine(d.r, func(l string, pos int, e error) {
		if path, ok := isInclude(l); ok {
			d.includes = append(d.includes, include{
				path,
				"",
				pos,
			})
		}
		d.lineContent = append(d.lineContent, l)
	})
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
	doc.parse()

	return doc, nil
}

func isInclude(l string) (string, bool) {
	return "", false
}

func readLineByLine(data io.Reader, eachLine func(string, int, error)) {
	rr := bufio.NewReader(data)
	count := 0
	for {
		rr.ReadLine()
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
