// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package ux

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

var Table *CustomTable

// Wrapper for ux.DefaultTable to support JSON rendering.
type CustomTable struct {
	table.Writer
	Name        string
	Headers     []string
	Data        map[string]interface{} // Holds the data for JSON rendering
	ArrayData   []map[string]interface{}
	AllData     map[string]interface{}
	headerAdded bool
	isArray     bool // Flag to differentiate between flat and hierarchical tables
	AsJson      *bool
}

// Initializes a new CustomTable.
func NewCustomTable(name string, headers table.Row) CustomTable {
	if Table == nil {
		Table = &CustomTable{
			Writer:    table.NewWriter(),
			Name:      name,
			Data:      make(map[string]interface{}),
			ArrayData: []map[string]interface{}{},
			isArray:   headers != nil,
			AllData:   make(map[string]interface{}),
		}
	} else {
		// Reset the table except for the json array AllData
		Table.Name = name
		Table.Data = make(map[string]interface{})
		Table.ArrayData = []map[string]interface{}{}
		Table.isArray = headers != nil
		Table.headerAdded = false
		Table.Writer = table.NewWriter()
	}
	if headers != nil {
		Table.AppendHeader(headers)
	}
	return *Table
}

func (jt *CustomTable) SetAsJson(AsJson *bool) {
	jt.AsJson = AsJson
	Logger.SetMute(AsJson)
}

// Appends headers to the table and stores them for JSON conversion.
func (jt *CustomTable) AppendHeader(headers table.Row) {
	if jt.headerAdded {
		return
	}
	jt.headerAdded = true
	if *jt.AsJson {
		jt.Headers = make([]string, len(headers))
		for i, h := range headers {
			jt.Headers[i] = toSnakeCase(fmt.Sprint(h)) // JSON-compatible keys
		}
		return
	}
	jt.Writer.AppendHeader(headers)
}

// AppendRow appends a row to the table and updates the JSON structure.
func (jt *CustomTable) AppendRow(row table.Row, rowConfig ...table.RowConfig) {
	if *jt.AsJson {
		if jt.isArray {
			jt.appendToArray(row)
		} else {
			jt.appendToHierarchy(row)
		}
		return
	}
	jt.Writer.AppendRow(row, rowConfig...)
}

// appendToArray handles rows for array-based tables (e.g., Smart Contracts).
func (jt *CustomTable) appendToArray(row table.Row) {
	if len(jt.Headers) == 0 {
		panic("Headers must be set before appending rows.")
	}
	if len(row) > len(jt.Headers) {
		panic("Row length exceeds header count.")
	}

	rowData := make(map[string]interface{})
	for i, value := range row {
		if i < len(jt.Headers) {
			rowData[jt.Headers[i]] = noColorNoLines(fmt.Sprint(value))
		}
	}
	jt.ArrayData = append(jt.ArrayData, rowData)
}

// appendToHierarchy handles rows for hierarchical tables
func (jt *CustomTable) appendToHierarchy(row table.Row) {
	if len(row) < 2 {
		return
	}

	key := toSnakeCase(fmt.Sprint(row[0])) // Convert key to JSON-compatible
	value := row[1]

	// Check for nested rows
	if len(row) > 2 {
		if row[1] == row[2] {
			jt.Data[key] = noColorNoLines(fmt.Sprint(value))
			return
		}
		if _, ok := jt.Data[key]; !ok {
			jt.Data[key] = make(map[string]interface{})
		}
		if subMap, ok := jt.Data[key].(map[string]interface{}); ok {
			subKey := toSnakeCase(fmt.Sprint(row[1]))
			subValue := noColorNoLines(fmt.Sprint(row[2]))
			subMap[subKey] = subValue
			jt.Data[key] = subMap
		}
	} else {
		jt.Data[key] = noColorNoLines(fmt.Sprint(value))
	}
}

// Renders all table as JSON.
func (jt *CustomTable) PrintIfJson() {
	if !*jt.AsJson {
		return
	}
	jsonData, err := json.Marshal(jt.AllData)
	Logger.OneShotUnmute()
	if err != nil {
		Logger.PrintToUser(fmt.Sprintf(`{"error": "failed to render JSON: %s"}`, err))
		return
	}
	Logger.PrintToUser(string(jsonData))
}

// Render renders the table as a string or JSON based on the input flag.
func (jt *CustomTable) Render() string {
	if *jt.AsJson {
		name := toSnakeCase(jt.Name)
		if jt.isArray {
			jt.AllData[name] = jt.ArrayData
		} else {
			jt.AllData[name] = jt.Data
		}
		return ""
	}
	return jt.Writer.Render()
}

func DefaultTable(title string, header table.Row) CustomTable {
	t := NewCustomTable(title, header)
	t.Style().Title.Align = text.AlignCenter
	t.Style().Title.Format = text.FormatUpper
	t.Style().Options.SeparateRows = true
	t.SetTitle(title)
	return t
}

// Utils

// Regex to match ANSI escape sequences
var reColors = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Regex to match parentheses
var reParentheses = regexp.MustCompile(`\((.*?)\)`)

// Regex to match non-alphanumeric characters
var reNonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// noColorNoLines removes ANSI escape sequences and converts newlines to hyphens.
func noColorNoLines(input string) string {
	input = reColors.ReplaceAllString(input, "")
	input = strings.ReplaceAll(input, "\n", "-")
	return input
}

// toSnakeCase converts a string to snake_case. It removes ANSI escape sequences, replaces parentheses with underscores, and converts newlines to hyphens.
func toSnakeCase(input string) string {

	input = noColorNoLines(input)

	input = reParentheses.ReplaceAllStringFunc(input, func(match string) string {
		inner := strings.Trim(match, "()")
		if inner == "" {
			return "_"
		}
		return "_" + inner + "_"
	})

	input = reNonAlphanumeric.ReplaceAllString(input, "_")
	input = strings.ToLower(strings.Trim(input, "_"))

	return input
}
