package service

import (
	"context"

	"github.com/worddeck/internal/auth"
	"github.com/worddeck/internal/domain"
	"github.com/worddeck/internal/store"
)

const shareTokenBytes = 12

type Shares struct {
	store *store.Store
}

func NewShares(st *store.Store) *Shares { return &Shares{store: st} }

func (s *Shares) List(ctx context.Context, ownerID int64, only *domain.ResourceType, id int64) ([]domain.ShareLink, error) {
	return s.store.ListShares(ctx, ownerID, only, id)
}

func (s *Shares) Create(ctx context.Context, ownerID int64, t domain.ResourceType, id int64) (domain.ShareLink, error) {
	if !t.Valid() {
		return domain.ShareLink{}, domain.Invalid("resource_type", domain.CodeInvalid)
	}
	token, err := auth.RandomToken(shareTokenBytes)
	if err != nil {
		return domain.ShareLink{}, err
	}
	return s.store.CreateShare(ctx, ownerID, t, id, token)
}

func (s *Shares) Delete(ctx context.Context, ownerID int64, token string) error {
	return s.store.DeleteShare(ctx, ownerID, token)
}

func (s *Shares) Resolve(ctx context.Context, token string) (domain.SharedContent, error) {
	link, err := s.store.GetShare(ctx, token)
	if err != nil {
		return domain.SharedContent{}, err
	}
	c := domain.SharedContent{Link: link}
	if c.OwnerName, err = s.store.UserDisplayName(ctx, link.OwnerID); err != nil {
		return c, err
	}
	switch link.ResourceType {
	case domain.ResourceWorkspace:
		w, err := s.store.SharedWorkspace(ctx, link.ResourceID)
		c.Workspace = &w
		return c, err
	case domain.ResourceFolder:
		f, err := s.store.SharedFolder(ctx, link.ResourceID)
		c.Folder = &f
		return c, err
	default:
		card, err := s.store.SharedCard(ctx, link.ResourceID)
		c.Card = &card
		return c, err
	}
}

// Область копируется в новую область, папка — в целевую область,
// карточка — в целевую папку.
func (s *Shares) Copy(ctx context.Context, userID int64, token string, targetID int64) (domain.CopyResult, error) {
	var res domain.CopyResult
	err := s.store.InTx(ctx, func(tx *store.Store) error {
		link, err := tx.GetShare(ctx, token)
		if err != nil {
			return err
		}
		res.ResourceType = link.ResourceType
		if link.ResourceType != domain.ResourceWorkspace && targetID <= 0 {
			return domain.Invalid("target_id", domain.CodeRequired)
		}
		switch link.ResourceType {
		case domain.ResourceWorkspace:
			res.ID, res.CardsCopied, err = tx.CopyWorkspace(ctx, link.ResourceID, userID)
		case domain.ResourceFolder:
			if err := tx.CheckWorkspaceOwner(ctx, userID, targetID); err != nil {
				return err
			}
			res.ID, res.CardsCopied, err = tx.CopyFolder(ctx, link.ResourceID, targetID)
		case domain.ResourceCard:
			if err := tx.CheckFolderOwner(ctx, userID, targetID); err != nil {
				return err
			}
			res.CardsCopied = 1
			res.ID, err = tx.CopyCard(ctx, link.ResourceID, targetID)
		}
		return err
	})
	return res, err
}
