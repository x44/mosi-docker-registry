package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	"golang.org/x/exp/slices"

	"mosi-docker-registry/pkg/app"
	"mosi-docker-registry/pkg/client"
	"mosi-docker-registry/pkg/config"
	"mosi-docker-registry/pkg/logging"
	"mosi-docker-registry/pkg/server"
)

var Version = "DEV"

const (
	serviceName        = "MosiDockerRegistry"
	serviceDisplayName = "Mosi Docker Registry"
	serviceDescription = "The Mosi Docker Registry"

	cfgFileName = "config.json"
	logFileName = "mosi.log"
)

var helpCommands = [...]string{"-h", "--help", "help", "-?"}
var serviceCommands = [...]string{"install", "uninstall", "start", "stop", "restart", "status"}
var programCommands = []app.ProgramCommand{
	{
		Run:         printVersion,
		Cmd:         "version",
		Description: "Print program version.",
		Args:        []app.ProgramCommandArg{},
	},
	{
		Run:         client.List,
		Cmd:         "ls",
		Description: "List images, tags and layers.",
		Args: []app.ProgramCommandArg{
			{
				Arg: "[name]:[tag]", Description: "Image name and tag filter.\nExamples:\nls                  List all images.\nls my*              List all images starting with 'my'.\nls myimage:1.*      List layers of 'myimage' with tags starting with '1.'.\nls :                List layers of all images.\n",
			},
			{
				Arg: "-s host:port", Description: "Run the command on the given machine. Optional.",
			},
			{
				Arg: "-u username", Description: "Authenticate with the given username. Optional.",
			},
			{
				Arg: "-p password", Description: "Authenticate with the given password. Optional.",
			},
		},
	},
}

type program struct {
	Exe  string
	Cwd  string
	Cmd  *app.ProgramCommand
	Args []string
}

var cfgDir = ""
var logDir = ""
var cfgFile = ""
var logFile = ""
var serviceLogger service.Logger

func run(exe, cwd string, cmd *app.ProgramCommand, args []string) {

	InitLogging(logging.INFO, logging.INFO, logging.INFO, true, true, true)

	if cmd != nil {
		// the CLI may be used from a remote machine without a local server & config on that remote machine, so just try to read the config, but do not create a config file
		config.ReadIfExists(cwd, cfgFile)
		cmd.Run(args)
		os.Exit(0)
	}

	if !config.ReadOrCreate(cwd, cfgFile) {
		// config file did not exist, the default config file got created, exit and let user do initial config
		os.Exit(0)
	}

	// re-init logging with config log settings
	InitLogging(config.LogLevelService(), config.LogLevelConsole(), config.LogLevelFile(), true, true, true)

	logging.Info("MAIN", "run()\nrunning as a service: %v\ncwd: %s\nexe: %s\ncfg: %s\nlog: %s\nrepo: %s\ncmd: %v\nargs: %v\n", !service.Interactive(), cwd, exe, cfgFile, logFile, config.RepoDir(), cmd, args)
	server.Start(Version)
}

func (p *program) Start(s service.Service) error {
	go run(p.Exe, p.Cwd, p.Cmd, p.Args)
	return nil
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func InitLogging(levelService, levelConsole, levelFile int, printDate, printTime, printMicros bool) {
	logging.Init(serviceLogger, !service.Interactive(), logFile, levelService, levelConsole, levelFile, printDate, printTime, printMicros)
}

func isServiceCommand(cmd string) bool {
	return slices.Contains(serviceCommands[:], cmd)
}

func isProgramCommand(cmd string) bool {
	return app.GetProgCommand(&programCommands, cmd) != nil
}

func toString(a []string) string {
	s := fmt.Sprintf("%v", a)
	return s[1 : len(s)-1]
}

func printVersion(args []string) {
	fmt.Printf("%s %s\n", serviceDisplayName, Version)
}

func printHelp() {
	exe := os.Args[0]
	fmt.Printf("Usage: %s <service-command> [service-command] ...\n", filepath.Base(exe))
	fmt.Printf("Usage: %s <program-command> [arguments]\n", filepath.Base(exe))
	fmt.Printf("\nMultiple service commands can be passed as a space-separated list.\n")
	fmt.Printf("If no command is given, the server will start in interactive mode.\n")
	fmt.Printf("\nService Commands:\n\n")
	fmt.Printf("%s\n", toString(serviceCommands[:]))
	fmt.Printf("\nProgram Commands:\n\n")
	app.PrintHelpForCommands(&programCommands)
	fmt.Printf("\nFor further help run %s -h <program-command>\n", filepath.Base(exe))
	os.Exit(0)
}

func printHelpForCommand(cmd string) {
	if isServiceCommand(cmd) {
		fmt.Printf("No help for service-command '%s'. Run with -h for help.\n", cmd)
		os.Exit(1)
	}
	progCommand := app.GetProgCommand(&programCommands, cmd)
	if progCommand == nil {
		fmt.Printf("Unknown command '%s'. Run with -h for help.\n", cmd)
		os.Exit(1)
	}
	app.PrintHelpForCommand(progCommand)
	os.Exit(0)
}

func handleServiceCommand(s service.Service, cmd string) {
	fmt.Printf("Running service command '%s'... ", cmd)
	if cmd == "status" {
		status, err := s.Status()
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		} else {
			fmt.Printf("%s\n", serviceStatusString(status))
		}
	} else {
		err := service.Control(s, cmd)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		} else {
			fmt.Printf("OK\n")
		}
	}
}

func serviceStatusString(status service.Status) string {
	statusStr := "Unknown"
	switch status {
	case service.StatusRunning:
		statusStr = "Running"
	case service.StatusStopped:
		statusStr = "Stopped"
	}
	return statusStr
}

func isDevModeArg(args []string) (bool, []string) {
	if len(args) > 0 && args[0] == "-dev" {
		return true, args[1:]
	}
	return false, args
}

func isHelpArg(args []string) (bool, []string) {
	if len(args) > 0 && slices.Contains(helpCommands[:], args[0]) {
		return true, args[1:]
	}
	return false, args
}

func getCommand(args []string) (string, []string) {
	if len(args) > 0 {
		return args[0], args[1:]
	}
	return "", args
}

func getWorkDirAndProgramExe(isDevMode bool) (string, string) {
	var exe string
	var err error
	if isDevMode {
		exe = findDevModeProgramExe()
	} else {
		exe, err = os.Executable()
		if err != nil {
			fmt.Printf("Failed to get executable")
			os.Exit(1)
		}
	}
	cwd := filepath.Dir(exe)
	return cwd, exe
}

func findDevModeProgramExe() string {
	exe, _ := os.Executable()
	ext := filepath.Ext(exe)

	// the dev mode executable we are looking for
	program := "program" + ext

	if program == filepath.Base(exe) {
		// we are the dev mode executable
		return exe
	}

	// when launched via launch.json with "cwd": "_dev"
	exe, _ = filepath.Abs(program)
	if _, err := os.Stat(exe); err == nil {
		return exe
	}
	// when launched via go command in the project's root directory
	exe, _ = filepath.Abs(filepath.Join("_dev", program))
	if _, err := os.Stat(exe); err == nil {
		return exe
	}
	fmt.Printf("DevMode executable '%s' not found.\n"+
		"It must be started before service commands can be applied.\n"+
		"Run the 'Launch Program' configuration from launch.json to start it.\n", program)
	os.Exit(1)
	return ""
}

func main() {

	args := os.Args[1:]

	isDevMode, args := isDevModeArg(args)
	isHelp, args := isHelpArg(args)

	cmd, args := getCommand(args)

	if isHelp {
		if cmd == "" {
			printHelp()
		} else {
			printHelpForCommand(cmd)
		}
	}

	if cmd != "" && !isServiceCommand(cmd) && !isProgramCommand(cmd) {
		fmt.Printf("Unknown command '%s'. Run with -h for help.\n", cmd)
		os.Exit(1)
	}

	cwd, exe := getWorkDirAndProgramExe(isDevMode)

	cfgDir = filepath.Join(cwd, "conf")
	logDir = filepath.Join(cwd, "logs")
	cfgFile = filepath.Join(cfgDir, cfgFileName)
	logFile = filepath.Join(logDir, logFileName)

	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
	}

	if isDevMode {
		svcConfig.Name += "DEV"
		svcConfig.DisplayName += " DEV"
		svcConfig.Executable = exe
		svcConfig.Arguments = []string{"-dev"}
	}

	prg := &program{}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if isServiceCommand(cmd) {
		for {
			handleServiceCommand(s, cmd)
			cmd, args = getCommand(args)
			if cmd == "" {
				break
			}
			if !isServiceCommand(cmd) {
				fmt.Printf("Unknown command '%s'. Run with -h for help.\n", cmd)
				os.Exit(1)
				break
			}
		}
		return
	}

	serviceLogger, err = s.Logger(nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	prg.Exe = exe
	prg.Cwd = cwd
	prg.Cmd = app.GetProgCommand(&programCommands, cmd)
	prg.Args = args

	err = s.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
