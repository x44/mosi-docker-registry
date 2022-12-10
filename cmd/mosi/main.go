package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	"golang.org/x/exp/slices"

	"mosi-docker-repo/pkg/config"
	"mosi-docker-repo/pkg/logging"
	"mosi-docker-repo/pkg/server"
)

var Version = "DEV"

const (
	serviceName        = "MosiDockerRepo"
	serviceDisplayName = "Mosi Docker Repository"
	serviceDescription = "Mosi docker repository."

	cfgFileName = "config.json"
	logFileName = "mosi.log"
)

var serviceCommands = [...]string{"start", "stop", "restart", "install", "uninstall", "status"}
var programCommands = [...]string{"version", "test"}
var helpCommands = [...]string{"-h", "--help", "help", "-?"}

type program struct {
	Exe  string
	Cwd  string
	Cmd  string
	Args []string
}

var cfgDir = ""
var logDir = ""
var cfgFile = ""
var logFile = ""
var serviceLogger service.Logger

func run(exe, cwd, cmd string, args []string) {

	if cmd == "version" {
		fmt.Printf("%s %s\n", serviceDisplayName, Version)
		os.Exit(0)
	}
	if cmd == "test" {
		os.Exit(0)
	}

	InitLogging(logging.INFO, logging.INFO, logging.INFO, true, true, true)

	config.ReadConfig(cwd, cfgFile)

	// re-init logging with config log settings
	InitLogging(config.LogLevelService(), config.LogLevelConsole(), config.LogLevelFile(), true, true, true)

	logging.Info("PROG", "run()\nrunning as a service: %v\nexe: %s\ncfg: %s\nlog: %s\ncwd: %s\ncmd: %s\nargs: %v\n", !service.Interactive(), exe, cfgFile, logFile, cwd, cmd, args)
	fmt.Printf("fmt ****** SIMULATED ERROR ********\n")
	log.Printf("log ****** SIMULATED ERROR ********\n")
	logging.Error("PROG", "****** SIMULATED ERROR ********")
	os.Exit(1)
	server.Start(Version)
}

func (p *program) Start(s service.Service) error {
	go run(p.Exe, p.Cwd, p.Cmd, p.Args)
	// return errors.New("THIS IS A TEST ERROR")
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
	return slices.Contains(programCommands[:], cmd)
}

func toString(a []string) string {
	s := fmt.Sprintf("%v", a)
	return s[1 : len(s)-1]
}

func printHelp() {
	exe := os.Args[0]
	fmt.Printf("Usage: %s [service-command service-command ...] | [program-command]\n", filepath.Base(exe))
	fmt.Printf("\nIf no command is given, the program will run in interactive mode.\n")
	fmt.Printf("Multiple service commands can be passed as a space-separated list.\n")
	fmt.Printf("\nSERVICE COMMANDS:\n\n")
	fmt.Printf("%s\n", toString(serviceCommands[:]))
	fmt.Printf("\nPROGRAM COMMANDS:\n\n")
	fmt.Printf("%s\n\n", toString(programCommands[:]))
	fmt.Printf("%s        %s\n", "version", "Prints the version.")
	fmt.Printf("%s        %s\n", "test", "Is a test command")
	fmt.Printf("\nFor further help run %s <program-command> -h\n", filepath.Base(exe))
	os.Exit(0)
}

func printHelpForCommand(cmd string) {
	if isServiceCommand(cmd) {
		fmt.Printf("No help for service-command '%s'. Run with -h for help.\n", cmd)
		os.Exit(1)
	}
	if !isProgramCommand(cmd) {
		fmt.Printf("Unknown command '%s'. Run with -h for help.\n", cmd)
		os.Exit(1)
	}

	exe := os.Args[0]
	fmt.Printf("Usage: %s %s ", filepath.Base(exe), cmd)

	switch cmd {
	case "test":
		fmt.Printf("\n\nNo further help available\n")
	}
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
	if isDevMode {
		exe := findDevModeProgramExe()
		cwd := filepath.Dir(exe)
		return cwd, exe
	} else {
		exe, err := os.Executable()
		if err != nil {
			fmt.Printf("Failed to get executable")
			os.Exit(1)
		}
		cwd := ""
		if service.Interactive() {
			cwd, err = filepath.Abs("")
			if err != nil {
				fmt.Printf("Failed to get workdir")
				os.Exit(1)
			}
		} else {
			cwd = filepath.Dir(exe)
		}
		return cwd, exe
	}
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
	if isHelp {
		printHelp()
	}

	cmd, args := getCommand(args)
	if cmd != "" {
		isHelp, args = isHelpArg(args)
		if isHelp {
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
	prg.Cmd = cmd
	prg.Args = args

	err = s.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
