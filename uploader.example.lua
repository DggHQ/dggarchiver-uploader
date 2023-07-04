local http = require("http")
local json = require("json")
local inspect = require("inspect")

ReceiveResponse = {}
FinishResponse = {}

local client = http.client()

local pushoverToken = "xxx"
local pushoverUser = "xxx"

local char_to_hex = function(c)
	return string.format("%%%02X", string.byte(c))
end

local function urlencode(url)
	if url == nil then
		return
	end
	url = url:gsub("\n", "\r\n")
	url = url:gsub("([^%w ])", char_to_hex)
	url = url:gsub(" ", "+")
	return url
end

function OnReceive(vod)
	local vodTable = {
		Platform = vod.Platform,
		Downloader = vod.Downloader,
		ID = vod.ID,
		PlaybackURL = vod.PlaybackURL,
		PubTime = vod.PubTime,
		Title = vod.Title,
		StartTime = vod.StartTime,
		EndTime = vod.EndTime,
		Thumbnail = vod.Thumbnail,
		ThumbnailPath = vod.ThumbnailPath,
		Path = vod.Path,
		Duration = vod.Duration,
	}
	local pushoverJSON, err = json.encode(vodTable)
	if err then
		ReceiveResponse.filled = true
		ReceiveResponse.error = true
		ReceiveResponse.message = err
		return
	end
	local urlParams = string.format("?token=%s&user=%s&message=%s&priority=-2&title=%s", pushoverToken
		, pushoverUser, urlencode(pushoverJSON), urlencode("Starting to upload VOD..."))
	local request = http.request("POST", "https://api.pushover.net/1/messages.json" .. urlParams)
	local result, err = client:do_request(request)
	if err then
		ReceiveResponse.filled = true
		ReceiveResponse.error = true
		ReceiveResponse.message = err
		return
	end
	if not (result.code == 200) then
		ReceiveResponse.filled = true
		ReceiveResponse.error = true
		ReceiveResponse.message = tostring(inspect(result))
		return
	end
	ReceiveResponse.filled = true
	ReceiveResponse.error = false
	ReceiveResponse.message = "Sent a notification to Pushover successfully"
end

function OnFinish(vod, success)
	local vodTable = {
		Platform = vod.Platform,
		Downloader = vod.Downloader,
		ID = vod.ID,
		PlaybackURL = vod.PlaybackURL,
		PubTime = vod.PubTime,
		Title = vod.Title,
		StartTime = vod.StartTime,
		EndTime = vod.EndTime,
		Thumbnail = vod.Thumbnail,
		ThumbnailPath = vod.ThumbnailPath,
		Path = vod.Path,
		Duration = vod.Duration,
	}
	local pushoverJSON, err = json.encode(vodTable)
	if err then
		FinishResponse.filled = true
		FinishResponse.error = true
		FinishResponse.message = err
		return
	end
	local urlParams = string.format("?token=%s&user=%s&message=%s&priority=-2", pushoverToken, pushoverUser, urlencode(pushoverJSON))
	if success then
		urlParams = urlParams .. string.format("&title=%s", urlencode("Uploaded VOD to Odysee"))
	else
		urlParams = urlParams .. string.format("&title=%s", urlencode("Wasn't able to upload VOD to Odysee"))
	end
	local request = http.request("POST", "https://api.pushover.net/1/messages.json" .. urlParams)
	local result, err = client:do_request(request)
	if err then
		FinishResponse.filled = true
		FinishResponse.error = true
		FinishResponse.message = err
		return
	end
	if not (result.code == 200) then
		FinishResponse.filled = true
		FinishResponse.error = true
		FinishResponse.message = tostring(inspect(result))
		return
	end
	FinishResponse.filled = true
	FinishResponse.error = false
	FinishResponse.message = "Sent a notification to Pushover successfully"
end