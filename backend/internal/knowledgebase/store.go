package knowledgebase

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/juniorqazaq/Agro/backend/internal/domain"
)

type Store struct {
	byID map[string]domain.Disease
	ids  []string
}

func Load(path string) (*Store, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read knowledge base: %w", err)
	}

	var diseases []domain.Disease
	if err := json.Unmarshal(raw, &diseases); err != nil {
		return nil, fmt.Errorf("decode knowledge base: %w", err)
	}
	if len(diseases) == 0 {
		return nil, errors.New("knowledge base is empty")
	}

	store := &Store{
		byID: make(map[string]domain.Disease, len(diseases)),
		ids:  make([]string, 0, len(diseases)),
	}
	for i, disease := range diseases {
		disease.ID = strings.TrimSpace(disease.ID)
		if disease.ID == "" {
			return nil, fmt.Errorf("knowledge base entry %d has an empty id", i)
		}
		if _, exists := store.byID[disease.ID]; exists {
			return nil, fmt.Errorf("duplicate disease id %q", disease.ID)
		}
		store.byID[disease.ID] = disease
		store.ids = append(store.ids, disease.ID)
	}

	return store, nil
}

func (s *Store) Find(id string) (domain.Disease, bool) {
	disease, ok := s.byID[strings.TrimSpace(id)]
	return disease, ok
}

func (s *Store) IDs() []string {
	return append([]string(nil), s.ids...)
}
