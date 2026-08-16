package lifeplancmd

import (
	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/breakevencmd"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/comparecmd"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/importmfcmd"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/resolvecmd"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/tablescmd"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/validatecmd"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/wizardcmd"
	"github.com/Kuniwak/lifeplan/version"
)

const Summary = "Life plan simulator. Turns normalised TSV tables into a projection of assets."

func Subcommands() []cli.Subcommand {
	return []cli.Subcommand{
		{Name: "resolve", Summary: resolvecmd.Summary, Run: resolvecmd.NewCommandFunc()},
		{Name: "validate", Summary: validatecmd.Summary, Run: validatecmd.NewCommandFunc()},
		{Name: "tables", Summary: tablescmd.Summary, Run: tablescmd.NewCommandFunc()},
		{Name: "breakeven", Summary: breakevencmd.Summary, Run: breakevencmd.NewCommandFunc()},
		{Name: "compare", Summary: comparecmd.Summary, Run: comparecmd.NewCommandFunc()},
		{Name: "wizard", Summary: wizardcmd.Summary, Run: wizardcmd.NewCommandFunc()},
		{Name: "import-mf-balance", Summary: importmfcmd.BalanceSummary, Run: importmfcmd.NewBalanceCommandFunc()},
		{Name: "import-mf-cashflow", Summary: importmfcmd.CashflowSummary, Run: importmfcmd.NewCashflowCommandFunc()},
	}
}

func NewCommandFunc() cli.CommandFunc {
	return cli.NewDispatchFunc("lifeplan", Summary, version.Version, Subcommands())
}
