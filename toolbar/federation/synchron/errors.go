package synchron

import (
	"github.com/bytom/vapor/errors"
)

var (
	ErrInconsistentDB = errors.New("inconsistent db status")
	ErrOutputType     = errors.New("error output type")
)
