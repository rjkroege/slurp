package main

import (
	"flag"
	"log"
	"os"
	"fmt"
	"strings"
	"path/filepath"
	"io/ioutil"

	"9fans.net/go/acme"
)

const usagetext = `slurp edwood-prefix remote-prefix

Copies every file saved in Edwood rooted at edwood-prefix to
remote-prefix. Intended usage is to run slurp on server while
Edwood runs on the client machine.
`

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 2 {
		fmt.Println(usagetext)
		os.Exit(-1)
	}

	edwoodprefix := args[0]
	remoteprefix := args[1]

	logreader, err := acme.Log()
	if err != nil {
		fmt.Println("can't connect to acme:", err)
		os.Exit(-1)
	}

	for {
		ev, err := logreader.Read()
		if err != nil {
			fmt.Println("can't read acme log", err )
			os.Exit(-1)
		}

		// log.Println(ev)

		if ev.Op == "put" && strings.HasPrefix(ev.Name, edwoodprefix) {
			// I could put this in a goroutine but am a bit worried that
			// Edwood might not be thrilled with this.
			copyEdwoodToRemote(ev, edwoodprefix, remoteprefix)
		}
	}
}

func copyEdwoodToRemote(ev acme.LogEvent, edwoodprefix, remoteprefix string) {
	log.Println("put file", ev.Name)
	w, err := acme.Open(ev.ID, nil)
	if err != nil {
		fmt.Printf("can't open acme window %q: %v", ev.Name, err)
		os.Exit(-1)
	}
	defer w.CloseFiles()

	buffercontents, err := w.ReadAll("body")
	if err != nil {
		fmt.Printf("can't read body of %s: %v", ev.Name, err)
		os.Exit(-1)
	}

	relpath := strings.TrimPrefix(ev.Name, edwoodprefix)
	remotepath := filepath.Join(remoteprefix, relpath)

	if err := ioutil.WriteFile(remotepath, buffercontents, 0644); err != nil {
		acme.Errf("can't write remote %q: %v", remotepath, err)
		if err := w.Ctl("dirty"); err != nil {
			fmt.Printf("can't retry %q: %v", ev.Name, err)
			os.Exit(-1)
		}
	}
}
