package localauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	RoleAdmin    = "admin"
	StatusActive = "active"
)

var (
	ErrInvalidConfiguration = errors.New("local authentication configuration is invalid")
	ErrInvalidCredentials   = errors.New("email or password is invalid")
	ErrSessionInvalid       = errors.New("session is invalid")

	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)
)

type Workspace struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

type Principal struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type Session struct {
	ID             string
	Principal      Principal
	TokenDigest    [32]byte
	CreatedAt      time.Time
	LastSeenAt     time.Time
	IdleExpiresAt  time.Time
	AbsoluteExpiry time.Time
}

type Config struct {
	Workspace          Workspace
	User               User
	Password           string
	PasswordParameters PasswordParameters
	IdleTimeout        time.Duration
	AbsoluteTimeout    time.Duration
	Now                func() time.Time
	Random             io.Reader
}

type Service struct {
	mu              sync.Mutex
	workspace       Workspace
	user            User
	sessions        map[[32]byte]Session
	idleTimeout     time.Duration
	absoluteTimeout time.Duration
	now             func() time.Time
	random          io.Reader
}

func New(config Config) (*Service, error) {
	config.Workspace.ID = strings.ToLower(config.Workspace.ID)
	config.Workspace.Slug = strings.ToLower(config.Workspace.Slug)
	config.User.ID = strings.ToLower(config.User.ID)
	config.User.WorkspaceID = strings.ToLower(config.User.WorkspaceID)
	config.User.Email = normalizeEmail(config.User.Email)
	if config.PasswordParameters == (PasswordParameters{}) {
		config.PasswordParameters = DefaultPasswordParameters()
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Random == nil {
		config.Random = rand.Reader
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 30 * time.Minute
	}
	if config.AbsoluteTimeout == 0 {
		config.AbsoluteTimeout = 8 * time.Hour
	}
	if !uuidPattern.MatchString(config.Workspace.ID) ||
		!slugPattern.MatchString(config.Workspace.Slug) ||
		len(config.Workspace.Name) < 2 || len(config.Workspace.Name) > 80 ||
		!uuidPattern.MatchString(config.User.ID) ||
		config.User.WorkspaceID != config.Workspace.ID ||
		!validEmail(config.User.Email) ||
		len(config.User.DisplayName) < 2 || len(config.User.DisplayName) > 80 ||
		config.User.Role != RoleAdmin ||
		config.User.Status != StatusActive ||
		config.IdleTimeout < time.Minute || config.IdleTimeout > time.Hour ||
		config.AbsoluteTimeout < config.IdleTimeout || config.AbsoluteTimeout > 24*time.Hour {
		return nil, ErrInvalidConfiguration
	}
	passwordHash, err := HashPassword(config.Password, config.PasswordParameters)
	if err != nil {
		return nil, err
	}
	now := config.Now().UTC()
	config.Workspace.CreatedAt = now
	config.User.CreatedAt = now
	config.User.PasswordHash = passwordHash
	return &Service{
		workspace:       config.Workspace,
		user:            config.User,
		sessions:        make(map[[32]byte]Session),
		idleTimeout:     config.IdleTimeout,
		absoluteTimeout: config.AbsoluteTimeout,
		now:             config.Now,
		random:          config.Random,
	}, nil
}

func (service *Service) Authenticate(email, password string) (Principal, error) {
	service.mu.Lock()
	user := service.user
	service.mu.Unlock()
	matchedEmail := normalizeEmail(email) == user.Email && user.Status == StatusActive
	matchedPassword, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !matchedEmail || !matchedPassword {
		return Principal{}, ErrInvalidCredentials
	}
	return principalFor(user), nil
}

func (service *Service) CreateSession(principal Principal) (string, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if principal != principalFor(service.user) {
		return "", ErrSessionInvalid
	}
	token, err := randomToken(service.random, 32)
	if err != nil {
		return "", err
	}
	sessionID, err := randomToken(service.random, 18)
	if err != nil {
		return "", err
	}
	now := service.now().UTC()
	digest := sha256.Sum256([]byte(token))
	service.sessions[digest] = Session{
		ID:             "ses_" + sessionID,
		Principal:      principal,
		TokenDigest:    digest,
		CreatedAt:      now,
		LastSeenAt:     now,
		IdleExpiresAt:  now.Add(service.idleTimeout),
		AbsoluteExpiry: now.Add(service.absoluteTimeout),
	}
	return token, nil
}

func (service *Service) VerifySession(token string) (Principal, error) {
	if len(token) < 32 || len(token) > 128 {
		return Principal{}, ErrSessionInvalid
	}
	digest := sha256.Sum256([]byte(token))
	service.mu.Lock()
	defer service.mu.Unlock()
	session, ok := service.sessions[digest]
	if !ok {
		return Principal{}, ErrSessionInvalid
	}
	now := service.now().UTC()
	if !now.Before(session.IdleExpiresAt) || !now.Before(session.AbsoluteExpiry) {
		delete(service.sessions, digest)
		return Principal{}, ErrSessionInvalid
	}
	session.LastSeenAt = now
	session.IdleExpiresAt = minimumTime(now.Add(service.idleTimeout), session.AbsoluteExpiry)
	service.sessions[digest] = session
	return session.Principal, nil
}

func (service *Service) RevokeSession(token string) {
	digest := sha256.Sum256([]byte(token))
	service.mu.Lock()
	delete(service.sessions, digest)
	service.mu.Unlock()
}

func (service *Service) Workspace() Workspace {
	return service.workspace
}

func principalFor(user User) Principal {
	return Principal{
		UserID:      user.ID,
		WorkspaceID: user.WorkspaceID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validEmail(value string) bool {
	if len(value) < 5 || len(value) > 254 || strings.Count(value, "@") != 1 {
		return false
	}
	parts := strings.SplitN(value, "@", 2)
	return parts[0] != "" && strings.Contains(parts[1], ".") &&
		!strings.HasPrefix(parts[1], ".") && !strings.HasSuffix(parts[1], ".")
}

func randomToken(reader io.Reader, size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func minimumTime(first, second time.Time) time.Time {
	if first.Before(second) {
		return first
	}
	return second
}
