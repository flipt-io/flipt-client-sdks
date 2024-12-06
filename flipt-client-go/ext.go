package evaluation

import "embed"

//go:embed ext/*
var _ embed.FS

// https://github.com/flipt-io/flipt-client-sdks/issues/558
// to support vendoring of non-go files
