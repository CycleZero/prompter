package internal

import (
	"prompter/internal/domain"
	"prompter/internal/router"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	domain.ProviderSet,
	router.RouterProviderSet,
)
