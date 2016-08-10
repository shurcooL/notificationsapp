package common

type State struct {
	BaseURI string
	ReqPath string
}

type MarkReadRequest struct {
	AppID    string
	RepoURI  string
	ThreadID uint64
}
type MarkAllReadRequest struct {
	RepoURI string
}
