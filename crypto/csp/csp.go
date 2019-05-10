// csp is a package of cipher service provider

package csp

type (
	//XPrv external private key
	XPrv [64]byte
	//XPub external public key
	XPub [64]byte

	// Sm2XPrv external sm2 private key
	Sm2XPrv [64]byte
	// Sm2XPub external sm2 public key
	Sm2XPub [65]byte
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
	// PublicKey extracts the ed25519 public key from an xpub.
	PublicKey() interface{}
	// Derive generates a child xpub by recursively deriving
	// non-hardened child xpubs over the list of selectors:
	// `Derive([a,b,c,...]) == Child(a).Child(b).Child(c)...`
	Derive(path [][]byte) XPubKeyer
	// Verify checks an EdDSA signature using public key
	// extracted from the first 32 bytes of the xpub.
	Verify(msg []byte, sig []byte) bool
}
