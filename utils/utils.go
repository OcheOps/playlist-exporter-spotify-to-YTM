package utils

import (
	"strconv"
	"strings"
)

func ParseIndices(input string) []int {
	var indices []int
	for _, s := range strings.Fields(input) {
		if i, err := strconv.Atoi(s); err == nil {
			indices = append(indices, i)
		}
	}
	return indices
}