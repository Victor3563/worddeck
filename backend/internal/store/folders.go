package store

import (
	"context"

	"github.com/worddeck/internal/domain"
)

const folderSelect = `
	SELECT f.id, f.workspace_id, w.name, f.name, f.description,
	       (SELECT count(*) FROM cards c WHERE c.folder_id = f.id),
	       f.created_at, f.updated_at
	FROM folders f JOIN workspaces w ON w.id = f.workspace_id`

func scanFolder(sc scanner) (domain.Folder, error) {
	var f domain.Folder
	err := sc.Scan(&f.ID, &f.WorkspaceID, &f.WorkspaceName, &f.Name, &f.Description,
		&f.CardCount, &f.CreatedAt, &f.UpdatedAt)
	return f, notFound(err)
}

func (s *Store) CheckFolderOwner(ctx context.Context, ownerID, folderID int64) error {
	var ok bool
	if err := s.q.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM folders f JOIN workspaces w ON w.id = f.workspace_id
		               WHERE f.id = $1 AND w.owner_id = $2)`, folderID, ownerID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListFolders(ctx context.Context, ownerID, workspaceID int64) ([]domain.Folder, error) {
	if err := s.CheckWorkspaceOwner(ctx, ownerID, workspaceID); err != nil {
		return nil, err
	}
	rows, err := s.q.QueryContext(ctx, folderSelect+` WHERE f.workspace_id = $1 ORDER BY f.created_at, f.id`, workspaceID)
	return collect(rows, err, scanFolder)
}

func (s *Store) GetFolder(ctx context.Context, ownerID, id int64) (domain.Folder, error) {
	return scanFolder(s.q.QueryRowContext(ctx, folderSelect+` WHERE f.id = $1 AND w.owner_id = $2`, id, ownerID))
}

func (s *Store) CreateFolder(ctx context.Context, ownerID, workspaceID int64, in domain.FolderFields) (domain.Folder, error) {
	var id int64
	err := s.q.QueryRowContext(ctx, `
		INSERT INTO folders (workspace_id, name, description)
		SELECT w.id, $3, $4 FROM workspaces w WHERE w.id = $1 AND w.owner_id = $2
		RETURNING id`, workspaceID, ownerID, in.Name, in.Description).Scan(&id)
	if err != nil {
		return domain.Folder{}, notFound(err)
	}
	return s.GetFolder(ctx, ownerID, id)
}

func (s *Store) UpdateFolder(ctx context.Context, ownerID, id int64, p domain.FolderPatch) (domain.Folder, error) {
	if p.WorkspaceID != nil {
		if err := s.CheckWorkspaceOwner(ctx, ownerID, *p.WorkspaceID); err != nil {
			return domain.Folder{}, err
		}
	}
	err := affected(s.q.ExecContext(ctx, `
		UPDATE folders f SET
			name         = COALESCE($3, f.name),
			description  = COALESCE($4, f.description),
			workspace_id = COALESCE($5, f.workspace_id),
			updated_at   = now()
		FROM workspaces w
		WHERE f.id = $1 AND w.id = f.workspace_id AND w.owner_id = $2`,
		id, ownerID, p.Name, p.Description, p.WorkspaceID))
	if err != nil {
		return domain.Folder{}, err
	}
	return s.GetFolder(ctx, ownerID, id)
}

func (s *Store) DeleteFolder(ctx context.Context, ownerID, id int64) error {
	return affected(s.q.ExecContext(ctx, `
		DELETE FROM folders f USING workspaces w
		WHERE f.id = $1 AND w.id = f.workspace_id AND w.owner_id = $2`, id, ownerID))
}
