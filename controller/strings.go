package controller

import (
	"strings"
	"unicode"
)

func SentenceCase(fieldName string) string {
	return strings.ReplaceAll(strings.ToUpper(fieldName[:1])+strings.ToLower(fieldName[1:]), "Id ", "ID ")
}

func FirstLower(str string) string {
	return strings.ToLower(str[:1]) + str[1:]
}

func UpperSnakeCase(str string) string {
	var parts []string
	start := 0
	for end, r := range str {
		if end != 0 && unicode.IsUpper(r) {
			parts = append(parts, str[start:end])
			start = end
		}
	}
	if start != len(str) {
		parts = append(parts, str[start:])
	}
	return strings.ToUpper(strings.Join(parts, "_"))
}

func SnakeToCamelCase(snakeCase string) string {
	pieces := strings.Split(strings.ToLower(snakeCase), "_")
	for i, piece := range pieces {
		if i > 0 {
			pieces[i] = strings.ToUpper(piece[:1]) + piece[1:]
		}
	}
	return strings.Join(pieces, "")
}

func StrLen(text *string) int {
	if text != nil {
		return len(*text)
	}
	return 0
}
