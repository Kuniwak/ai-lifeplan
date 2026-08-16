package wizard_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/wizard"
)

func theBaseProject(t *testing.T) *plan.Input {
	t.Helper()

	in, err := plan.Load(plan.Sources{ProjectPath: "../projects/base.tsv", Root: ".."})
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	return in
}

func TestEveryQuestionShouldReachTheProjectsOwnTables(t *testing.T) {
	in := theBaseProject(t)

	questions, err := wizard.Questions(in)
	if err != nil {
		t.Fatalf("wizard.Questions: %v", err)
	}
	if len(questions) == 0 {
		t.Fatal("問いが 1 つも無い")
	}

	for _, q := range questions {
		t.Run(q.Name, func(t *testing.T) {
			if len(q.Choices) < 2 {
				t.Errorf("選択肢が %d 個しかない。選ばせるなら 2 つ以上要る", len(q.Choices))
			}
			if q.Why == "" {
				t.Error("なぜ訊くのかが書かれていない")
			}
			if q.AnsweredByTable() {
				if wizard.WrittenPath(in, q) == "" {
					t.Error("いまどの表が入っているのか読めない")
				}
				return
			}
			if _, err := wizard.Written(in, q); err != nil {
				t.Errorf("いまの設定を読めない: %v", err)
			}
		})
	}
}

func TestAskShouldWorkOutEveryChoiceAgainstTheWholePlan(t *testing.T) {
	in := theBaseProject(t)
	questions, err := wizard.Questions(in)
	if err != nil {
		t.Fatalf("wizard.Questions: %v", err)
	}

	var living wizard.Question
	for _, q := range questions {
		if strings.Contains(q.Name, "生活費") {
			living = q
		}
	}
	if living.Name == "" {
		t.Fatal("生活費の問いが無い")
	}

	answered, err := wizard.Ask(in, living)
	if err != nil {
		t.Fatalf("wizard.Ask: %v", err)
	}

	if len(answered) != len(living.Choices) {
		t.Fatalf("選択肢 %d 個に対し結末が %d 個", len(living.Choices), len(answered))
	}
	for i := 1; i < len(answered); i++ {
		before, after := answered[i-1], answered[i]
		if before.Setting.Cmp(after.Setting) >= 0 {
			t.Fatalf("選択肢が昇順でない: %v, %v", before.Setting, after.Setting)
		}
		if before.Final < after.Final {
			t.Errorf("生活費 %v で %d、%v で %d。上げたのに増えている",
				before.Setting, before.Final, after.Setting, after.Final)
		}
	}
}

func TestWriteShouldLeaveAProjectThatBuilds(t *testing.T) {
	in := theBaseProject(t)
	questions, err := wizard.Questions(in)
	if err != nil {
		t.Fatalf("wizard.Questions: %v", err)
	}

	answers := make([]wizard.Answer, 0, 1)
	for _, q := range questions {
		if strings.Contains(q.Name, "生活費") {
			answers = append(answers, wizard.Answer{Question: q, Setting: q.Choices[len(q.Choices)-1].Setting})
		}
	}

	dir := filepath.Join("out", "wizard-test")
	t.Cleanup(func() { os.RemoveAll(filepath.Join("..", dir)) })

	written, err := wizard.Write(wizard.Request{
		Input:   in,
		Base:    filepath.Join("..", "..", "projects", "base.tsv"),
		Root:    "..",
		Dir:     dir,
		Name:    "wizard-test",
		Answers: answers,
	})
	if err != nil {
		t.Fatalf("wizard.Write: %v", err)
	}

	if _, err := plan.Load(plan.Sources{ProjectPath: written, Root: ".."}); err != nil {
		t.Fatalf("書いた project から計画を組めない: %v", err)
	}

	manifest, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if !strings.Contains(string(manifest), "extends") {
		t.Error("extends が書かれていない。答えなかったものは base から継ぐはず")
	}
}
