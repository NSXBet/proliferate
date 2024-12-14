package core

import (
	"go.uber.org/fx"

	"github.com/nsxbet/proliferate/pkg/mygit"
	"github.com/nsxbet/proliferate/pkg/printer"
	"github.com/nsxbet/proliferate/pkg/types"
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
