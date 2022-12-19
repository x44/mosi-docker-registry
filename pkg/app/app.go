package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ProgramCommandArg struct {
	Arg         string
	Description string
}

type ProgramCommand struct {
	Run         func(args []string)
	Cmd         string
	Description string
	Args        []ProgramCommandArg
}

func GetProgCommand(progCommands *[]ProgramCommand, cmd string) *ProgramCommand {
	for _, progCommand := range *progCommands {
		if progCommand.Cmd == cmd {
			return &progCommand
		}
	}
	return nil
}

func PrintHelpForCommands(progCommands *[]ProgramCommand) {
	maxLen := 0
	for _, progCommand := range *progCommands {
		if len(progCommand.Cmd) > maxLen {
			maxLen = len(progCommand.Cmd)
		}
	}
	fmtStr := "%-" + strconv.Itoa(maxLen) + "s    %s\n"

	for _, progCommand := range *progCommands {
		ds := strings.Split(progCommand.Description, "\n")
		a := progCommand.Cmd
		for _, d := range ds {
			fmt.Printf(fmtStr, a, d)
			a = ""
		}
	}
}

func PrintHelpForCommand(progCommand *ProgramCommand) {
	exe := os.Args[0]
	if len(progCommand.Args) == 0 {
		fmt.Printf("Usage: %s %s\n", filepath.Base(exe), progCommand.Cmd)
		fmt.Printf("\n%s\n", progCommand.Description)
	} else {
		fmt.Printf("Usage: %s %s <arguments>\n", filepath.Base(exe), progCommand.Cmd)
		fmt.Printf("\n%s\n", progCommand.Description)
		maxLen := 0
		for _, progCommandArg := range progCommand.Args {
			if len(progCommandArg.Arg) > maxLen {
				maxLen = len(progCommandArg.Arg)
			}
		}
		fmtStr := "%-" + strconv.Itoa(maxLen) + "s    %s\n"

		fmt.Printf("\nArguments:\n\n")
		for _, progCommandArg := range progCommand.Args {
			ds := strings.Split(progCommandArg.Description, "\n")
			a := progCommandArg.Arg
			for _, d := range ds {
				fmt.Printf(fmtStr, a, d)
				a = ""
			}
		}
	}
}

func CheckError(msg string, err error) {
	if err != nil {
		if len(msg) > 0 {
			fmt.Printf("%s: %v\n", msg, err)
		} else {
			fmt.Printf("%v\n", err)
		}
		os.Exit(1)
	}
}

func GetHomeDir() string {
	home, err := os.UserHomeDir()
	CheckError("Failed to get user home dir", err)
	return home
}

func CleanArgs(args *[]string) {
	tmp := []string{}
	for _, a := range *args {
		if len(a) > 0 {
			tmp = append(tmp, a)
		}
	}
	*args = tmp
}

func BoolArg(key string, def bool, args *[]string) bool {
	for i, a := range *args {
		if a == key {
			(*args)[i] = ""
			return true
		}
	}
	return def
}

func StringArg(key string, def string, args *[]string) string {
	key2 := key + "="
	for i, a := range *args {
		if a == key {
			(*args)[i] = ""
			if i < len(*args)-1 {
				i++
				ret := (*args)[i]
				(*args)[i] = ""
				return ret
			}
			return def
		}
		if strings.HasPrefix(a, key2) {
			ret := (*args)[i][len(key2):]
			(*args)[i] = ""
			return ret
		}
	}
	return def
}
