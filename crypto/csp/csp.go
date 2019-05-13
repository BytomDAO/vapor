// csp is a package of cipher service provider

package csp

import (
	"crypto"
)

type XPrvKeyer interface {
	// XPub derives an extended public key from a given xprv.
	XPub() XPubKeyer
	// Derive generates a child xprv by recursively deriving
	// non-hardened child xprvs over the list of selectors:
	// `Derive([a,b,c,...]) == Child(a).Child(b).Child(c)...`
	Derive(path [][]byte) XPrvKeyer
	// Sign creates an EdDSA signature using expanded private key
	// derived from the xprv.
	Sign(msg []byte) []byte
}

type XPubKeyer interface {
	// PublicKey extracts the public key from an xpub.
	PublicKey() crypto.PublicKey
	// Derive generates a child xpub by recursively deriving
	// non-hardened child xpubs over the list of selectors:
	// `Derive([a,b,c,...]) == Child(a).Child(b).Child(c)...`
	Derive(path [][]byte) XPubKeyer
	// Verify checks an EdDSA signature using public key
	Verify(msg []byte, sig []byte) bool
}
