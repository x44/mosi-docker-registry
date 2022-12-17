package client

import (
	"fmt"
	"mosi-docker-registry/pkg/app"
	"mosi-docker-registry/pkg/json"
	"os"
	"strconv"
	"strings"
)

func handleError(msg string) {
	fmt.Printf("ERROR: %s\n", msg)
	os.Exit(1)
}

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

func printTable(table *json.JsonObject) {
	jsonFields := table.GetArray("fields", nil)
	jsonRows := table.GetArray("rows", nil)
	if jsonFields == nil {
		handleError("no fields in table")
	}
	if jsonRows == nil {
		handleError("no rows in table")
	}

	fields := jsonFields.ToStringArray("")

	lens := make([]int, jsonFields.Len())

	for i, field := range fields {
		l := len(field)
		if l > lens[i] {
			lens[i] = l
		}
	}
	for r := 0; r < jsonRows.Len(); r++ {
		jsonRow := jsonRows.GetArray(r, nil)
		if jsonRow == nil {
			handleError("missing row in table")
		}
		values := jsonRow.ToStringArray("")
		for i, value := range values {
			l := len(value)
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

	fmt.Printf(fmtstr, jsonFields.ToAnyArray()...)
	fmt.Printf("%s\n", strings.Repeat("-", total))
	for r := 0; r < jsonRows.Len(); r++ {
		fmt.Printf(fmtstr, jsonRows.GetArrayUnsafe(r).ToAnyArray()...)
	}
	fmt.Printf("\n")
}

func printTables(jsonObject *json.JsonObject) {
	tables := jsonObject.GetArray("tables", nil)
	if tables == nil {
		handleError("no tables in response")
	}
	for i := 0; i < tables.Len(); i++ {
		printTable(tables.GetObjectUnsafe(i))
	}
}

func List(args []string) {
	client := create(&args, 0)
	jsonObject := client.Get(makePath("/v2/cli/ls/", args))
	printTables(jsonObject)
}
