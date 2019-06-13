package synchron

import (
	vaporCfg "github.com/vapor/config"
	"github.com/vapor/errors"
)

var (
	fedProg = vaporCfg.FederationProgrom(vaporCfg.CommonConfig)

	ErrInconsistentDB = errors.New("inconsistent db status")
)
