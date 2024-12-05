package ext

import "embed"

//go:embed *.h
var _ embed.FS

// https://github.com/flipt-io/flipt-client-sdks/issues/558
// to support vendoring of non-go files
