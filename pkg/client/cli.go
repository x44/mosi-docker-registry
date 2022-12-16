package client

import (
	"fmt"
	"mosi-docker-registry/pkg/app"
	"os"
	"strconv"
	"strings"
)

// handled in /pkg/server/cli.go

func create(args *[]string, minArgs int) *mosiClient {
	client := createClient(*args)
	app.ArgsClean(args)

	if len(*args) < minArgs {
		fmt.Printf("Not enough arguments. Run with -h for help.\n")
		os.Exit(1)
	}

	return client
}

func makePath(base string, args []string) string {
	s := base
	for _, a := range args {
		if !strings.HasSuffix(s, "/") {
			s += "/"
		}
		s += a
	}
	return s
}

func printTable(fields []any, rows []any) {
	lens := make([]int, len(fields))
	for i, field := range fields {
		s := fmt.Sprintf("%v", field)
		l := len(s)
		if l > lens[i] {
			lens[i] = l
		}
	}
	for _, r := range rows {
		row := r.([]any)
		for i, val := range row {
			s := fmt.Sprintf("%v", val)
			l := len(s)
			if l > lens[i] {
				lens[i] = l
			}
		}
	}
	fmtstr := ""
	space := 4
	total := -space
	for _, l := range lens {
		fmtstr += "%-" + strconv.Itoa(l+space) + "v"
		total += l + space
	}
	fmtstr += "\n"

	fmt.Printf(fmtstr, fields[:]...)
	fmt.Printf("%s\n", strings.Repeat("-", total))
	for _, r := range rows {
		row := r.([]any)
		fmt.Printf(fmtstr, row...)
	}
	fmt.Printf("\n")
}

func printTables(json *map[string]interface{}) {
	if tables, ok := (*json)["tables"].([]any); ok {
		for _, t := range tables {
			if table, ok := t.(map[string]any); ok {
				printTable(table["fields"].([]any), (table)["values"].([]any))
			}
		}
	}
}

func List(args []string) {
	client := create(&args, 0)

	json := client.Get(makePath("/v2/cli/ls/", args))
	// fmt.Printf("%v\n", json)
	// printTable((*json)["fields"].([]any), (*json)["values"].([]any))
	printTables(json)
}
