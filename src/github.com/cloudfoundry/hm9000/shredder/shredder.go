package shredder

import (
	storepackage "github.com/cloudfoundry/hm9000/store"
)

type Shredder struct {
	store storepackage.Store
}

func New(store storepackage.Store) *Shredder {
	return &Shredder{store}
}

func (s *Shredder) Shred() error {
	return s.store.Compact()
}
