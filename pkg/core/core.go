package core

import (
	"go.uber.org/fx"

	"github.com/nsxbet/masspr/pkg/mygit"
	"github.com/nsxbet/masspr/pkg/printer"
	"github.com/nsxbet/masspr/pkg/types"
)

type Core struct {
	fx.In

	Config  types.Config
	Git     *mygit.Git
	Printer *printer.ConsolePrinter
}

var Module = fx.Options(
	fx.Provide(
		printer.NewConsolePrinter,
		mygit.NewGit,
	),
)
