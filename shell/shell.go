package shell

import (
	"github.com/qmsk/close/shell/config"
 	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

// Pluggable options, each command can Register() itself
type Options struct {
	Config       config.Config
	ConfigFile   string   `short:"c" long:"config" description:"configuration file"`
	
	commands  map[string]config.CommandConfig

	cmd       string
	subCmd    string
	cmdConfig config.CommandConfig
}

func (options Options) Url() string {
	return options.Config.URL
}

func (options Options) User() config.User {
	return options.Config.User
}

func (options Options) SubCmd() string {
	return options.subCmd
}

// Options itself is a CompositionalCommand
func (options *Options) Register(name string, cmdConfig config.CommandConfig) {
	if options.commands == nil {
		options.commands = make(map[string]config.CommandConfig)
	}

	options.commands[name] = cmdConfig
}

func (options Options) SubCommands() map[string]config.CommandConfig {
	return options.commands
}


func (options *Options) RegisterSub(cmd string, subcmd string, cmdConfig config.CommandConfig) {
	if options.commands == nil {
		return
	} else if command, found := options.commands[cmd]; !found {
		return
	} else {
		// Allowing to panic here if assertion fails
		compcmd := command.(config.CompositionalCommandConfig)
		compcmd.Register(subcmd, cmdConfig)
	}
}

func (options *Options) Parse() {
	parser := flags.NewParser(options, flags.Default)

	for cmd, cmdConfig := range options.commands {
		if command, err := parser.AddCommand(cmd, "", "", cmdConfig); err != nil {
			panic(err)
		} else if compcmd, ok := cmdConfig.(config.CompositionalCommandConfig); ok {
			subcmds := compcmd.SubCommands();
			for subcmd, subcmdConfig := range subcmds {
				if _, err := command.AddCommand(subcmd, "", "", subcmdConfig); err != nil {
					panic(err)
				}
			}
		}
	}

	if args, err := parser.Parse(); err != nil {
		os.Exit(1)
	} else if len(args) > 0 {
		log.Printf("flags Parser.Parser: extra arguments: %v\n", args)
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	if options.ConfigFile != "" {
		if config, err := config.NewConfig(options.ConfigFile); err != nil {
			log.Printf("Error parsing the configuration file: %v\n", err)
		} else {
			options.Config = *config
			log.Printf("Setting a user from config file: %v, id=%v\n", options.ConfigFile, config.User.Id)
		}
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
		log.Printf("Main: config.Command: %v", err)
	} else {
		if err := cmd.Execute(); err != nil {
			log.Printf("Main: command.Execute: %v", err)
		}
	}
}
