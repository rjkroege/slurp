package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"net"
	"time"

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

	ender, err := net.Dial("unix", "/tmp/ns.root.:0/sessionender")
	if err != nil {
		fmt.Println("can't connect to sessionender. oh well:", err)
		ender = nil
	}
	lastwritten := time.Now()

	for {
		ev, err := logreader.Read()
		if err != nil {
			fmt.Println("can't read acme log", err)
			os.Exit(-1)
		}

		// log.Println(ev)

		if ev.Op == "put" && strings.HasPrefix(ev.Name, edwoodprefix) {
			// I am worried that Edwood will have trouble with this but have a go.
			go copyEdwoodToRemote(ev, edwoodprefix, remoteprefix)

			lastwritten = extendsessionender(ender, lastwritten)
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
		return
	}

	relpath := strings.TrimPrefix(ev.Name, edwoodprefix)
	remotepath := filepath.Join(remoteprefix, relpath)

	dir := filepath.Dir(remotepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("can't make a dir for file %q: %v", dir, err)
		return
	}

	if err := ioutil.WriteFile(remotepath, buffercontents, 0644); err != nil {
		acme.Errf("can't write remote %q: %v", remotepath, err)
		if err := w.Ctl("dirty"); err != nil {
			fmt.Printf("can't retry %q: %v", ev.Name, err)
		}
	}
}

// extendsessionender tells sessionender to defer shutdown.
func extendsessionender(ender net.Conn, lasttime time.Time) time.Time {
	// Without an ender, can't do anything so just skip and continue.
	if ender == nil {
		return lasttime
	}

	// Don't pester sessionender too often.
	now := time.Now()
	if now.Sub(lasttime) < 15 * time.Second {
		return lasttime
	}

	if _, err := ender.Write([]byte("helo")); err != nil {
		fmt.Println("can't write to sessionender? shucks:", err)
		// Should be lasttime or now returned here. Assume now?
	}
	return now
}
