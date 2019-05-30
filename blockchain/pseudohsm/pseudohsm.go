// Package pseudohsm provides a pseudo HSM for development environments.
package pseudohsm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/pborman/uuid"

	vcrypto "github.com/vapor/crypto"
	"github.com/vapor/crypto/csp"
	edchainkd "github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	mnem "github.com/vapor/wallet/mnemonic"
)

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateKeyAlias = errors.New("duplicate key alias")
	ErrXPubFormat        = errors.New("xpub format error")
	ErrLoadKey           = errors.New("key not found or wrong password ")
	ErrDecrypt           = errors.New("could not decrypt key with given passphrase")
	ErrMnemonicLength    = errors.New("mnemonic length error")
)

// EntropyLength random entropy length to generate mnemonics.
const EntropyLength = 128

// HSM type for storing pubkey and privatekey
type HSM struct {
	cacheMu  sync.Mutex
	keyStore keyStore
	cache    *keyCache
	//kdCache  map[chainkd.XPub]chainkd.XPrv
}

// XPub type for pubkey for anyone can see
type XPub struct {
	Alias string            `json:"alias"`
	XPub  vcrypto.XPubKeyer `json:"xpub"`
	File  string            `json:"file"`
}

// New method for HSM struct
func New(keypath string) (*HSM, error) {
	keydir, _ := filepath.Abs(keypath)
	return &HSM{
		keyStore: &keyStorePassphrase{keydir, LightScryptN, LightScryptP},
		cache:    newKeyCache(keydir),
		//kdCache:  make(map[chainkd.XPub]chainkd.XPrv),
	}, nil
}

// XCreate produces a new random xprv and stores it in the db.
func (h *HSM) XCreate(alias string, auth string, language string) (*XPub, *string, error) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	fmt.Println("XCreate start h.cache:", h.cache)

	normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
	if ok := h.cache.hasAlias(normalizedAlias); ok {
		return nil, nil, ErrDuplicateKeyAlias
	}

	xpub, mnemonic, err := h.createChainKDKey(normalizedAlias, auth, language)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("XCreate ...xpub:", xpub)
	fmt.Println("XCreate ...xpub type:", reflect.TypeOf(xpub.XPub).String())
	h.cache.add(*xpub)

	fmt.Println("XCreate after add h.cache:", h.cache)
	for i, v := range h.cache.byPubs {
		fmt.Println("XCreate i:", i)
		fmt.Println("XCreate i type:", reflect.TypeOf(i))
		fmt.Println("XCreate v:", v)
		fmt.Println("XCreate v type:", reflect.TypeOf(v))
	}

	fmt.Println("----XCreate end------")
	return xpub, mnemonic, err
}

// ImportKeyFromMnemonic produces a xprv from mnemonic and stores it in the db.
func (h *HSM) ImportKeyFromMnemonic(alias string, auth string, mnemonic string, language string) (*XPub, error) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	// checksum length = entropy length /32
	// mnemonic length = (entropy length + checksum length)/11
	if len(strings.Fields(mnemonic)) != (EntropyLength+EntropyLength/32)/11 {
		return nil, ErrMnemonicLength
	}

	normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
	if ok := h.cache.hasAlias(normalizedAlias); ok {
		return nil, ErrDuplicateKeyAlias
	}

	// Pre validate that the mnemonic is well formed and only contains words that
	// are present in the word list
	if !mnem.IsMnemonicValid(mnemonic, language) {
		return nil, mnem.ErrInvalidMnemonic
	}

	xpub, err := h.createKeyFromMnemonic(alias, auth, mnemonic)
	if err != nil {
		return nil, err
	}

	h.cache.add(*xpub)
	return xpub, nil
}

func (h *HSM) createKeyFromMnemonic(alias string, auth string, mnemonic string) (*XPub, error) {
	// Generate a Bip32 HD wallet for the mnemonic and a user supplied password
	seed := mnem.NewSeed(mnemonic, "")
	xprv, xpub, err := csp.NewXKeys(bytes.NewBuffer(seed))
	fmt.Println("createKeyFromMnemonic xprv:", xprv, "xpub:", xpub)
	fmt.Println("createKeyFromMnemonic xpub type:", reflect.TypeOf(xpub))
	if err != nil {
		return nil, err
	}
	id := uuid.NewRandom()
	key := &XKey{
		ID:      id,
		KeyType: "bytom_kd",
		XPub:    xpub,
		XPrv:    xprv,
		Alias:   alias,
	}
	file := h.keyStore.JoinPath(keyFileName(key.ID.String()))
	if err := h.keyStore.StoreKey(file, key, auth); err != nil {
		return nil, errors.Wrap(err, "storing keys")
	}
	fmt.Println("createKeyFromMnemonic new XPub:", &XPub{XPub: xpub, Alias: alias, File: file})
	return &XPub{XPub: xpub, Alias: alias, File: file}, nil
}

func (h *HSM) createChainKDKey(alias string, auth string, language string) (*XPub, *string, error) {
	// Generate a mnemonic for memorization or user-friendly seeds
	entropy, err := mnem.NewEntropy(EntropyLength)
	if err != nil {
		return nil, nil, err
	}
	mnemonic, err := mnem.NewMnemonic(entropy, language)
	if err != nil {
		return nil, nil, err
	}
	xpub, err := h.createKeyFromMnemonic(alias, auth, mnemonic)
	if err != nil {
		return nil, nil, err
	}
	return xpub, &mnemonic, nil
}

// UpdateKeyAlias update key alias
func (h *HSM) UpdateKeyAlias(xpub vcrypto.XPubKeyer, newAlias string) error {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	h.cache.maybeReload()
	h.cache.mu.Lock()
	xpb, err := h.cache.find(XPub{XPub: xpub})
	h.cache.mu.Unlock()
	if err != nil {
		return err
	}

	keyjson, err := ioutil.ReadFile(xpb.File)
	if err != nil {
		return err
	}

	encrptKeyJSON := new(encryptedKeyJSON)
	if err := json.Unmarshal(keyjson, encrptKeyJSON); err != nil {
		return err
	}

	normalizedAlias := strings.ToLower(strings.TrimSpace(newAlias))
	if ok := h.cache.hasAlias(normalizedAlias); ok {
		return ErrDuplicateKeyAlias
	}

	encrptKeyJSON.Alias = normalizedAlias
	keyJSON, err := json.Marshal(encrptKeyJSON)
	if err != nil {
		return err
	}

	if err := writeKeyFile(xpb.File, keyJSON); err != nil {
		return err
	}

	// update key alias
	h.cache.delete(xpb)
	xpb.Alias = normalizedAlias
	h.cache.add(xpb)

	return nil
}

// ListKeys returns a list of all xpubs from the store
func (h *HSM) ListKeys() []XPub {
	xpubs := h.cache.keys()
	return xpubs
}

// XSign looks up the xprv given the xpub, optionally derives a new
// xprv with the given path (but does not store the new xprv), and
// signs the given msg.
func (h *HSM) XSign(xpub vcrypto.XPubKeyer, path [][]byte, msg []byte, auth string) ([]byte, error) {
	fmt.Println("XSign start...")
	xprv, err := h.LoadChainKDKey(xpub, auth)
	if err != nil {
		fmt.Println("some err...")
		return nil, err
	}
	fmt.Println("XSign xprv:", xprv)
	fmt.Println("XSign xprv type:", reflect.TypeOf(xprv))
	fmt.Println("XSign path:", path)
	fmt.Println("XSign len(path):", len(path))
	// if len(path) > 0 {
	// 	switch xprvkey := xprv.(type) {
	// 	case edchainkd.XPrv:
	// 		xprvk := xprvkey.Derive(path)
	// 		sig := xprvk.Sign(msg)
	// 		fmt.Println("XSign sig:", sig)
	// 		return sig, nil
	// 	}
	// }
	fmt.Println("XSign end...")
	switch xprvkey := xprv.(type) {
	case edchainkd.XPrv:
		if len(path) > 0 {
			xprvk := xprvkey.Derive(path)
			return xprvk.Sign(msg), nil
		}
		return xprvkey.Sign(msg), nil
	}
	return nil, nil
}

//LoadChainKDKey get xprv from xpub
func (h *HSM) LoadChainKDKey(xpub vcrypto.XPubKeyer, auth string) (xprv vcrypto.XPrvKeyer, err error) {
	fmt.Println("LoadChainKDKey start...")
	fmt.Println("LoadChainKDKey xpub:", xpub)
	fmt.Println("LoadChainKDKey xpub type:", reflect.TypeOf(xpub))
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()

	//if xprv, ok := h.kdCache[xpub]; ok {
	//	return xprv, nil
	//}

	fmt.Println("LoadChainKDKey h.cache:", h.cache)
	// for i, c := range h.cache.byPubs {
	// 	if reflect.TypeOf(i).String() == "string" {
	// 		if xpb, err := edchainkd.NewXPub(reflect.ValueOf(i).String()); err != nil {
	// 			panic(err)
	// 		} else {
	// 			h.cache.byPubs[*xpb] = c
	// 			delete(h.cache.byPubs, i)
	// 		}
	// 	}
	// 	fmt.Println("LoadChainKDKey i:", i)
	// 	fmt.Println("LoadChainKDKey i type:", reflect.TypeOf(i))
	// 	fmt.Println("LoadChainKDKey c:", c)
	// 	fmt.Println("LoadChainKDKey c type:", reflect.TypeOf(c[0]))
	// }
	_, xkey, err := h.loadDecryptedKey(xpub, auth)
	if err != nil {
		return xprv, ErrLoadKey
	}
	fmt.Println("LoadChainKDKey end...")
	//h.kdCache[xpb.XPub] = xkey.XPrv
	return xkey.XPrv, nil
}

// XDelete deletes the key matched by xpub if the passphrase is correct.
// If a contains no filename, the address must match a unique key.
func (h *HSM) XDelete(xpub vcrypto.XPubKeyer, auth string) error {
	// Decrypting the key isn't really necessary, but we do
	// it anyway to check the password and zero out the key
	// immediately afterwards.

	xpb, xkey, err := h.loadDecryptedKey(xpub, auth)
	if xkey != nil {
		zeroKey(xkey)
	}
	if err != nil {
		return err
	}

	h.cacheMu.Lock()
	// The order is crucial here. The key is dropped from the
	// cache after the file is gone so that a reload happening in
	// between won't insert it into the cache again.
	err = os.Remove(xpb.File)
	if err == nil {
		h.cache.delete(xpb)
	}
	h.cacheMu.Unlock()
	return err
}

func (h *HSM) loadDecryptedKey(xpub vcrypto.XPubKeyer, auth string) (XPub, *XKey, error) {
	fmt.Println("loadDecryptedKey start...")
	fmt.Println("loadDecryptedKey xpub:", xpub)
	fmt.Println("loadDecryptedKey xpub type:", reflect.TypeOf(xpub))
	fmt.Println("loadDecryptedKey h.cache:", h.cache)
	h.cache.maybeReload()
	h.cache.mu.Lock()
	fmt.Println("loadDecryptedKey h.cache:", h.cache)
	fmt.Println("loadDecryptedKey xpub:", xpub)
	fmt.Println("loadDecryptedKey xpub type:", reflect.TypeOf(xpub))
	xpb, err := h.cache.find(XPub{XPub: xpub})
	fmt.Println("loadDecryptedKey xpb:", xpb)
	fmt.Println("loadDecryptedKey find...")

	h.cache.mu.Unlock()
	if err != nil {
		fmt.Println("loadDecryptedKey:err", err)
		return xpb, nil, err
	}
	xkey, err := h.keyStore.GetKey(xpb.Alias, xpb.File, auth)
	fmt.Println("loadDecryptedKey start...")
	return xpb, xkey, err
}

// ResetPassword reset passphrase for an existing xpub
func (h *HSM) ResetPassword(xpub vcrypto.XPubKeyer, oldAuth, newAuth string) error {
	xpb, xkey, err := h.loadDecryptedKey(xpub, oldAuth)
	if err != nil {
		return err
	}
	return h.keyStore.StoreKey(xpb.File, xkey, newAuth)
}

// HasAlias check whether the key alias exists
func (h *HSM) HasAlias(alias string) bool {
	return h.cache.hasAlias(alias)
}

// HasKey check whether the private key exists
func (h *HSM) HasKey(xprv vcrypto.XPrvKeyer) bool {
	switch xprvkey := xprv.(type) {
	case edchainkd.XPrv:
		return h.cache.hasKey(xprvkey.XPub())
	}
	return false
}
