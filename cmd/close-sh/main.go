package main

import (
    "github.com/jessevdk/go-flags"
//	"net/http"
	"log"
	"os"
)

type CommandConfig interface {
	Command() (Command, error)
}

type Command interface {
	Execute() error
}

type Options struct {
	commands     map[string]CommandConfig
	cmd          string
	cmdConfig    CommandConfig
}

var Opts Options

func (options *Options) Register(name string, cmdConfig CommandConfig) {
    if options.commands == nil {
        options.commands = make(map[string]CommandConfig)
    }

    options.commands[name] = cmdConfig
}

func main() {
	options := Opts
	parser := flags.NewParser(&options, flags.Default)

	for cmd, cmdConfig := range options.commands {
		if _, err := parser.AddCommand(cmd, "", "", cmdConfig); err != nil {
			panic(err)
		}
	}

	if args, err := parser.Parse(); err != nil {
		os.Exit(1)
    } else if len(args) > 0 {
        log.Printf("flags Parser.Parser: extra arguments: %v\n", args)
        parser.WriteHelp(os.Stderr)
        os.Exit(1)
    }

    if command := parser.Active; command == nil {
        log.Fatalf("No command given\n")
    } else if cmdConfig, found := options.commands[command.Name]; !found {
        log.Fatalf("Invalid command: %v\n", command)
    } else {
        options.cmd = command.Name
        options.cmdConfig = cmdConfig
    }

    log.Printf("Command %v: %#v\n", options.cmd, options.cmdConfig)
	if cmd, err := options.cmdConfig.Command(); err != nil {
		os.Exit(1)
	} else {
	    cmd.Execute()
	}
}
