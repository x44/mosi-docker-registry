package cmdline

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
