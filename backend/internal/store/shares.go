package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/worddeck/internal/domain"
)

// Имя колонки берётся из этого списка, а не из запроса, поэтому его можно подставлять в SQL.
var shareColumn = map[domain.ResourceType]string{
	domain.ResourceWorkspace: "workspace_id",
	domain.ResourceFolder:    "folder_id",
	domain.ResourceCard:      "card_id",
}

const shareSelect = `
	SELECT s.token, s.owner_id, s.created_at, s.workspace_id, s.folder_id, s.card_id,
	       COALESCE(w.name, f.name, c.en_word || ' — ' || c.ru_word, '')
	FROM share_links s
	LEFT JOIN workspaces w ON w.id = s.workspace_id
	LEFT JOIN folders f    ON f.id = s.folder_id
	LEFT JOIN cards c      ON c.id = s.card_id`

func scanShare(sc scanner) (domain.ShareLink, error) {
	var (
		l             domain.ShareLink
		ws, fold, crd sql.NullInt64
	)
	if err := sc.Scan(&l.Token, &l.OwnerID, &l.CreatedAt, &ws, &fold, &crd, &l.ResourceName); err != nil {
		return l, notFound(err)
	}
	switch {
	case ws.Valid:
		l.ResourceType, l.ResourceID = domain.ResourceWorkspace, ws.Int64
	case fold.Valid:
		l.ResourceType, l.ResourceID = domain.ResourceFolder, fold.Int64
	case crd.Valid:
		l.ResourceType, l.ResourceID = domain.ResourceCard, crd.Int64
	}
	return l, nil
}

func (s *Store) checkResourceOwner(ctx context.Context, ownerID int64, t domain.ResourceType, id int64) error {
	switch t {
	case domain.ResourceWorkspace:
		return s.CheckWorkspaceOwner(ctx, ownerID, id)
	case domain.ResourceFolder:
		return s.CheckFolderOwner(ctx, ownerID, id)
	case domain.ResourceCard:
		_, err := s.GetCard(ctx, ownerID, id)
		return err
	}
	return domain.ErrNotFound
}

func (s *Store) CreateShare(ctx context.Context, ownerID int64, t domain.ResourceType, id int64, token string) (domain.ShareLink, error) {
	col, ok := shareColumn[t]
	if !ok {
		return domain.ShareLink{}, domain.ErrNotFound
	}
	if err := s.checkResourceOwner(ctx, ownerID, t, id); err != nil {
		return domain.ShareLink{}, err
	}
	if _, err := s.q.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO share_links (token, owner_id, %s) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, col),
		token, ownerID, id); err != nil {
		return domain.ShareLink{}, err
	}
	return scanShare(s.q.QueryRowContext(ctx, shareSelect+fmt.Sprintf(` WHERE s.%s = $1`, col), id))
}

func (s *Store) ListShares(ctx context.Context, ownerID int64, only *domain.ResourceType, id int64) ([]domain.ShareLink, error) {
	q, args := shareSelect+` WHERE s.owner_id = $1`, []any{ownerID}
	if only != nil {
		col, ok := shareColumn[*only]
		if !ok {
			return []domain.ShareLink{}, nil
		}
		q += fmt.Sprintf(` AND s.%s = $2`, col)
		args = append(args, id)
	}
	rows, err := s.q.QueryContext(ctx, q+` ORDER BY s.created_at DESC`, args...)
	return collect(rows, err, scanShare)
}

func (s *Store) DeleteShare(ctx context.Context, ownerID int64, token string) error {
	return affected(s.q.ExecContext(ctx, `DELETE FROM share_links WHERE token = $1 AND owner_id = $2`, token, ownerID))
}

func (s *Store) GetShare(ctx context.Context, token string) (domain.ShareLink, error) {
	return scanShare(s.q.QueryRowContext(ctx, shareSelect+` WHERE s.token = $1`, token))
}

// Чтение по ссылке: владелец не проверяется.

func (s *Store) UserDisplayName(ctx context.Context, id int64) (string, error) {
	var name string
	err := s.q.QueryRowContext(ctx, `SELECT display_name FROM users WHERE id = $1`, id).Scan(&name)
	return name, notFound(err)
}

func (s *Store) SharedCard(ctx context.Context, id int64) (domain.Card, error) {
	return scanCard(s.q.QueryRowContext(ctx, `SELECT `+cardCols+` FROM cards c WHERE c.id = $1`, id))
}

func (s *Store) SharedFolder(ctx context.Context, id int64) (domain.SharedFolder, error) {
	f := domain.SharedFolder{ID: id}
	if err := s.q.QueryRowContext(ctx, `SELECT name, description FROM folders WHERE id = $1`, id).
		Scan(&f.Name, &f.Description); err != nil {
		return f, notFound(err)
	}
	rows, err := s.q.QueryContext(ctx, `SELECT `+cardCols+` FROM cards c WHERE c.folder_id = $1 ORDER BY c.created_at, c.id`, id)
	f.Cards, err = collect(rows, err, scanCard)
	return f, err
}

func (s *Store) SharedWorkspace(ctx context.Context, id int64) (domain.SharedWorkspace, error) {
	w := domain.SharedWorkspace{ID: id}
	if err := s.q.QueryRowContext(ctx, `SELECT name, description, color FROM workspaces WHERE id = $1`, id).
		Scan(&w.Name, &w.Description, &w.Color); err != nil {
		return w, notFound(err)
	}
	rows, err := s.q.QueryContext(ctx, `SELECT id, name, description FROM folders WHERE workspace_id = $1 ORDER BY created_at, id`, id)
	w.Folders, err = collect(rows, err, func(sc scanner) (domain.SharedFolder, error) {
		f := domain.SharedFolder{Cards: []domain.Card{}}
		return f, sc.Scan(&f.ID, &f.Name, &f.Description)
	})
	if err != nil {
		return w, err
	}
	rows, err = s.q.QueryContext(ctx, `
		SELECT `+cardCols+` FROM cards c JOIN folders f ON f.id = c.folder_id
		WHERE f.workspace_id = $1 ORDER BY c.created_at, c.id`, id)
	cards, err := collect(rows, err, scanCard)
	if err != nil {
		return w, err
	}
	index := make(map[int64]int, len(w.Folders))
	for i, f := range w.Folders {
		index[f.ID] = i
	}
	for _, c := range cards {
		i := index[c.FolderID]
		w.Folders[i].Cards = append(w.Folders[i].Cards, c)
	}
	return w, nil
}

// Функции копирования вызываются внутри InTx.

func (s *Store) CopyWorkspace(ctx context.Context, srcID, ownerID int64) (newID int64, cards int, err error) {
	if err = s.q.QueryRowContext(ctx, `
		INSERT INTO workspaces (owner_id, name, description, color)
		SELECT $2, name, description, color FROM workspaces WHERE id = $1 RETURNING id`,
		srcID, ownerID).Scan(&newID); err != nil {
		return 0, 0, notFound(err)
	}
	rows, err := s.q.QueryContext(ctx, `SELECT id FROM folders WHERE workspace_id = $1 ORDER BY created_at, id`, srcID)
	folders, err := collect(rows, err, func(sc scanner) (int64, error) {
		var id int64
		return id, sc.Scan(&id)
	})
	if err != nil {
		return 0, 0, err
	}
	for _, fid := range folders {
		_, n, err := s.CopyFolder(ctx, fid, newID)
		if err != nil {
			return 0, 0, err
		}
		cards += n
	}
	return newID, cards, nil
}

func (s *Store) CopyFolder(ctx context.Context, srcID, workspaceID int64) (newID int64, cards int, err error) {
	if err = s.q.QueryRowContext(ctx, `
		INSERT INTO folders (workspace_id, name, description)
		SELECT $2, name, description FROM folders WHERE id = $1 RETURNING id`,
		srcID, workspaceID).Scan(&newID); err != nil {
		return 0, 0, notFound(err)
	}
	res, err := s.q.ExecContext(ctx, `
		INSERT INTO cards (folder_id, ru_word, ru_example, en_word, en_example)
		SELECT $2, ru_word, ru_example, en_word, en_example FROM cards
		WHERE folder_id = $1 ORDER BY created_at, id`, srcID, newID)
	if err != nil {
		return 0, 0, err
	}
	n, err := res.RowsAffected()
	return newID, int(n), err
}

func (s *Store) CopyCard(ctx context.Context, srcID, folderID int64) (newID int64, err error) {
	err = s.q.QueryRowContext(ctx, `
		INSERT INTO cards (folder_id, ru_word, ru_example, en_word, en_example)
		SELECT $2, ru_word, ru_example, en_word, en_example FROM cards WHERE id = $1 RETURNING id`,
		srcID, folderID).Scan(&newID)
	return newID, notFound(err)
}
