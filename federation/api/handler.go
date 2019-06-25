package api

import (
	"github.com/gin-gonic/gin"
)

func (s *Server) ListCrosschainTxs(c *gin.Context, req *listCrosschainTxsReq, query *PaginationQuery) ([]*crosschainTx, error) {
	return nil, nil
}
