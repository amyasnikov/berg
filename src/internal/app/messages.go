package app

import "github.com/amyasnikov/berg/internal/dto"



type msgCode int


const (
	stopAppMsg msgCode = iota
	reloadConfigMsg
)

type message struct {
	Code msgCode
	VrfDiff *dto.VrfDiff
}
