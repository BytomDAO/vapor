package api

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// TODO:
func RespondErrorResp(c *gin.Context, err error) {
	log.WithFields(log.Fields{
		"url": c.Request.URL,
		// "request": c.Value(ReqBodyLabel),
		"err": err,
	}).Error("request fail")
	// resp := formatErrResp(err)
	// c.AbortWithStatusJSON(http.StatusOK, resp)
}

func RespondSuccessResp(c *gin.Context, data interface{}) {
	// result := make(map[string]interface{})
	// result["data"] = data
	// c.AbortWithStatusJSON(http.StatusOK, Response{Code: 200, Result: result})
}
