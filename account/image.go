// Package account stores and tracks accounts within a Bytom Core.
package account

import (
	log "github.com/sirupsen/logrus"
)

// ImageSlice record info of single account
type ImageSlice struct {
	Account       *Account `json:"account"`
	ContractIndex uint64   `json:"contract_index"`
}

// Image is the struct for hold export account data
type Image struct {
	Slice []*ImageSlice `json:"slices"`
}

// Backup export all the account info into image
func (m *Manager) Backup() (*Image, error) {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	image := &Image{
		Slice: []*ImageSlice{},
	}

	accounts, err := m.store.ListAccounts("")
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		image.Slice = append(image.Slice, &ImageSlice{
			Account:       account,
			ContractIndex: m.store.GetContractIndex(account.ID),
		})
	}
	return image, nil
}

// Restore import the accountImages into account manage
func (m *Manager) Restore(image *Image) error {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	m.store.InitBatch()

	for _, slice := range image.Slice {
		existed, err := m.store.GetAccountByID(slice.Account.ID)
		if err != nil || existed != nil {
			log.WithFields(log.Fields{
				"module": logModule,
				"alias":  slice.Account.Alias,
				"id":     slice.Account.ID,
			}).Warning("skip restore account due to already existed")
			continue
		}
		if existed := m.store.GetAccountIDByAlias(slice.Account.Alias); existed != "" {
			return ErrDuplicateAlias
		}

		if err := m.store.SetAccount(slice.Account); err != nil {
			return err
		}
	}

	m.store.CommitBatch()

	return nil
}
