package keys

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"
)

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I/O/0/1

type Tier string

const (
	TierFull Tier = "FULL"
)

type Key struct {
	Code       string    `json:"code"`
	Tier       Tier      `json:"tier"`
	Email      string    `json:"email,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	RedeemedAt *time.Time `json:"redeemed_at,omitempty"`
	Revoked    bool      `json:"revoked,omitempty"`
	StripeID   string    `json:"stripe_id,omitempty"`
}

type Store struct {
	mu   sync.RWMutex
	keys map[string]*Key
	path string
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		keys: make(map[string]*Key),
		path: path,
	}
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var keys []*Key
		if err := json.Unmarshal(data, &keys); err != nil {
			return nil, err
		}
		for _, k := range keys {
			s.keys[k.Code] = k
		}
	}
	return s, nil
}

func (s *Store) save() error {
	keys := make([]*Key, 0, len(s.keys))
	for _, k := range s.keys {
		keys = append(keys, k)
	}
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *Store) Issue(tier Tier, email, stripeID string) (*Key, error) {
	code, err := generateCode(tier)
	if err != nil {
		return nil, err
	}

	k := &Key{
		Code:      code,
		Tier:      tier,
		Email:     email,
		CreatedAt: time.Now().UTC(),
		StripeID:  stripeID,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[code] = k
	if err := s.save(); err != nil {
		return nil, err
	}
	return k, nil
}

func (s *Store) Redeem(code string) (*Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k, ok := s.keys[code]
	if !ok {
		return nil, fmt.Errorf("invalid key")
	}
	if k.Revoked {
		return nil, fmt.Errorf("key revoked")
	}
	if k.RedeemedAt != nil {
		return nil, fmt.Errorf("key already redeemed")
	}

	now := time.Now().UTC()
	k.RedeemedAt = &now
	if err := s.save(); err != nil {
		return nil, err
	}
	return k, nil
}

func (s *Store) Validate(code string) (*Key, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	k, ok := s.keys[code]
	if !ok {
		return nil, fmt.Errorf("invalid key")
	}
	return k, nil
}

func (s *Store) Revoke(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k, ok := s.keys[code]
	if !ok {
		return fmt.Errorf("invalid key")
	}
	k.Revoked = true
	return s.save()
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.keys)
}

func generateCode(tier Tier) (string, error) {
	seg1, err := randomSegment(4)
	if err != nil {
		return "", err
	}
	seg2, err := randomSegment(4)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("WAVE-%s-%s-%s", tier, seg1, seg2), nil
}

func randomSegment(length int) (string, error) {
	var sb strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		sb.WriteByte(charset[n.Int64()])
	}
	return sb.String(), nil
}
