// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package base defines shared basic pieces of the go command,
// in particular logging and the Command structure.
package base

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	exec "internal/execabs"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"

	"cmd/go/internal/cfg"
	"cmd/go/internal/str"
)

// A Command is an implementation of a go command
// like go build or go fix.
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(ctx context.Context, cmd *Command, args []string)

	// UsageLine is the one-line usage message.
	// The words between "go" and the first flag or argument in the line are taken to be the command name.
	UsageLine string

	// Short is the short description shown in the 'go help' output.
	Short string

	// Long is the long message shown in the 'go help <this-command>' output.
	Long string

	// Flag is a set of flags specific to this command.
	Flag flag.FlagSet

	// CustomFlags indicates that the command will do its own
	// flag parsing.
	CustomFlags bool

	// Commands lists the available commands and help topics.
	// The order here is the order in which they are printed by 'go help'.
	// Note that subcommands are in general best avoided.
	Commands []*Command
}

type command struct {
	Name        string
	LongName    string
	UsageLine   string
	Short       string
	Long        string
	Flags       []*goflag
	CustomFlags bool
	Commands    []*command
}

func newCommand(c *Command) *command {
	var flags []*goflag
	c.Flag.VisitAll(func(ff *flag.Flag) {
		flags = append(flags, newGoFlag(ff))
	})
	x := &command{
		Name:        c.Name(),
		LongName:    c.LongName(),
		UsageLine:   c.UsageLine,
		Short:       c.Short,
		Long:        c.Long,
		Flags:       flags,
		CustomFlags: c.CustomFlags,
		// Commands:    c.Commands,
	}
	for _, child := range c.Commands {
		x.Commands = append(x.Commands, newCommand(child))
	}
	return x
}

type flagValue struct {
	Type   string
	String string
}

type goflag struct {
	Name     string    // name as it appears on command line
	Usage    string    // help message
	Value    flagValue // value as set
	DefValue string    // default value (as text); for usage message
}

func newGoFlag(f *flag.Flag) *goflag {
	return &goflag{
		Name:  f.Name,
		Usage: f.Usage,
		Value: flagValue{
			Type:   reflect.TypeOf(f.Value).String(),
			String: fmt.Sprintf("%v", f.Value),
		},
		DefValue: f.DefValue,
	}
}

func (c Command) MarshalJSON() ([]byte, error) {
	// var flags []*goflag
	// c.Flag.VisitAll(func(ff *flag.Flag) {
	// 	flags = append(flags, newGoFlag(ff))
	// })
	// x := command{
	// 	Name:        c.Name(),
	// 	LongName:    c.LongName(),
	// 	UsageLine:   c.UsageLine,
	// 	Short:       c.Short,
	// 	Long:        c.Long,
	// 	Flags:       flags,
	// 	CustomFlags: c.CustomFlags,
	// 	Commands:    c.commands,
	// }
	return json.Marshal(newCommand(&c))
}

var CmdCmds = &Command{
	Run: func(ctx context.Context, cmd *Command, args []string) {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(Go); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		return
	},
	UsageLine: "go cmds",
	Short:     "dump go commands",
}

var Go = &Command{
	UsageLine: "go",
	Long:      `Go is a tool for managing Go source code.`,
	// Commands initialized in package main
}

// hasFlag reports whether a command or any of its subcommands contain the given
// flag.
func hasFlag(c *Command, name string) bool {
	if f := c.Flag.Lookup(name); f != nil {
		return true
	}
	for _, sub := range c.Commands {
		if hasFlag(sub, name) {
			return true
		}
	}
	return false
}

// LongName returns the command's long name: all the words in the usage line between "go" and a flag or argument,
func (c *Command) LongName() string {
	name := c.UsageLine
	if i := strings.Index(name, " ["); i >= 0 {
		name = name[:i]
	}
	if name == "go" {
		return ""
	}
	return strings.TrimPrefix(name, "go ")
}

// Name returns the command's short name: the last word in the usage line before a flag or argument.
func (c *Command) Name() string {
	name := c.LongName()
	if i := strings.LastIndex(name, " "); i >= 0 {
		name = name[i+1:]
	}
	return name
}

func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "Run 'go help %s' for details.\n", c.LongName())
	SetExitStatus(2)
	Exit()
}

// Runnable reports whether the command can be run; otherwise
// it is a documentation pseudo-command such as importpath.
func (c *Command) Runnable() bool {
	return c.Run != nil
}

var atExitFuncs []func()

func AtExit(f func()) {
	atExitFuncs = append(atExitFuncs, f)
}

func Exit() {
	for _, f := range atExitFuncs {
		f()
	}
	os.Exit(exitStatus)
}

func Fatalf(format string, args ...interface{}) {
	Errorf(format, args...)
	Exit()
}

func Errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
	SetExitStatus(1)
}

func ExitIfErrors() {
	if exitStatus != 0 {
		Exit()
	}
}

var exitStatus = 0
var exitMu sync.Mutex

func SetExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}

func GetExitStatus() int {
	return exitStatus
}

// Run runs the command, with stdout and stderr
// connected to the go command's own stdout and stderr.
// If the command fails, Run reports the error using Errorf.
func Run(cmdargs ...interface{}) {
	cmdline := str.StringList(cmdargs...)
	if cfg.BuildN || cfg.BuildX {
		fmt.Printf("%s\n", strings.Join(cmdline, " "))
		if cfg.BuildN {
			return
		}
	}

	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		Errorf("%v", err)
	}
}

// RunStdin is like run but connects Stdin.
func RunStdin(cmdline []string) {
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = cfg.OrigEnv
	StartSigHandlers()
	if err := cmd.Run(); err != nil {
		Errorf("%v", err)
	}
}

// Usage is the usage-reporting function, filled in by package main
// but here for reference by other packages.
var Usage func()
