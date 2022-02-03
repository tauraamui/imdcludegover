package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/tacusci/logging/v2"
	log "github.com/tauraamui/imdclude/logging"
	"github.com/tauraamui/imdclude/md"
)

type opts struct {
	Doc       string `short:"t" long:"doc" description:"Target document to import includes into." required:"true"`
	LookupDir string `short:"d" long:"dir" description:"Path to dir containing markdown files to search." default:"."`
	Debug     bool   `short:"v" long:"verbose" description:"Displays all internal/debug logs to assist with user level debugging."`
}

func main() {
	opts := opts{}
	if _, err := flags.Parse(&opts); err != nil {
		logging.Fatal(err.Error())
	}

	log.OUTPUT = opts.Debug

	doc, err := md.Open(opts.Doc)
	if err != nil {
		logging.Fatal(err.Error())
	}
	defer doc.Close()

	if err := doc.ResolveIncludes(opts.LookupDir); err != nil {
		logging.Fatal(err.Error())
	}

	dfd, err := os.Create(opts.Doc)
	if err != nil {
		logging.Fatal(err.Error())
	}

	doc.Write(dfd)
	dfd.Close()
}
