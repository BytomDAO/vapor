package account

import (
	"github.com/bytom/vapor/blockchain/query"
)

//Annotated init an annotated account object
func Annotated(a *Account) *query.AnnotatedAccount {
	return &query.AnnotatedAccount{
		ID:         a.ID,
		Alias:      a.Alias,
		Quorum:     a.Quorum,
		XPubs:      a.XPubs,
		KeyIndex:   a.KeyIndex,
		DeriveRule: a.DeriveRule,
	}
}
