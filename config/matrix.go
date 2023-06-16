package config

import (
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type KV struct {
	Key   string
	Value any
}

func flattenKeyedMatrix(m map[string][]any) [][]KV {
	keys := maps.Keys(m)
	slices.Sort(keys)

	matrix := make([][]any, len(keys))
	for i, k := range keys {
		matrix[i] = m[k]
	}

	rows := crossProduct(matrix)

	result := make([][]KV, len(rows))
	for i, row := range rows {
		keyValues := make([]KV, len(row))
		for j, v := range row {
			keyValues[j] = KV{
				Key:   keys[j],
				Value: v,
			}
		}
		result[i] = keyValues
	}

	return result
}

func crossProduct[T any](m [][]T) [][]T {
	if len(m) == 0 {
		return nil
	}

	// len(m) parameters, so each row shall be of size len(m).
	rowSize := len(m)

	total := 1
	for _, set := range m {
		total *= len(set)
	}

	subTotals := make([]int, len(m))
	for i := range m {
		subTotals[i] = func() int {
			total := 1
			for _, set := range m[i+1:] {
				total *= len(set)
			}
			return total
		}()
	}

	result := make([][]T, total)

	for i := 0; i < total; i++ {
		row := make([]T, rowSize)
		for pos := 0; pos < rowSize; pos++ {
			row[pos] = m[pos][(i/subTotals[pos])%len(m[pos])]
		}
		result[i] = row
	}

	return result
}
