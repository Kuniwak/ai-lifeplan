package tools

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/slograw"
	"github.com/Kuniwak/lifeplan/tsv"
)

type CommonOptions struct {
	Help     bool
	Version  bool
	LogLevel slog.Level
}

var CommonOptionsHelp = &CommonOptions{Help: true}
var CommonOptionsVersion = &CommonOptions{Version: true}

func NewCommonOptionsDefault() *CommonOptions {
	return &CommonOptions{
		LogLevel: slog.LevelInfo,
	}
}

type CommonRawOptions struct {
	Help         bool
	ShortVersion bool
	Version      bool
	Silent       bool
	Debug        bool
}

func DeclareCommonOptions(flags *flag.FlagSet, options *CommonRawOptions) {
	flags.BoolVar(&options.ShortVersion, "v", false, "show version")
	flags.BoolVar(&options.Version, "version", false, "show version")
	flags.BoolVar(&options.Silent, "silent", false, "silent mode")
	flags.BoolVar(&options.Debug, "debug", false, "debug mode")
}

func ValidateCommonOptions(options *CommonRawOptions) (*CommonOptions, error) {
	if options.ShortVersion || options.Version {
		return &CommonOptions{Version: true}, nil
	}

	opts := NewCommonOptionsDefault()
	if options.Debug {
		opts.LogLevel = slog.LevelDebug
	} else if options.Silent {
		opts.LogLevel = slog.LevelError
	}

	return opts, nil
}

func NewLogger(logLevel slog.Level, w io.Writer) *slog.Logger {
	return slog.New(slograw.NewHandler(w, logLevel))
}

func WarnUnreadColumns(log *slog.Logger, unread map[tsv.Slot][]tsv.ColumnName) {
	for _, slot := range slices.Sorted(maps.Keys(unread)) {
		for _, column := range unread[slot] {
			log.Warn("この列は誰も読んでいない。書かれたまま残る",
				"slot", slot, "列", column)
		}
	}
}

func DebugInput(log *slog.Logger, root, projectPath string) {
	log.Debug("入力を読む", "root", root, "project", projectPath)
}

const OutRoot = "out"

func AssertUnderOut(dir string) error {
	for _, element := range strings.Split(filepath.ToSlash(filepath.Clean(dir)), "/") {
		if element == OutRoot {
			return nil
		}
	}
	return fmt.Errorf(
		"%q is not under a directory called %s, and the tables are only ever written under one so that no run can overwrite its own input",
		dir, OutRoot)
}

type SlotOverrideFlag []string

func (f *SlotOverrideFlag) String() string { return fmt.Sprint([]string(*f)) }

func (f *SlotOverrideFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func DeclareSlotOverride(flags *flag.FlagSet, into *SlotOverrideFlag) {
	flags.Var(into, "slot-override", "override a slot as name=path; may be repeated")
}
