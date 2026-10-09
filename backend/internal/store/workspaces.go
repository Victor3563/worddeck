package store

import (
	"context"

	"github.com/worddeck/internal/domain"
)

const workspaceSelect = `
	SELECT w.id, w.owner_id, w.name, w.description, w.color,
	       (SELECT count(*) FROM folders f WHERE f.workspace_id = w.id),
	       (SELECT count(*) FROM cards c JOIN folders f ON f.id = c.folder_id WHERE f.workspace_id = w.id),
	       w.created_at, w.updated_at
	FROM workspaces w`

func scanWorkspace(sc scanner) (domain.Workspace, error) {
	var w domain.Workspace
	err := sc.Scan(&w.ID, &w.OwnerID, &w.Name, &w.Description, &w.Color,
		&w.FolderCount, &w.CardCount, &w.CreatedAt, &w.UpdatedAt)
	return w, notFound(err)
}

func (s *Store) CheckWorkspaceOwner(ctx context.Context, ownerID, workspaceID int64) error {
	var ok bool
	if err := s.q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM workspaces WHERE id = $1 AND owner_id = $2)`,
		workspaceID, ownerID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListWorkspaces(ctx context.Context, ownerID int64) ([]domain.Workspace, error) {
	rows, err := s.q.QueryContext(ctx, workspaceSelect+` WHERE w.owner_id = $1 ORDER BY w.created_at, w.id`, ownerID)
	return collect(rows, err, scanWorkspace)
}

func (s *Store) GetWorkspace(ctx context.Context, ownerID, id int64) (domain.Workspace, error) {
	return scanWorkspace(s.q.QueryRowContext(ctx, workspaceSelect+` WHERE w.id = $1 AND w.owner_id = $2`, id, ownerID))
}

func (s *Store) CreateWorkspace(ctx context.Context, ownerID int64, in domain.WorkspaceFields) (domain.Workspace, error) {
	var id int64
	if err := s.q.QueryRowContext(ctx, `
		INSERT INTO workspaces (owner_id, name, description, color) VALUES ($1, $2, $3, $4) RETURNING id`,
		ownerID, in.Name, in.Description, in.Color).Scan(&id); err != nil {
		return domain.Workspace{}, err
	}
	return s.GetWorkspace(ctx, ownerID, id)
}

func (s *Store) UpdateWorkspace(ctx context.Context, ownerID, id int64, p domain.WorkspacePatch) (domain.Workspace, error) {
	err := affected(s.q.ExecContext(ctx, `
		UPDATE workspaces SET
			name        = COALESCE($3, name),
			description = COALESCE($4, description),
			color       = COALESCE($5, color),
			updated_at  = now()
		WHERE id = $1 AND owner_id = $2`,
		id, ownerID, p.Name, p.Description, p.Color))
	if err != nil {
		return domain.Workspace{}, err
	}
	return s.GetWorkspace(ctx, ownerID, id)
}

func (s *Store) DeleteWorkspace(ctx context.Context, ownerID, id int64) error {
	return affected(s.q.ExecContext(ctx, `DELETE FROM workspaces WHERE id = $1 AND owner_id = $2`, id, ownerID))
}
