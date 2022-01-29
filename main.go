package main

import (
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/tacusci/logging/v2"
	"github.com/tauraamui/imdclude/md"
)

type opts struct {
	Doc       string `short:"t" long:"doc" description:"Target document to import includes into." required:"true"`
	LookupDir string `short:"d" long:"dir" description:"Path to dir containing markdown files to search." default:"."`
}

func main() {
	opts := opts{}
	if _, err := flags.Parse(&opts); err != nil {
		logging.Fatal(err.Error())
	}
	fmt.Println(opts.Doc)

	doc, err := md.Open(opts.Doc)
	if err != nil {
		logging.Fatal(err.Error())
	}

	if err := doc.ResolveIncludes(opts.LookupDir); err != nil {
		logging.Fatal(err.Error())
	}
}
