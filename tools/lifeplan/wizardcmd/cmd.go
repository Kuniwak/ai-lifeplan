package wizardcmd

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Kuniwak/lifeplan/breakeven"
	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/version"
	"github.com/Kuniwak/lifeplan/wizard"
)

const Summary = "ask the household what to assume, and write a project from the answers"

type Options struct {
	Common *tools.CommonOptions

	Base, Root string

	Dir, Name string
}

func NewParseOptionsFunc() cli.ParseOptionsFunc[*Options] {
	return func(args []string, inout *cli.ProcInout) (*Options, error) {
		opts := &Options{}
		var raw tools.CommonRawOptions

		fs := flag.NewFlagSet("wizard", flag.ContinueOnError)
		fs.SetOutput(inout.Stderr)
		fs.StringVar(&opts.Base, "base", "projects/base.tsv", "the manifest the written one extends")
		fs.StringVar(&opts.Root, "root", ".", "where the paths in the manifest are taken from")
		fs.StringVar(&opts.Dir, "dir", "projects", "where to write, relative to -root")
		fs.StringVar(&opts.Name, "name", "", "what to call the written project")
		tools.DeclareCommonOptions(fs, &raw)
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return &Options{Common: tools.CommonOptionsHelp}, nil
			}
			return nil, err
		}
		common, err := tools.ValidateCommonOptions(&raw)
		if err != nil {
			return nil, err
		}
		opts.Common = common
		if opts.Common.Help || opts.Common.Version {
			return opts, nil
		}
		if opts.Name == "" {
			return nil, fmt.Errorf("-name が要る。書き出す project の名前である")
		}
		return opts, nil
	}
}

func NewMainFunc() cli.MainFunc[*Options] {
	return func(opts *Options, inout *cli.ProcInout) error {
		if opts.Common.Help {
			fmt.Fprintf(inout.Stdout, "%s\n", Summary)
			return nil
		}
		if opts.Common.Version {
			fmt.Fprintln(inout.Stdout, version.Version)
			return nil
		}

		in, err := plan.Load(plan.Sources{Root: opts.Root, ProjectPath: filepath.Join(opts.Root, opts.Base)})
		if err != nil {
			return fmt.Errorf("wizard: %w", err)
		}
		questions, err := wizard.Questions(in)
		if err != nil {
			return err
		}

		read := bufio.NewScanner(inout.Stdin)
		answers := make([]wizard.Answer, 0, len(questions))
		for i, q := range questions {
			outcomes, err := wizard.Ask(in, q)
			if err != nil {
				return err
			}
			var written breakeven.Setting
			if !q.AnsweredByTable() {
				if written, err = wizard.Written(in, q); err != nil {
					return err
				}
			}
			inPlace := wizard.WrittenPath(in, q)

			fmt.Fprintf(inout.Stdout, "\n[%d/%d] %s\n%s\n\n", i+1, len(questions), q.Name, q.Why)
			for n, c := range q.Choices {
				mark := inForce(c.Setting, written)
				if q.AnsweredByTable() {
					mark = ""
					if c.Path == inPlace {
						mark = "  ← いまの計画"
					}
				}
				fmt.Fprintf(inout.Stdout, "  %d) %-22s %s%s\n",
					n+1, c.Label, describe(outcomes[n]), mark)
			}

			chosen, err := choose(inout, read, len(q.Choices))
			if err != nil {
				return err
			}
			answers = append(answers, wizard.Answer{
				Question: q,
				Setting:  q.Choices[chosen].Setting,
				Choice:   q.Choices[chosen],
			})
		}

		path, err := wizard.Write(wizard.Request{
			Input: in, Base: baseFrom(opts), Root: opts.Root,
			Dir: opts.Dir, Name: opts.Name, Answers: answers,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(inout.Stdout, "\n%s を書いた。`make` で表が出る。\n", path)
		return nil
	}
}

func baseFrom(opts *Options) string {
	if strings.HasPrefix(opts.Base, opts.Dir+"/") {
		return strings.TrimPrefix(opts.Base, opts.Dir+"/")
	}
	return opts.Base
}

func describe(o breakeven.Outcome) string {
	if o.ShortFrom != 0 {
		return fmt.Sprintf("%d 年に尽きる（不足 %s 円）", o.ShortFrom, o.Shortfall)
	}
	return fmt.Sprintf("%d 年に %s 円", o.LastYear, o.Final)
}

func inForce(choice, written breakeven.Setting) string {
	if choice.Field() == written.Field() {
		return "  ← いまの計画"
	}
	return ""
}

func choose(inout *cli.ProcInout, read *bufio.Scanner, n int) (int, error) {
	for {
		fmt.Fprintf(inout.Stdout, "\n1〜%d を選んでください: ", n)
		if !read.Scan() {
			if err := read.Err(); err != nil {
				return 0, fmt.Errorf("wizard: %w", err)
			}
			return 0, fmt.Errorf("wizard: 答えのないまま入力が終わった")
		}
		chosen, err := strconv.Atoi(strings.TrimSpace(read.Text()))
		if err != nil || chosen < 1 || chosen > n {
			fmt.Fprintf(inout.Stdout, "1〜%d の数で答えてください。\n", n)
			continue
		}
		return chosen - 1, nil
	}
}

func NewCommandFunc() cli.CommandFunc {
	return cli.NewCommandFunc(NewParseOptionsFunc(), NewMainFunc())
}
