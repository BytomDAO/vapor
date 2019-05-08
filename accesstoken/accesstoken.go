// Package accesstoken provides storage and validation of Chain Core
// credentials.
package accesstoken

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/vapor/database/orm"

	"github.com/vapor/crypto/sha3pool"
	dbm "github.com/vapor/database/db"
	"github.com/vapor/errors"
)

const tokenSize = 32

var (
	// ErrBadID is returned when Create is called on an invalid id string.
	ErrBadID = errors.New("invalid id")
	// ErrDuplicateID is returned when Create is called on an existing ID.
	ErrDuplicateID = errors.New("duplicate access token ID")
	// ErrBadType is returned when Create is called with a bad type.
	ErrBadType = errors.New("type must be client or network")
	// ErrNoMatchID is returned when Delete is called on nonexisting ID.
	ErrNoMatchID = errors.New("nonexisting access token ID")
	// ErrInvalidToken is returned when Check is called on invalid token
	ErrInvalidToken = errors.New("invalid token")

	// validIDRegexp checks that all characters are alphumeric, _ or -.
	// It also must have a length of at least 1.
	validIDRegexp = regexp.MustCompile(`^[\w-]+$`)
)

// Token describe the access token.
type Token struct {
	ID      string    `json:"id"`
	Token   string    `json:"token,omitempty"`
	Type    string    `json:"type,omitempty"`
	Created time.Time `json:"created_at"`
}

func tokenFromOrmToken(ac orm.AccessToken) *Token {
	return &Token{
		ID:      ac.ID,
		Token:   ac.Token,
		Type:    ac.Type,
		Created: ac.Created,
	}
}

// CredentialStore store user access credential.
type CredentialStore struct {
	DB dbm.SQLDB
}

// NewStore creates and returns a new Store object.
func NewStore(db dbm.SQLDB) *CredentialStore {
	return &CredentialStore{
		DB: db,
	}
}

// Create generates a new access token with the given ID.
func (cs *CredentialStore) Create(id, typ string) (*Token, error) {
	if !validIDRegexp.MatchString(id) {
		return nil, errors.WithDetailf(ErrBadID, "invalid id %q", id)
	}

	accessToken := orm.AccessToken{ID: id}

	if err := cs.DB.Db().Where(&accessToken).Find(&accessToken).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		secret := make([]byte, tokenSize)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		hashedSecret := make([]byte, tokenSize)
		sha3pool.Sum256(hashedSecret, secret)
		accessToken = orm.AccessToken{
			ID:      id,
			Token:   fmt.Sprintf("%s:%x", id, hashedSecret),
			Type:    typ,
			Created: time.Now(),
		}
		if err = cs.DB.Db().Create(&accessToken).Error; err != nil {
			return nil, err
		}
		return tokenFromOrmToken(accessToken), nil
	}
	return nil, errors.WithDetailf(ErrDuplicateID, "id %q already in use", id)
}

// Check returns whether or not an id-secret pair is a valid access token.
func (cs *CredentialStore) Check(id string, secret string) error {
	if !validIDRegexp.MatchString(id) {
		return errors.WithDetailf(ErrBadID, "invalid id %q", id)
	}
	accessToken := orm.AccessToken{ID: id}

	if err := cs.DB.Db().Where(&accessToken).Find(&accessToken).Error; err != nil {
		return err
	}

	if strings.Split(accessToken.Token, ":")[1] == secret {
		return nil
	}

	return ErrInvalidToken
}

// List lists all access tokens.
func (cs *CredentialStore) List() ([]*Token, error) {
	tokens := make([]*Token, 0)
	rows, err := cs.DB.Db().Model(&orm.AccessToken{}).Rows()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		accessToken := orm.AccessToken{}
		if err := rows.Scan(&accessToken.ID, &accessToken.Token, &accessToken.Type, &accessToken.Created); err != nil {
			return nil, err
		}
		tokens = append(tokens, tokenFromOrmToken(accessToken))
	}
	return tokens, nil
}

// Delete deletes an access token by id.
func (cs *CredentialStore) Delete(id string) error {
	if !validIDRegexp.MatchString(id) {
		return errors.WithDetailf(ErrBadID, "invalid id %q", id)
	}
	if err := cs.DB.Db().Delete(&orm.AccessToken{ID: id}).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.WithDetailf(ErrNoMatchID, "check id %q", id)
		}
		return err
	}
	return nil
}
