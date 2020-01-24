package main

import (
	"strconv"

	"github.com/Sirupsen/logrus"
)

// utility function
func debug(format string) {
	if settings.Debug {
		logrus.Info(format)
	}
}
func debugf(format string, args ...interface{}) {
	if settings.Debug {
		logrus.Infof(format, args)
	}
}

func twos_comp(inp string) int64 {
	i, _ := strconv.ParseInt("0x"+inp, 0, 64)
	return i - 256
}
