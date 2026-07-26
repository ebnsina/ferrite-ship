// Package auth handles passwords and sessions.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/ebnsina/ferrite-ship/internal/ids"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

var (
	ErrInvalidCredentials = errors.New("that email and password do not match")
	ErrWeakPassword       = errors.New("password is too short")
	ErrSetupClosed        = errors.New("an account already exists")
	ErrNoSession          = errors.New("not signed in")
)

// SessionLifetime is how long a sign-in lasts. Long enough not to be a
// nuisance for a tool you keep open, short enough that a forgotten laptop
// stops being a way in.
const SessionLifetime = 14 * 24 * time.Hour

// MinPasswordLength favours length over composition rules, which push people
// towards predictable substitutions without adding real strength.
const MinPasswordLength = 10

// argon2id parameters. Deliberately costly: this runs once per sign-in, and
// the cost is what makes a stolen database expensive to attack.
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

type Service struct {
	store *store.Store
}

func NewService(st *store.Store) *Service { return &Service{store: st} }

// HashPassword returns an encoded argon2id hash, salt included.
func HashPassword(password string) (string, error) {
	if len([]rune(password)) < MinPasswordLength {
		return "", ErrWeakPassword
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: read salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword compares in constant time, so the answer cannot be learned
// by measuring how long the comparison took.
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}

	var memory uint32
	var iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	got := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// NeedsSetup reports whether no account exists yet.
func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// Setup creates the first and only account. It refuses once one exists, so
// the bootstrap cannot be replayed to add a second way in.
func (s *Service) Setup(ctx context.Context, email, password string) (store.User, error) {
	needsSetup, err := s.NeedsSetup(ctx)
	if err != nil {
		return store.User{}, err
	}
	if !needsSetup {
		return store.User{}, ErrSetupClosed
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") {
		return store.User{}, errors.New("that does not look like an email address")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return store.User{}, err
	}

	user := store.User{
		ID:           ids.New("usr"),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return store.User{}, err
	}

	// The first account adopts anything created before ownership existed.
	if _, err := s.store.ClaimUnownedServers(ctx, user.ID); err != nil {
		return store.User{}, err
	}

	return user, nil
}

// Create makes an account regardless of how many already exist. Setup uses
// the guarded path; this is for the command line, where the caller has already
// proved control of the machine.
func (s *Service) Create(ctx context.Context, email, password string) (store.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") {
		return store.User{}, errors.New("that does not look like an email address")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return store.User{}, err
	}

	user := store.User{
		ID:           ids.New("usr"),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return store.User{}, err
	}
	return user, nil
}

// GeneratePassword returns a long random password. Length rather than symbol
// soup: it is meant to be copied into a password manager, not memorised.
func GeneratePassword() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: generate password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Authenticate checks credentials and opens a session.
func (s *Service) Authenticate(ctx context.Context, email, password string) (store.Session, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		// Hash anyway so a missing account and a wrong password take the same
		// time, and neither can be told apart from the outside.
		VerifyPassword("$argon2id$v=19$m=65536,t=3,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", password)
		return store.Session{}, ErrInvalidCredentials
	}

	if !VerifyPassword(user.PasswordHash, password) {
		return store.Session{}, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	session := store.Session{
		ID:        newSessionID(),
		UserID:    user.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(SessionLifetime),
	}
	if err := s.store.CreateSession(ctx, session); err != nil {
		return store.Session{}, err
	}

	// Opportunistic tidy-up; a failure here is not worth failing a sign-in.
	_ = s.store.DeleteExpiredSessions(ctx, now)

	return session, nil
}

// UserForSession resolves a cookie value to the account that owns it.
func (s *Service) UserForSession(ctx context.Context, sessionID string) (store.User, error) {
	if sessionID == "" {
		return store.User{}, ErrNoSession
	}

	session, err := s.store.GetValidSession(ctx, sessionID, time.Now().UTC())
	if err != nil {
		return store.User{}, ErrNoSession
	}
	return s.store.GetUser(ctx, session.UserID)
}

func (s *Service) SignOut(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, sessionID)
}

// newSessionID returns 256 bits of randomness. Session identifiers are bearer
// tokens, so they must not be guessable.
func newSessionID() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("auth: could not read random bytes: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
