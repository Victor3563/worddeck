package store

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/lib/pq"

	"github.com/worddeck/internal/domain"
)

const cardCols = `c.id, c.folder_id, c.ru_word, c.ru_example, c.en_word, c.en_example, c.created_at, c.updated_at`

func scanCard(sc scanner) (domain.Card, error) {
	var c domain.Card
	err := sc.Scan(&c.ID, &c.FolderID, &c.RuWord, &c.RuExample, &c.EnWord, &c.EnExample, &c.CreatedAt, &c.UpdatedAt)
	return c, notFound(err)
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (s *Store) ListCards(ctx context.Context, ownerID, folderID int64, search string) ([]domain.Card, error) {
	if err := s.CheckFolderOwner(ctx, ownerID, folderID); err != nil {
		return nil, err
	}
	q := `SELECT ` + cardCols + ` FROM cards c WHERE c.folder_id = $1`
	args := []any{folderID}
	if search != "" {
		q += ` AND (c.ru_word ILIKE $2 OR c.en_word ILIKE $2 OR c.ru_example ILIKE $2 OR c.en_example ILIKE $2)`
		args = append(args, "%"+escapeLike(search)+"%")
	}
	rows, err := s.q.QueryContext(ctx, q+` ORDER BY c.created_at, c.id`, args...)
	return collect(rows, err, scanCard)
}

func (s *Store) ListWorkspaceCards(ctx context.Context, ownerID, workspaceID int64) ([]domain.Card, error) {
	if err := s.CheckWorkspaceOwner(ctx, ownerID, workspaceID); err != nil {
		return nil, err
	}
	rows, err := s.q.QueryContext(ctx, `
		SELECT `+cardCols+` FROM cards c JOIN folders f ON f.id = c.folder_id
		WHERE f.workspace_id = $1 ORDER BY f.id, c.created_at, c.id`, workspaceID)
	return collect(rows, err, scanCard)
}

func (s *Store) GetCard(ctx context.Context, ownerID, id int64) (domain.Card, error) {
	return scanCard(s.q.QueryRowContext(ctx, `
		SELECT `+cardCols+` FROM cards c
		JOIN folders f ON f.id = c.folder_id JOIN workspaces w ON w.id = f.workspace_id
		WHERE c.id = $1 AND w.owner_id = $2`, id, ownerID))
}

func (s *Store) CreateCards(ctx context.Context, ownerID, folderID int64, in []domain.CardFields) ([]domain.Card, error) {
	var out []domain.Card
	err := s.InTx(ctx, func(tx *Store) error {
		if err := tx.CheckFolderOwner(ctx, ownerID, folderID); err != nil {
			return err
		}
		ru, ruEx, en, enEx := make([]string, len(in)), make([]string, len(in)), make([]string, len(in)), make([]string, len(in))
		for i, c := range in {
			ru[i], ruEx[i], en[i], enEx[i] = c.RuWord, c.RuExample, c.EnWord, c.EnExample
		}
		rows, err := tx.q.QueryContext(ctx, `
			INSERT INTO cards AS c (folder_id, ru_word, ru_example, en_word, en_example)
			SELECT $1, ru, ru_ex, en, en_ex
			FROM unnest($2::text[], $3::text[], $4::text[], $5::text[]) WITH ORDINALITY AS t(ru, ru_ex, en, en_ex, n)
			ORDER BY n
			RETURNING `+cardCols,
			folderID, pq.Array(ru), pq.Array(ruEx), pq.Array(en), pq.Array(enEx))
		if out, err = collect(rows, err, scanCard); err != nil {
			return err
		}
		// RETURNING не гарантирует порядок, а id выдаются в порядке вставки.
		slices.SortFunc(out, func(a, b domain.Card) int { return cmp.Compare(a.ID, b.ID) })
		_, err = tx.q.ExecContext(ctx, `UPDATE folders SET updated_at = now() WHERE id = $1`, folderID)
		return err
	})
	return out, err
}

func (s *Store) UpdateCard(ctx context.Context, ownerID, id int64, p domain.CardPatch) (domain.Card, error) {
	if p.FolderID != nil {
		if err := s.CheckFolderOwner(ctx, ownerID, *p.FolderID); err != nil {
			return domain.Card{}, err
		}
	}
	err := affected(s.q.ExecContext(ctx, `
		UPDATE cards c SET
			ru_word    = COALESCE($3, c.ru_word),
			ru_example = COALESCE($4, c.ru_example),
			en_word    = COALESCE($5, c.en_word),
			en_example = COALESCE($6, c.en_example),
			folder_id  = COALESCE($7, c.folder_id),
			updated_at = now()
		FROM folders f, workspaces w
		WHERE c.id = $1 AND f.id = c.folder_id AND w.id = f.workspace_id AND w.owner_id = $2`,
		id, ownerID, p.RuWord, p.RuExample, p.EnWord, p.EnExample, p.FolderID))
	if err != nil {
		return domain.Card{}, err
	}
	return s.GetCard(ctx, ownerID, id)
}

func (s *Store) DeleteCard(ctx context.Context, ownerID, id int64) error {
	return affected(s.q.ExecContext(ctx, `
		DELETE FROM cards c USING folders f, workspaces w
		WHERE c.id = $1 AND f.id = c.folder_id AND w.id = f.workspace_id AND w.owner_id = $2`, id, ownerID))
}

func (s *Store) DeleteCards(ctx context.Context, ownerID, folderID int64, ids []int64) (int64, error) {
	if err := s.CheckFolderOwner(ctx, ownerID, folderID); err != nil {
		return 0, err
	}
	res, err := s.q.ExecContext(ctx, `DELETE FROM cards WHERE folder_id = $1 AND id = ANY($2)`, folderID, pq.Array(ids))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
