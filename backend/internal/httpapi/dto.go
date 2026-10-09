package httpapi

import (
	"time"

	"github.com/worddeck/internal/cardio"
	"github.com/worddeck/internal/domain"
	"github.com/worddeck/internal/service"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

type statusResponse struct {
	Status string `json:"status"`
}

type limitsResponse struct {
	DisplayName     int   `json:"display_name"`
	Email           int   `json:"email"`
	Name            int   `json:"name"`
	Description     int   `json:"description"`
	Word            int   `json:"word"`
	Example         int   `json:"example"`
	SessionTitle    int   `json:"session_title"`
	PasswordMin     int   `json:"password_min"`
	PasswordMax     int   `json:"password_max"`
	CardsPerRequest int   `json:"cards_per_request"`
	ImportBytes     int64 `json:"import_bytes"`
}

type deletedResponse struct {
	Deleted int64 `json:"deleted"`
}

func mapSlice[T, R any](in []T, f func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

type userJSON struct {
	ID          int64     `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

func toUser(u domain.User) userJSON {
	return userJSON{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, CreatedAt: u.CreatedAt}
}

type sessionResponse struct {
	User      userJSON  `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

type workspaceJSON struct {
	ID          int64     `json:"id"`
	OwnerID     int64     `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	FolderCount int       `json:"folder_count"`
	CardCount   int       `json:"card_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toWorkspace(w domain.Workspace) workspaceJSON {
	return workspaceJSON{
		ID: w.ID, OwnerID: w.OwnerID, Name: w.Name, Description: w.Description, Color: string(w.Color),
		FolderCount: w.FolderCount, CardCount: w.CardCount, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}
}

type folderJSON struct {
	ID            int64     `json:"id"`
	WorkspaceID   int64     `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	CardCount     int       `json:"card_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toFolder(f domain.Folder) folderJSON {
	return folderJSON{
		ID: f.ID, WorkspaceID: f.WorkspaceID, WorkspaceName: f.WorkspaceName, Name: f.Name,
		Description: f.Description, CardCount: f.CardCount, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
	}
}

type cardFieldsJSON struct {
	RuWord    string `json:"ru_word"`
	RuExample string `json:"ru_example"`
	EnWord    string `json:"en_word"`
	EnExample string `json:"en_example"`
}

func (c cardFieldsJSON) domain() domain.CardFields {
	return domain.CardFields{RuWord: c.RuWord, RuExample: c.RuExample, EnWord: c.EnWord, EnExample: c.EnExample}
}

func toCardFields(c domain.CardFields) cardFieldsJSON {
	return cardFieldsJSON{RuWord: c.RuWord, RuExample: c.RuExample, EnWord: c.EnWord, EnExample: c.EnExample}
}

type cardJSON struct {
	ID       int64 `json:"id"`
	FolderID int64 `json:"folder_id"`
	cardFieldsJSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toCard(c domain.Card) cardJSON {
	return cardJSON{ID: c.ID, FolderID: c.FolderID, cardFieldsJSON: toCardFields(c.CardFields), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

type lineErrorJSON struct {
	Line   int              `json:"line"`
	Code   string           `json:"code"`
	Field  string           `json:"field"`
	Params *errorParamsJSON `json:"params,omitempty"`
}

type importResponse struct {
	Format    string           `json:"format"`
	Delimiter string           `json:"delimiter,omitempty"`
	Parsed    int              `json:"parsed"`
	Errors    []lineErrorJSON  `json:"errors"`
	DryRun    bool             `json:"dry_run"`
	Cards     []cardFieldsJSON `json:"cards,omitempty"` // только при dry_run
	Created   int              `json:"created"`
}

func toImport(r service.ImportResult) importResponse {
	out := importResponse{
		Format: r.Format, Delimiter: r.Delimiter, Parsed: len(r.Cards), DryRun: r.DryRun, Created: r.Created,
		Errors: mapSlice(r.Errors, func(e cardio.LineError) lineErrorJSON {
			return lineErrorJSON{Line: e.Line, Code: e.Error.Code, Field: e.Error.Field, Params: toErrorParams(e.Error.Params)}
		}),
	}
	if r.DryRun {
		out.Cards = mapSlice(r.Cards, toCardFields)
	}
	return out
}

type shareJSON struct {
	Token        string    `json:"token"`
	ResourceType string    `json:"resource_type"`
	ResourceID   int64     `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	CreatedAt    time.Time `json:"created_at"`
}

func toShare(l domain.ShareLink) shareJSON {
	return shareJSON{Token: l.Token, ResourceType: string(l.ResourceType), ResourceID: l.ResourceID,
		ResourceName: l.ResourceName, CreatedAt: l.CreatedAt}
}

type sharedFolderJSON struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Cards       []cardJSON `json:"cards"`
}

func toSharedFolder(f domain.SharedFolder) sharedFolderJSON {
	return sharedFolderJSON{ID: f.ID, Name: f.Name, Description: f.Description, Cards: mapSlice(f.Cards, toCard)}
}

type sharedWorkspaceJSON struct {
	ID          int64              `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Color       string             `json:"color"`
	Folders     []sharedFolderJSON `json:"folders"`
}

type sharedContentJSON struct {
	Token        string               `json:"token"`
	ResourceType string               `json:"resource_type"`
	OwnerName    string               `json:"owner_name"`
	IsOwner      bool                 `json:"is_owner"`
	Workspace    *sharedWorkspaceJSON `json:"workspace,omitempty"`
	Folder       *sharedFolderJSON    `json:"folder,omitempty"`
	Card         *cardJSON            `json:"card,omitempty"`
}

func toSharedContent(c domain.SharedContent, viewerID int64) sharedContentJSON {
	out := sharedContentJSON{
		Token: c.Link.Token, ResourceType: string(c.Link.ResourceType), OwnerName: c.OwnerName,
		IsOwner: viewerID != 0 && viewerID == c.Link.OwnerID,
	}
	switch {
	case c.Workspace != nil:
		w := c.Workspace
		out.Workspace = &sharedWorkspaceJSON{ID: w.ID, Name: w.Name, Description: w.Description,
			Color: string(w.Color), Folders: mapSlice(w.Folders, toSharedFolder)}
	case c.Folder != nil:
		f := toSharedFolder(*c.Folder)
		out.Folder = &f
	case c.Card != nil:
		card := toCard(*c.Card)
		out.Card = &card
	}
	return out
}

type copyResponse struct {
	ResourceType string `json:"resource_type"`
	ID           int64  `json:"id"`
	CardsCopied  int    `json:"cards_copied"`
}

type studySessionJSON struct {
	ID          int64     `json:"id"`
	Mode        string    `json:"mode"`
	WorkspaceID *int64    `json:"workspace_id"`
	FolderID    *int64    `json:"folder_id"`
	Title       string    `json:"title"`
	TotalCards  int       `json:"total_cards"`
	Mistakes    int       `json:"mistakes"`
	Rounds      int       `json:"rounds"`
	DurationMs  int64     `json:"duration_ms"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

func toStudySession(s domain.StudySession) studySessionJSON {
	return studySessionJSON{
		ID: s.ID, Mode: string(s.Mode), WorkspaceID: s.WorkspaceID, FolderID: s.FolderID, Title: s.Title,
		TotalCards: s.TotalCards, Mistakes: s.Mistakes, Rounds: s.Rounds,
		DurationMs: s.Duration.Milliseconds(), Completed: s.Completed, CreatedAt: s.CreatedAt,
	}
}

type historyResponse struct {
	Items  []studySessionJSON `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type dayActivityJSON struct {
	Day      string `json:"day"`
	Sessions int    `json:"sessions"`
	Cards    int    `json:"cards"`
}

type statsResponse struct {
	Workspaces     int               `json:"workspaces"`
	Folders        int               `json:"folders"`
	Cards          int               `json:"cards"`
	Sessions       int               `json:"sessions"`
	CardsStudied   int               `json:"cards_studied"`
	MinutesStudied int               `json:"minutes_studied"`
	Accuracy       float64           `json:"accuracy"`
	StreakDays     int               `json:"streak_days"`
	ByMode         map[string]int    `json:"by_mode"`
	LastDays       []dayActivityJSON `json:"last_days"`
}

func toStats(s domain.Stats) statsResponse {
	out := statsResponse{
		Workspaces: s.Workspaces, Folders: s.Folders, Cards: s.Cards, Sessions: s.Sessions,
		CardsStudied: s.CardsStudied, MinutesStudied: s.MinutesStudied, Accuracy: s.Accuracy,
		StreakDays: s.StreakDays, ByMode: make(map[string]int, len(s.ByMode)),
		LastDays: mapSlice(s.LastDays, func(d domain.DayActivity) dayActivityJSON {
			return dayActivityJSON{Day: d.Day, Sessions: d.Sessions, Cards: d.Cards}
		}),
	}
	for m, n := range s.ByMode {
		out.ByMode[string(m)] = n
	}
	return out
}
