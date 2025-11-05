package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errTokenStoreDBRequired = errors.New("token store database is required")

	// ErrTokenNotFound indicates the provided token does not exist for the
	// associated machine.
	ErrTokenNotFound = errors.New("token not found")
	// ErrTokenExpired signals that the supplied token has elapsed its TTL
	// and can no longer be used.
	ErrTokenExpired = errors.New("token expired")
)

// Token represents a boot or agent authentication token tracked by the store.
type Token struct {
	ID        uuid.UUID
	MAC       string
	Value     string
	ExpiresAt time.Time
	Used      bool
}

type tokenModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	MAC       string    `gorm:"type:text;index;not null"`
	Token     string    `gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null"`
	Used      bool      `gorm:"type:boolean;not null;default:false"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoUpdateTime"`
}

func (tokenModel) TableName() string { return "tokens" }

func (m tokenModel) toToken() Token {
	return Token{
		ID:        m.ID,
		MAC:       m.MAC,
		Value:     m.Token,
		ExpiresAt: m.ExpiresAt,
		Used:      m.Used,
	}
}

type tokenStore struct {
	db  *gorm.DB
	ttl time.Duration
	now func() time.Time
}

func newTokenStore(db *gorm.DB, ttl time.Duration) (*tokenStore, error) {
	if db == nil {
		return nil, errTokenStoreDBRequired
	}

	if ttl <= 0 {
		ttl = defaultTokenTTL
	}

	store := &tokenStore{
		db:  db,
		ttl: ttl,
		now: func() time.Time { return time.Now().UTC() },
	}

	if err := db.AutoMigrate(&tokenModel{}); err != nil {
		return nil, fmt.Errorf("migrate tokens table: %w", err)
	}

	return store, nil
}

func (ts *tokenStore) purgeExpired(ctx context.Context, tx *gorm.DB) error {
	now := ts.now()
	if err := tx.WithContext(ctx).Where("expires_at <= ?", now).Delete(&tokenModel{}).Error; err != nil {
		return fmt.Errorf("purge expired tokens: %w", err)
	}
	return nil
}

func (ts *tokenStore) Issue(ctx context.Context, mac string) (Token, error) {
	mac = strings.TrimSpace(strings.ToLower(mac))
	if mac == "" {
		return Token{}, errors.New("mac is required")
	}

	if err := ts.purgeExpired(ctx, ts.db); err != nil {
		return Token{}, err
	}

	now := ts.now()
	model := tokenModel{
		ID:        uuid.New(),
		MAC:       mac,
		Token:     uuid.NewString(),
		ExpiresAt: now.Add(ts.ttl),
		Used:      false,
	}

	if err := ts.db.WithContext(ctx).Create(&model).Error; err != nil {
		return Token{}, fmt.Errorf("create token: %w", err)
	}

	return model.toToken(), nil
}

func (ts *tokenStore) Active(ctx context.Context, mac string) (Token, bool, error) {
	mac = strings.TrimSpace(strings.ToLower(mac))
	if mac == "" {
		return Token{}, false, errors.New("mac is required")
	}

	if err := ts.purgeExpired(ctx, ts.db); err != nil {
		return Token{}, false, err
	}

	now := ts.now()
	var model tokenModel
	err := ts.db.WithContext(ctx).
		Where("mac = ? AND used = FALSE AND expires_at > ?", mac, now).
		Order("expires_at DESC").
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Token{}, false, nil
	}
	if err != nil {
		return Token{}, false, fmt.Errorf("lookup active token: %w", err)
	}

	return model.toToken(), true, nil
}

func (ts *tokenStore) MarkUsed(ctx context.Context, tokenValue string) error {
	tokenValue = strings.TrimSpace(tokenValue)
	if tokenValue == "" {
		return errors.New("token is required")
	}

	res := ts.db.WithContext(ctx).
		Model(&tokenModel{}).
		Where("token = ?", tokenValue).
		Update("used", true)
	if res.Error != nil {
		return fmt.Errorf("mark token used: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrTokenNotFound
	}

	return nil
}

func (ts *tokenStore) Rotate(ctx context.Context, mac, oldToken string) (Token, error) {
	mac = strings.TrimSpace(strings.ToLower(mac))
	if mac == "" {
		return Token{}, errors.New("mac is required")
	}

	oldToken = strings.TrimSpace(oldToken)
	if oldToken == "" {
		return Token{}, errors.New("old token is required")
	}

	tx := ts.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return Token{}, tx.Error
	}
	defer func() {
		if tx.Error != nil {
			tx.Rollback()
		}
	}()

	if err := ts.purgeExpired(ctx, tx); err != nil {
		tx.Rollback()
		return Token{}, err
	}

	now := ts.now()
	var current tokenModel
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("mac = ? AND token = ?", mac, oldToken).
		First(&current).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return Token{}, ErrTokenNotFound
	}
	if err != nil {
		tx.Rollback()
		return Token{}, fmt.Errorf("lookup token: %w", err)
	}

	if !current.ExpiresAt.After(now) {
		tx.Rollback()
		return Token{}, ErrTokenExpired
	}

	if err := tx.Model(&current).Updates(map[string]any{
		"used":       true,
		"expires_at": now,
	}).Error; err != nil {
		tx.Rollback()
		return Token{}, fmt.Errorf("invalidate token: %w", err)
	}

	replacement := tokenModel{
		ID:        uuid.New(),
		MAC:       mac,
		Token:     uuid.NewString(),
		ExpiresAt: now.Add(ts.ttl),
		Used:      false,
	}

	if err := tx.Create(&replacement).Error; err != nil {
		tx.Rollback()
		return Token{}, fmt.Errorf("create replacement token: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return Token{}, err
	}

	return replacement.toToken(), nil
}
