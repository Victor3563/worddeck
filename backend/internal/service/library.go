package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/worddeck/internal/cardio"
	"github.com/worddeck/internal/domain"
	"github.com/worddeck/internal/store"
)

type Library struct {
	store *store.Store
	log   *slog.Logger
}

func NewLibrary(st *store.Store, log *slog.Logger) *Library { return &Library{store: st, log: log} }

func (l *Library) Workspaces(ctx context.Context, ownerID int64) ([]domain.Workspace, error) {
	return l.store.ListWorkspaces(ctx, ownerID)
}

func (l *Library) Workspace(ctx context.Context, ownerID, id int64) (domain.Workspace, error) {
	return l.store.GetWorkspace(ctx, ownerID, id)
}

func (l *Library) CreateWorkspace(ctx context.Context, ownerID int64, in domain.WorkspaceFields) (domain.Workspace, error) {
	if in.Color == "" {
		in.Color = domain.DefaultColor
	}
	p, err := validWorkspacePatch(domain.WorkspacePatch{Name: &in.Name, Description: &in.Description, Color: &in.Color})
	if err != nil {
		return domain.Workspace{}, err
	}
	return l.store.CreateWorkspace(ctx, ownerID, domain.WorkspaceFields{Name: *p.Name, Description: *p.Description, Color: *p.Color})
}

func (l *Library) UpdateWorkspace(ctx context.Context, ownerID, id int64, p domain.WorkspacePatch) (domain.Workspace, error) {
	p, err := validWorkspacePatch(p)
	if err != nil {
		return domain.Workspace{}, err
	}
	return l.store.UpdateWorkspace(ctx, ownerID, id, p)
}

func validWorkspacePatch(p domain.WorkspacePatch) (domain.WorkspacePatch, error) {
	var err error
	if p.Name, err = domain.OptionalText("name", p.Name, true, domain.MaxNameLen); err != nil {
		return p, err
	}
	if p.Description, err = domain.OptionalText("description", p.Description, false, domain.MaxDescriptionLen); err != nil {
		return p, err
	}
	if p.Color != nil && !p.Color.Valid() {
		return p, domain.Invalid("color", domain.CodeInvalid)
	}
	return p, nil
}

func (l *Library) DeleteWorkspace(ctx context.Context, ownerID, id int64) error {
	return l.store.DeleteWorkspace(ctx, ownerID, id)
}

func (l *Library) WorkspaceCards(ctx context.Context, ownerID, id int64) ([]domain.Card, error) {
	return l.store.ListWorkspaceCards(ctx, ownerID, id)
}

func (l *Library) Folders(ctx context.Context, ownerID, workspaceID int64) ([]domain.Folder, error) {
	return l.store.ListFolders(ctx, ownerID, workspaceID)
}

func (l *Library) Folder(ctx context.Context, ownerID, id int64) (domain.Folder, error) {
	return l.store.GetFolder(ctx, ownerID, id)
}

func (l *Library) CreateFolder(ctx context.Context, ownerID, workspaceID int64, in domain.FolderFields) (domain.Folder, error) {
	p, err := validFolderPatch(domain.FolderPatch{Name: &in.Name, Description: &in.Description})
	if err != nil {
		return domain.Folder{}, err
	}
	return l.store.CreateFolder(ctx, ownerID, workspaceID, domain.FolderFields{Name: *p.Name, Description: *p.Description})
}

func (l *Library) UpdateFolder(ctx context.Context, ownerID, id int64, p domain.FolderPatch) (domain.Folder, error) {
	p, err := validFolderPatch(p)
	if err != nil {
		return domain.Folder{}, err
	}
	return l.store.UpdateFolder(ctx, ownerID, id, p)
}

func validFolderPatch(p domain.FolderPatch) (domain.FolderPatch, error) {
	var err error
	if p.Name, err = domain.OptionalText("name", p.Name, true, domain.MaxNameLen); err != nil {
		return p, err
	}
	p.Description, err = domain.OptionalText("description", p.Description, false, domain.MaxDescriptionLen)
	return p, err
}

func (l *Library) DeleteFolder(ctx context.Context, ownerID, id int64) error {
	return l.store.DeleteFolder(ctx, ownerID, id)
}

func (l *Library) Cards(ctx context.Context, ownerID, folderID int64, search string) ([]domain.Card, error) {
	return l.store.ListCards(ctx, ownerID, folderID, strings.TrimSpace(search))
}

func (l *Library) Card(ctx context.Context, ownerID, id int64) (domain.Card, error) {
	return l.store.GetCard(ctx, ownerID, id)
}

func (l *Library) CreateCard(ctx context.Context, ownerID, folderID int64, in domain.CardFields) (domain.Card, error) {
	in, err := in.Normalize()
	if err != nil {
		return domain.Card{}, err
	}
	cards, err := l.store.CreateCards(ctx, ownerID, folderID, []domain.CardFields{in})
	if err != nil {
		return domain.Card{}, err
	}
	return cards[0], nil
}

// Одна некорректная карточка отклоняет весь пакет.
func (l *Library) CreateCards(ctx context.Context, ownerID, folderID int64, in []domain.CardFields) ([]domain.Card, error) {
	if len(in) == 0 {
		return nil, domain.Invalid("cards", domain.CodeRequired)
	}
	if len(in) > domain.MaxCardsPerRequest {
		return nil, domain.TooMany("cards", domain.MaxCardsPerRequest)
	}
	valid := make([]domain.CardFields, len(in))
	for i, c := range in {
		n, err := c.Normalize()
		if err != nil {
			var ve *domain.ValidationError
			if errors.As(err, &ve) {
				return nil, ve.InItem("cards", i+1)
			}
			return nil, err
		}
		valid[i] = n
	}
	return l.store.CreateCards(ctx, ownerID, folderID, valid)
}

func (l *Library) UpdateCard(ctx context.Context, ownerID, id int64, p domain.CardPatch) (domain.Card, error) {
	var err error
	if p.RuWord, err = domain.OptionalText("ru_word", p.RuWord, true, domain.MaxWordLen); err != nil {
		return domain.Card{}, err
	}
	if p.EnWord, err = domain.OptionalText("en_word", p.EnWord, true, domain.MaxWordLen); err != nil {
		return domain.Card{}, err
	}
	if p.RuExample, err = domain.OptionalText("ru_example", p.RuExample, false, domain.MaxExampleLen); err != nil {
		return domain.Card{}, err
	}
	if p.EnExample, err = domain.OptionalText("en_example", p.EnExample, false, domain.MaxExampleLen); err != nil {
		return domain.Card{}, err
	}
	return l.store.UpdateCard(ctx, ownerID, id, p)
}

func (l *Library) DeleteCard(ctx context.Context, ownerID, id int64) error {
	return l.store.DeleteCard(ctx, ownerID, id)
}

func (l *Library) DeleteCards(ctx context.Context, ownerID, folderID int64, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, domain.Invalid("ids", domain.CodeRequired)
	}
	return l.store.DeleteCards(ctx, ownerID, folderID, ids)
}

type ImportResult struct {
	cardio.Result
	DryRun  bool
	Created int
}

func (l *Library) Import(ctx context.Context, ownerID, folderID int64, data []byte, dryRun bool) (ImportResult, error) {
	if err := l.store.CheckFolderOwner(ctx, ownerID, folderID); err != nil {
		return ImportResult{}, err
	}
	parsed, err := cardio.Parse(data)
	if err != nil {
		return ImportResult{}, err
	}
	res := ImportResult{Result: parsed, DryRun: dryRun}
	if dryRun {
		return res, nil
	}
	if len(parsed.Cards) == 0 {
		return res, domain.Invalid("file", domain.CodeNoValidCards)
	}
	created, err := l.store.CreateCards(ctx, ownerID, folderID, parsed.Cards)
	if err != nil {
		return res, err
	}
	res.Created = len(created)
	l.log.Info("cards imported", "folder_id", folderID, "count", res.Created, "skipped", len(parsed.Errors))
	return res, nil
}

func (l *Library) Export(ctx context.Context, ownerID, folderID int64) (domain.Folder, []cardio.Card, error) {
	f, err := l.store.GetFolder(ctx, ownerID, folderID)
	if err != nil {
		return f, nil, err
	}
	cards, err := l.store.ListCards(ctx, ownerID, folderID, "")
	if err != nil {
		return f, nil, err
	}
	out := make([]cardio.Card, len(cards))
	for i, c := range cards {
		out[i] = c.CardFields
	}
	return f, out, nil
}
