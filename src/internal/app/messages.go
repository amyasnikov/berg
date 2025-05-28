package app



type appMessage int


const (
	stopAppMsg appMessage = iota
	reloadConfigMsg
)
