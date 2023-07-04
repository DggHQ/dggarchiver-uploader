# dggarchiver-uploader
This is the uploader service of the dggarchiver service that uploads the recorded livestreams.

## Features

1. Supported video platforms:
   - LBRY/Odysee
2. Adds the uploaded livestreams to an SQLite database
3. Broadcasts current livestream upload progress via Prometheus
4. Lua plugin support

## Lua

The service can be extended with Lua plugins/scripts. An example can be found in the ```uploader.example.lua``` file.

If enabled, the service will call these functions from the specified ```.lua``` file:
- ```OnReceive(vod)``` when a livestream recording is received, where ```vod``` is the livestream struct
- ```OnProgress(progressPercentage)``` when upload progress is updated, where ```progressPercentage``` is the percentage number
- ```OnFinish(vod, success)``` when a livestream recording is uploaded, where ```vod``` is the livestream struct, and ```success``` is the boolean signifying success
- ```OnInsert(vod, success)``` when a livestream recording has been added to the SQLite database, where ```vod``` is the livestream struct, and ```success``` is the boolean signifying success

After the functions are done executing, the service will check the global ```ReceiveResponse```, ```ProgressResponse```, ```FinishResponse``` and ```InsertResponse``` variables for errors, before returning the struct. The struct's fields are:
```go
type LuaResponse struct {
	Filled  bool
	Error   bool
	Message string
	Data    map[string]interface{}
}
```

## Configuration

The config file location can be set with the ```CONFIG``` environment variable. Example configuration can be found below and in the ```config.example.yaml``` file.

```yaml
uploader:
  lbry:
    uri: https://example.com/ # lbry-sdk server uri
    author: example # author name to set for livestreams
    channel_name: example # channel to upload the livestreams to
  sqlite: 
    uri: example.sqlite # path to the SQLite database
  plugins:
    enabled: no
    path: uploader.lua # path to the lua plugin
  verbose: no # increases log verbosity

nats:
  host: nats # nats uri
  topic: archiver # main nats topic
```