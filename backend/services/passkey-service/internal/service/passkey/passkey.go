package passkey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"order-fill/backend/services/passkey-service/internal/ceremony"
	"order-fill/backend/services/passkey-service/internal/domain"
	"order-fill/backend/services/passkey-service/internal/webauthn"
)

const challengeTTL = 2 * time.Minute

type Store interface {
	Save(ctx context.Context, cred domain.PasskeyCredential) error
	Get(ctx context.Context, id string) (domain.PasskeyCredential, error)
	List(ctx context.Context, userID string) ([]domain.PasskeyCredential, error)
	Update(ctx context.Context, cred domain.PasskeyCredential) error
	Delete(ctx context.Context, userID, id string) error
}

type Directory interface {
	Lookup(ctx context.Context, login string) (webauthn.Account, error)
}

type Begin struct {
	ChallengeID string
	Options     json.RawMessage
}

type Service struct {
	store      Store
	challenges *ceremony.Store
	wa         webauthn.Ceremony
	users      Directory
	now        func() time.Time
}

func New(store Store, challenges *ceremony.Store, wa webauthn.Ceremony, users Directory, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, challenges: challenges, wa: wa, users: users, now: now}
}

func (s *Service) BeginRegistration(ctx context.Context, userID, origin string) (Begin, error) {
	user := webauthn.Account{ID: userID, Login: userID}
	existing, err := s.store.List(ctx, userID)
	if err != nil {
		return Begin{}, err
	}
	options, session, err := s.wa.BeginRegistration(origin, user, existing)
	if err != nil {
		return Begin{}, err
	}
	return s.saveBegin(userID, domain.PasskeyPurposeRegister, options, session)
}

func (s *Service) FinishRegistration(ctx context.Context, userID, origin, challengeID string, response []byte) (domain.PasskeyPublicView, error) {
	ch, err := s.challenges.Consume(challengeID, s.now())
	if err != nil || ch.Purpose != domain.PasskeyPurposeRegister || ch.UserID != userID {
		return domain.PasskeyPublicView{}, domain.ErrUnauthorized
	}
	user := webauthn.Account{ID: userID, Login: userID}
	cred, err := s.wa.FinishRegistration(origin, user, ch.Session, response)
	if err != nil {
		return domain.PasskeyPublicView{}, domain.ErrUnauthorized
	}
	cred.UserID = userID
	cred.CreatedAt = s.now().UTC()
	if cred.Name == "" {
		cred.Name = cred.DisplayName()
	}
	if err := s.store.Save(ctx, cred); err != nil {
		return domain.PasskeyPublicView{}, err
	}
	return cred.PublicView(), nil
}

func (s *Service) List(ctx context.Context, userID string) ([]domain.PasskeyPublicView, error) {
	items, err := s.store.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PasskeyPublicView, 0, len(items))
	for _, item := range items {
		out = append(out, item.PublicView())
	}
	return out, nil
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return s.store.Delete(ctx, userID, id)
}

func (s *Service) BeginLogin(ctx context.Context, login, origin string) (Begin, error) {
	if login != "" && s.users != nil {
		user, err := s.users.Lookup(ctx, login)
		if err == nil {
			existing, listErr := s.store.List(ctx, user.ID)
			if listErr != nil {
				return Begin{}, listErr
			}
			if len(existing) > 0 {
				options, session, beginErr := s.wa.BeginLogin(origin, user, existing)
				if beginErr != nil {
					return Begin{}, beginErr
				}
				return s.saveBegin(user.ID, domain.PasskeyPurposeLogin, options, session)
			}
		}
	}
	options, session, err := s.wa.BeginDiscoverableLogin(origin)
	if err != nil {
		return Begin{}, err
	}
	return s.saveBegin("", domain.PasskeyPurposeLogin, options, session)
}

func (s *Service) FinishLogin(ctx context.Context, origin, challengeID string, response []byte) (string, error) {
	ch, err := s.challenges.Consume(challengeID, s.now())
	if err != nil || ch.Purpose != domain.PasskeyPurposeLogin {
		return "", domain.ErrUnauthorized
	}
	var cred domain.PasskeyCredential
	if ch.UserID == "" {
		_, cred, err = s.wa.FinishDiscoverableLogin(origin, ch.Session, response, func(rawID, _ []byte) (webauthn.Account, []domain.PasskeyCredential, error) {
			stored, getErr := s.store.Get(ctx, base64.RawURLEncoding.EncodeToString(rawID))
			if getErr != nil {
				return webauthn.Account{}, nil, getErr
			}
			existing, listErr := s.store.List(ctx, stored.UserID)
			if listErr != nil {
				return webauthn.Account{}, nil, listErr
			}
			return webauthn.Account{ID: stored.UserID, Login: stored.UserID}, existing, nil
		})
	} else {
		existing, listErr := s.store.List(ctx, ch.UserID)
		if listErr != nil {
			return "", listErr
		}
		cred, err = s.wa.FinishLogin(origin, webauthn.Account{ID: ch.UserID, Login: ch.UserID}, existing, ch.Session, response)
	}
	if err != nil {
		return "", domain.ErrUnauthorized
	}
	stored, err := s.store.Get(ctx, cred.ID)
	if err != nil {
		return "", domain.ErrUnauthorized
	}
	now := s.now().UTC()
	stored.SignCount = cred.SignCount
	stored.LastUsedAt = &now
	if err := s.store.Update(ctx, stored); err != nil {
		return "", err
	}
	return stored.UserID, nil
}

func (s *Service) saveBegin(userID, purpose string, options, session []byte) (Begin, error) {
	id, err := newID()
	if err != nil {
		return Begin{}, err
	}
	s.challenges.Put(domain.PasskeyChallenge{
		ID:        id,
		UserID:    userID,
		Purpose:   purpose,
		Session:   session,
		ExpiresAt: s.now().Add(challengeTTL),
	})
	return Begin{ChallengeID: id, Options: options}, nil
}

func newID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
