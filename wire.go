//go:build wireinject
// +build wireinject

package main

import (
	"prompter/infra"
	"prompter/internal"
	"prompter/log"

	"github.com/google/wire"
	"github.com/spf13/viper"
)

func initApp(vc *viper.Viper, logger *log.Logger) (*MainApp, error) {
	wire.Build(
		NewMainApp,
		infra.ProviderSet,
		internal.ProviderSet,
	)
	return nil, nil
}
