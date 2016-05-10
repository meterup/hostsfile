package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	"github.com/kevinburke/hostsfile/lib"
)

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(2)
	}
}

const usage = `Hostsfile manages your /etc/hosts file.

The commands are:

	add     <hostname> [<hostname...>] <ip>
	remove  <hostname> [<hostname...>]
	help    You're looking at it.

`

const addUsage = `Add a set of hostnames to your /etc/hosts file.

The last argument is the IP address to use for all host files.

Example: 

	hostsfile add www.facebook.com www.twitter.com 127.0.0.1
	hostsfile add --dry-run www.facebook.com 127.0.0.1
`

const removeUsage = `Remove a set of hostnames from your /etc/hosts file.

Example: 

	hostsfile remove www.facebook.com www.twitter.com
	hostsfile remove --dry-run www.facebook.com
`

func usg(msg string, fs *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(os.Stderr, msg)
		fs.PrintDefaults()
		os.Exit(2)
	}
}

func doAdd(hfile io.Reader, out io.Writer, args []string) error {
	if len(args) == 0 {
		return errors.New("Please provide a domain to add")
	}
	lastAddr := args[len(args)-1]
	ip, err := net.ResolveIPAddr("ip", lastAddr)
	if err != nil {
		return err
	}
	h, err := hostsfile.Decode(hfile)
	if err != nil {
		return err
	}
	for _, arg := range args[:len(args)-1] {
		err = h.Set(*ip, arg)
		if err != nil {
			return err
		}
	}
	return hostsfile.Encode(out, h)
}

func doRemove(hfile io.Reader, out io.Writer, args []string) error {
	h, err := hostsfile.Decode(hfile)
	if err != nil {
		return err
	}
	for _, arg := range args {
		// XXX remove arguments
		h.Remove(arg)
	}
	return hostsfile.Encode(out, h)
}

func doRename(writtenFile *os.File, renameTo string) error {
	if err := os.Chmod(writtenFile.Name(), 0644); err != nil {
		return err
	}

	return os.Rename(writtenFile.Name(), renameTo)
}

func checkWritable(file string) error {
	f, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func main() {
	flag.Usage = usg(usage, flag.CommandLine)
	dryRunArg := flag.Bool("dry-run", false, "Print the updated host file to stdout instead of writing it")
	fileArg := flag.String("file", "/etc/hosts", "File to read/write")

	addflags := flag.NewFlagSet("add", flag.ExitOnError)
	addflags.Usage = usg(addUsage, addflags)
	addflags.BoolVar(dryRunArg, "dry-run", false, "Print the updated host file to stdout instead of writing it")
	addflags.StringVar(fileArg, "file", "/etc/hosts", "File to read/write")

	removeflags := flag.NewFlagSet("remove", flag.ExitOnError)
	removeflags.Usage = usg(removeUsage, removeflags)
	removeflags.BoolVar(dryRunArg, "dry-run", false, "Print the updated host file to stdout instead of writing it")
	removeflags.StringVar(fileArg, "file", "/etc/hosts", "File to read/write")

	flag.Parse()
	if flag.NArg() < 2 {
		usg(usage, flag.CommandLine)()
	}
	subargs := flag.Args()[1:]
	switch flag.Arg(0) {
	case "add":
		err := addflags.Parse(subargs)
		checkError(err)
		if *dryRunArg == false {
			err = checkWritable(*fileArg)
			checkError(err)
		}
		f, err := os.Open(*fileArg)
		checkError(err)
		defer f.Close()
		if *dryRunArg {
			err = doAdd(f, os.Stdout, addflags.Args())
			checkError(err)
		} else {
			tmp, err := ioutil.TempFile("/tmp", "hostsfile-temp")
			checkError(err)
			defer tmp.Close()
			err = doAdd(f, tmp, addflags.Args())
			checkError(err)
			err = doRename(tmp, *fileArg)
			checkError(err)
		}
	case "remove":
		err := removeflags.Parse(subargs)
		checkError(err)
		if *dryRunArg == false {
			err = checkWritable(*fileArg)
			checkError(err)
		}
		f, err := os.Open(*fileArg)
		checkError(err)
		if *dryRunArg {
			err = doRemove(f, os.Stdout, removeflags.Args())
			checkError(err)
		} else {
			tmp, err := ioutil.TempFile("/tmp", "hostsfile-temp")
			checkError(err)
			defer tmp.Close()
			err = doRemove(f, tmp, removeflags.Args())
			checkError(err)
			err = doRename(tmp, *fileArg)
			checkError(err)
		}
	case "help":
		switch flag.Arg(1) {
		case "add":
			usg(addUsage, addflags)()
		case "remove":
			usg(removeUsage, removeflags)()
		default:
			usg(usage, flag.CommandLine)()
		}
	default:
		fmt.Fprintf(os.Stderr, "hostsfile: unknown subcommand \"%s\"\n\n", flag.Arg(0))
		usg(usage, flag.CommandLine)()
	}
}
