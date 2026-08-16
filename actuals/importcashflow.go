package actuals

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
)

type CashflowFile struct {
	Name    string
	Content []byte
}

type CashflowImport struct {
	Cashflow *tsv.Table

	Excluded *tsv.Table

	Payees *tsv.Table

	Sources []Source

	Untrusted *tsv.Table
}

func ImportCashflowFiles(files []CashflowFile, categories ImportRules, excluded, payees *tsv.Table) (CashflowImport, error) {
	var empty CashflowImport

	tables := make([]*tsv.Table, 0, len(files))
	used := make(Used, 16)
	sources := make([]Source, 0, len(files))
	readers := make([]io.Reader, 0, len(files))

	var records []exportRecord

	for _, file := range files {
		read, err := readExportRecords(bytes.NewReader(file.Content))
		if err != nil {
			return empty, fmt.Errorf("actuals.ImportCashflowFiles: %s: %w", file.Name, err)
		}
		records = append(records, read...)

		table, matched, err := ImportMoneyForwardCashflow(bytes.NewReader(file.Content), categories)
		if err != nil {
			return empty, fmt.Errorf("actuals.ImportCashflowFiles: %s: %w", file.Name, err)
		}
		for rule := range matched {
			used[rule] = true
		}
		tables = append(tables, table)

		count, err := CountExportRecords(bytes.NewReader(file.Content))
		if err != nil {
			return empty, fmt.Errorf("actuals.ImportCashflowFiles: %s: %w", file.Name, err)
		}
		sources = append(sources, Source{
			Name:  file.Name,
			Bytes: len(file.Content),
			Hash:  fmt.Sprintf("%x", sha256.Sum256(file.Content)),
			Rows:  count,
		})

		readers = append(readers, bytes.NewReader(file.Content))
	}

	counted, err := CountExcluded(readers, categories, excluded)
	if err != nil {
		return empty, fmt.Errorf("actuals.ImportCashflowFiles: %w", err)
	}

	replay := make([]io.Reader, 0, len(files))
	for _, file := range files {
		replay = append(replay, bytes.NewReader(file.Content))
	}
	countedPayees, err := CountPayees(replay, categories, payees)
	if err != nil {
		return empty, fmt.Errorf("actuals.ImportCashflowFiles: %w", err)
	}

	seen := make(map[string]bool, 32)
	for _, record := range records {
		seen[record.account] = true
	}
	var missing []string
	for _, account := range categories.held.Accounts() {
		if !seen[account] {
			missing = append(missing, account)
		}
	}
	if len(missing) > 0 {
		return empty, fmt.Errorf(
			"actuals.ImportCashflowFiles: 次の口座は書き出しの 保有金融機関 に 1 度も現れない。綴り違いか、消えた口座である:\n  %s",
			strings.Join(missing, "\n  "))
	}

	spent, err := categories.spends(records, used)
	if err != nil {
		return empty, fmt.Errorf("actuals.ImportCashflowFiles: %w", err)
	}

	if unused := categories.Unused(used); len(unused) > 0 {
		return empty, fmt.Errorf(
			"actuals.ImportCashflowFiles: 次の規則はどの行にも当たらなかった。書き出しが変わって死んだ行か、書き間違いである:\n  %s",
			strings.Join(unused, "\n  "))
	}

	tables = append(tables, spent)

	merged, err := MergeCashflow(tables)
	if err != nil {
		return empty, fmt.Errorf("actuals.ImportCashflowFiles: %w", err)
	}

	if len(merged.Rows) == 0 {
		return empty, fmt.Errorf("actuals.ImportCashflowFiles: 1 行も数えられなかった")
	}

	return CashflowImport{
		Cashflow: merged, Excluded: counted, Payees: countedPayees, Sources: sources,
		Untrusted: categories.gaps(records),
	}, nil
}
