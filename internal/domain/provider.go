package domain

import (
	"prompter/internal/domain/demo"
	"prompter/internal/domain/prompt"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	demo.ProviderSet,
	prompt.ProviderSet,
	NewServiceHub,
)
