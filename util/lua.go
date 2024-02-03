package util

import (
	"log/slog"

	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

type LuaResponse struct {
	Filled  bool
	Error   bool
	Message string
	Data    map[string]interface{}
}

func LuaCallReceiveFunction(l *lua.LState, vod *dggarchivermodel.VOD) *LuaResponse {
	luaVOD := luar.New(l, vod)

	result := &LuaResponse{}
	l.SetGlobal("ReceiveResponse", luar.New(l, result))

	if err := l.CallByParam(lua.P{
		Fn:      l.GetGlobal("OnReceive"),
		NRet:    0,
		Protect: true,
	}, luaVOD); err != nil {
		slog.Debug("unable to access the \"OnReceive\" function of the Lua script", slog.Any("err", err))
		return nil
	}

	if result.Filled {
		if result.Error {
			slog.Debug("unable to execute the \"OnReceive\" function of the Lua script", slog.Any("err", result.Message))
			return nil
		}
	}

	return result
}

func LuaCallProgressFunction(l *lua.LState, progress int) *LuaResponse {
	luaProgress := luar.New(l, progress)

	result := &LuaResponse{}
	l.SetGlobal("ProgressResponse", luar.New(l, result))

	if err := l.CallByParam(lua.P{
		Fn:      l.GetGlobal("OnProgress"),
		NRet:    0,
		Protect: true,
	}, luaProgress); err != nil {
		slog.Debug("unable to access the \"OnProgress\" function of the Lua script", slog.Any("err", err))
		return nil
	}

	if result.Filled {
		if result.Error {
			slog.Debug("unable to execute the \"OnProgress\" function of the Lua script", slog.Any("err", result.Message))
			return nil
		}
	}

	return result
}

func LuaCallFinishFunction(l *lua.LState, vod *dggarchivermodel.VOD, success bool) *LuaResponse {
	luaVOD := luar.New(l, vod)
	luaSuccess := luar.New(l, success)

	result := &LuaResponse{}
	l.SetGlobal("FinishResponse", luar.New(l, result))

	if err := l.CallByParam(lua.P{
		Fn:      l.GetGlobal("OnFinish"),
		NRet:    0,
		Protect: true,
	}, luaVOD, luaSuccess); err != nil {
		slog.Debug("unable to access the \"OnFinish\" function of the Lua script", slog.Any("err", err))
		return nil
	}

	if result.Filled {
		if result.Error {
			slog.Debug("unable to execute the \"OnFinish\" function of the Lua script", slog.Any("err", result.Message))
			return nil
		}
	}

	return result
}

func LuaCallInsertFunction(l *lua.LState, vod *dggarchivermodel.VOD, success bool) *LuaResponse {
	luaVOD := luar.New(l, vod)
	luaSuccess := luar.New(l, success)

	result := &LuaResponse{}
	l.SetGlobal("InsertResponse", luar.New(l, result))

	if err := l.CallByParam(lua.P{
		Fn:      l.GetGlobal("OnInsert"),
		NRet:    0,
		Protect: true,
	}, luaVOD, luaSuccess); err != nil {
		slog.Debug("unable to access the \"OnInsert\" function of the Lua script", slog.Any("err", err))
		return nil
	}

	if result.Filled {
		if result.Error {
			slog.Debug("unable to execute the \"OnInsert\" function of the Lua script", slog.Any("err", result.Message))
			return nil
		}
	}

	return result
}
