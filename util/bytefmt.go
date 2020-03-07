package util

import (
	"strconv"
	"strings"
)

const (
	// BYTE 字节
	BYTE = 1 << (10 * iota)
	// KILOBYTE 千字节
	KILOBYTE
	// MEGABYTE 兆字节
	MEGABYTE
	// GIGABYTE 吉字节
	GIGABYTE
	// TERABYTE 太字节
	TERABYTE
	// PETABYTE 拍字节
	PETABYTE
	// EXABYTE 艾字节
	EXABYTE
)

// Bytefmt returns a human-readable byte string of the form 10M, 12.5K, and so forth.  The following units are available:
//	E: Exabyte
//	P: Petabyte
//	T: Terabyte
//	G: Gigabyte
//	M: Megabyte
//	K: Kilobyte
//	B: Byte
// The unit that results in the smallest number greater than or equal to 1 is always chosen.
func Bytefmt(bytes uint64) string {
	unit := ""
	value := float64(bytes)

	switch {
	case bytes >= EXABYTE:
		unit = "E"
		value = value / EXABYTE
	case bytes >= PETABYTE:
		unit = "P"
		value = value / PETABYTE
	case bytes >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case bytes >= BYTE:
		unit = "B"
	case bytes == 0:
		return "0B"
	}

	result := strconv.FormatFloat(value, 'f', 2, 64)
	result = strings.TrimSuffix(result, ".0")
	return result + unit
}
