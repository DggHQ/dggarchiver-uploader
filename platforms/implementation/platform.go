package implementation

import (
	"context"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	lua "github.com/yuin/gopher-lua"
)

type newPlatformFunc func(*config.Config, *monitoring.Monitor) (Platform, error)

var Map = map[string]newPlatformFunc{}

type Platform interface {
	Upload(context.Context, *dggarchivermodel.VOD, *lua.LState) error
}
