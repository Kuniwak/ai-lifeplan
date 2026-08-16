package tools_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestWarnUnreadColumnsShouldNameEveryColumnNothingReads(t *testing.T) {
	var out bytes.Buffer
	log := tools.NewLogger(slog.LevelInfo, &out)

	tools.WarnUnreadColumns(log, map[tsv.Slot][]tsv.ColumnName{
		"inflation": {"インフレ立", "メモ"},
		"residence": {"備考"},
	})

	for _, want := range []string{"インフレ立", "メモ", "備考", "inflation", "residence"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the warning does not mention %q: %s", want, out.String())
		}
	}
}

func TestSilentShouldKeepTheWarningsBack(t *testing.T) {
	var out bytes.Buffer
	log := tools.NewLogger(slog.LevelError, &out)

	tools.WarnUnreadColumns(log, map[tsv.Slot][]tsv.ColumnName{"inflation": {"メモ"}})

	if out.Len() != 0 {
		t.Errorf("want nothing written, got %q", out.String())
	}
}

func TestDebugInputShouldBeKeptBackUnlessAsked(t *testing.T) {
	var quiet, asked bytes.Buffer

	tools.DebugInput(tools.NewLogger(slog.LevelInfo, &quiet), ".", "projects/base.tsv")
	tools.DebugInput(tools.NewLogger(slog.LevelDebug, &asked), ".", "projects/base.tsv")

	if quiet.Len() != 0 {
		t.Errorf("want nothing at the default level, got %q", quiet.String())
	}
	if !strings.Contains(asked.String(), "projects/base.tsv") {
		t.Errorf("the debug line does not name the project: %q", asked.String())
	}
}
