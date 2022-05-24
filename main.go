package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/tacusci/logging/v2"
	log "github.com/tauraamui/imdclude/logging"
	"github.com/tauraamui/imdclude/md"
)

type opts struct {
	Doc       string `short:"f" long:"file" description:"Target file to import includes into."`
	LookupDir string `short:"d" long:"dir" description:"Path to dir containing markdown files to search." default:"."`
	Backup    bool   `short:"b" long:"backup" description:"Backup the original target document beforehand."`
	List      bool   `short:"l" long:"list" description:"List all available backups."`
	Restore   int    `short:"r" long:"restore" description:"Restore to a specified backup of given ID."`
	Debug     bool   `short:"v" long:"verbose" description:"Displays all internal/debug logs to assist with user level debugging."`
}

func backup(run bool, path string, doc *md.Document) {
	bf, err := md.Backup(doc)
	fmt.Printf("Backed up %s to %s: %s\n", path, bf, func() string {
		if err != nil {
			return "FAILED"
		}
		return "SUCCESS"
	}())
	if err != nil {
		logging.Fatal(err.Error())
	}
}

func main() {
	opts := opts{}
	if _, err := flags.Parse(&opts); err != nil {
		logging.Fatal(err.Error())
	}

	log.OUTPUT = opts.Debug

	if opts.List {
		backupFiles, err := md.Backups()
		if err != nil {
			logging.Fatal(err.Error())
		}

		fmt.Printf("listing backups: %s", func() string {
			if len(backupFiles) == 0 {
				return "none found...\n"
			}
			return "\n"
		}())

		for _, bf := range backupFiles {
			fmt.Printf("%s [%s] %s %dkb\n", bf.ID, time.Unix(int64(bf.Time), 0), bf.Name, len(bf.Content)/1000)
		}

		return // the listing option should be run as an individual instruction
	}

	if len(opts.Doc) == 0 {
		logging.Fatal("the required flag `-f, --file' was not specified")
	}

	doc, err := md.Open(opts.Doc)
	if err != nil {
		logging.Fatal(err.Error())
	}
	defer doc.Close()

	backup(opts.Backup, opts.Doc, doc)

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
