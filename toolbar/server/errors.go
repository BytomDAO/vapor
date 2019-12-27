package server

import (
	"github.com/bytom/vapor/errors"
)

//FormatErrResp format error response
func formatErrResp(err error) response {
	// default error response
	response := response{
		Code: 300,
		Msg:  "request error",
	}

	root := errors.Root(err)
	if errCode, ok := respErrFormatter[root]; ok {
		response.Code = errCode
		response.Msg = root.Error()
	}
	return response
}

var respErrFormatter = map[error]int{}
