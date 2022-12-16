package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func InputString(msg, def string, dst **string) {
	fmt.Printf("%s", msg)
	if len(def) > 0 {
		fmt.Printf(" [%s]", def)
	}
	fmt.Printf(": ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		s := scanner.Text()
		*dst = new(string)
		if len(s) == 0 {
			**dst = def
			return
		}
		**dst = s
	}
}

func InputBool(msg, def string) bool {
	yn := "[y/n]"
	if def == "y" || def == "Y" {
		yn = "[Y/n]"
	} else if def == "n" || def == "N" {
		yn = "[y/N]"
	}
	str := fmt.Sprintf("%s %s? ", msg, yn)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("%s", str)
		if scanner.Scan() {
			s := scanner.Text()
			if len(def) > 0 && len(s) == 0 {
				s = def
			}
			if s == "y" || s == "Y" {
				return true
			} else if s == "n" || s == "N" {
				return false
			}
		}
	}
}

func InputUserAndPassword(defaultUsr string) (string, string, error) {
	var usr *string
	InputString("Username", defaultUsr, &usr)
	fmt.Printf("Password: ")

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", "", err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	pwd, err := term.ReadPassword(fd)
	if err != nil {
		return "", "", err
	}
	fmt.Printf("\n")
	return *usr, string(pwd), nil
}

func SplitByCommaOrSpaceAndTrim(s string) []string {
	a := strings.Split(s, ",")
	b := []string{}
	for _, aa := range a {
		t := strings.Split(aa, " ")
		for _, tt := range t {
			tt = strings.TrimSpace(tt)
			if len(tt) > 0 {
				b = append(b, tt)
			}
		}
	}
	return b
}
