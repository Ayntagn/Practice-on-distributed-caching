package cacheClient

type Cmd struct {
	Name  string
	Key   string
	Value string
	Error error
}

type Client interface {
	Run(*Cmd)
	PipelinedRun([]*Cmd)
}

func New(typ, server string) Client {
	switch typ {
	case "redis":
		return newRedisClient(server)
	case "http":
		return newHTTPClient(server)
	case "tcp":
		return newTCPClient(server)
	default:
		panic("unknown client type " + typ)
	}
}
