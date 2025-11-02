package logging

import (
	"log/slog"
	"os"
	"runtime/debug"
)

func NewProgramAttr() slog.Attr {
	buildInfo, _ := debug.ReadBuildInfo()
	hostname, _ := os.Hostname()

	return slog.Group("program",
		slog.Int("pid", os.Getpid()),
		slog.String("machine", hostname),
		slog.String("version", buildInfo.Main.Version),
	)
}

func Error(err error) slog.Attr {
	return slog.Any("error", err)
}
