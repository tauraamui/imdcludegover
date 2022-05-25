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
	Doc       string `short:"f" long:"file" description:"File to import includes into."`
	LookupDir string `short:"d" long:"dir" description:"Path to dir containing markdown files to search." default:"."`
	Backup    bool   `short:"b" long:"backup" description:"Backup the original target document beforehand."`
	List      bool   `short:"l" long:"list" description:"List all available backups."`
	Restore   string `short:"r" long:"restore" description:"Restore to a specified backup of given ID."`
	Debug     bool   `short:"v" long:"verbose" description:"Displays all internal/debug logs to assist with user level debugging."`
}

func backup(run bool, path string, doc *md.Document) {
	if run {
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
}

func restore(backupID string) (bool, error) {
	if len(backupID) == 0 {
		return false, nil
	}

	backupFiles, err := md.Backups()
	if err != nil {
		return true, err
	}

	for _, bkup := range backupFiles {
		if bkup.ID == backupID {
			fmt.Printf("restoring backup %s to %s\n", bkup.ID, bkup.Path)
			return true, md.Restore(bkup)
		}
	}

	return true, fmt.Errorf("unable to find backup of ID: %s", backupID)
}

func listBackups(run bool) bool {
	if run {
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
	}
	return run
}

func main() {
	opts := opts{}
	if _, err := flags.Parse(&opts); err != nil {
		logging.Fatal(err.Error())
	}

	log.OUTPUT = opts.Debug

	ran, err := restore(opts.Restore)
	if ran {
		if err != nil {
			logging.Fatal(err.Error())
		}
		return
	}

	if listBackups(opts.List) {
		return
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
