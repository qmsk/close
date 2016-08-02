package shell

import (
 	"github.com/jessevdk/go-flags"
	//	"net/http"
	"log"
	"os"
)

// Shell commands
type CommandConfig interface {
	Command(options CommonOptions) (Command, error)
}

type Command interface {
	Execute() error
}

// Options, common for all commands
type CommonOptions interface {
	Url()     string
	User()    User
	SubCmd()  string
}

// Pluggable options, each command can Register() itself
type Options struct {
	URL       string   `short:"u" long:"url" required:"true" description:"controller URL"`

	AuthUser  User
	
	commands  map[string]CommandConfig

	cmd       string
	subCmd    string
	cmdConfig CommandConfig
}

func (options Options) Url() string {
	return options.URL
}

func (options Options) User() User {
	return options.AuthUser
}

func (options Options) SubCmd() string {
	return options.subCmd
}

func (options *Options) Register(name string, cmdConfig CommandConfig) {
	if options.commands == nil {
		options.commands = make(map[string]CommandConfig)
	}

	options.commands[name] = cmdConfig
}

func (options *Options) Parse() {
	parser := flags.NewParser(options, flags.Default)

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
		if subcommand := command.Active; subcommand != nil {
			options.subCmd = subcommand.Name
		}
	}
}

func Main(opts Options) {
	//log.Printf("Command %v: %#v\n", Opts.cmd, Opts.cmdConfig)
	if cmd, err := opts.cmdConfig.Command(opts); err != nil {
		os.Exit(1)
	} else {
		cmd.Execute()
	}
}
