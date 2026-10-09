package domain

import "time"

type User struct {
	ID           int64
	Email        string
	DisplayName  string
	PasswordHash string
	CreatedAt    time.Time
}

type Color string

const DefaultColor Color = "violet"

var colors = map[Color]bool{
	"violet": true, "indigo": true, "blue": true, "cyan": true,
	"magenta": true, "emerald": true, "amber": true, "rose": true,
}

func (c Color) Valid() bool { return colors[c] }

type Workspace struct {
	ID          int64
	OwnerID     int64
	Name        string
	Description string
	Color       Color
	FolderCount int
	CardCount   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkspaceFields struct {
	Name        string
	Description string
	Color       Color
}

type WorkspacePatch struct {
	Name        *string
	Description *string
	Color       *Color
}

type Folder struct {
	ID            int64
	WorkspaceID   int64
	WorkspaceName string
	Name          string
	Description   string
	CardCount     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type FolderFields struct {
	Name        string
	Description string
}

type FolderPatch struct {
	Name        *string
	Description *string
	WorkspaceID *int64
}

type Card struct {
	ID       int64
	FolderID int64
	CardFields
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CardFields struct {
	RuWord    string
	RuExample string
	EnWord    string
	EnExample string
}

const MaxCardsPerRequest = 5000

func (c CardFields) Normalize() (CardFields, error) {
	var err error
	if c.RuWord, err = Text("ru_word", c.RuWord, true, MaxWordLen); err != nil {
		return c, err
	}
	if c.EnWord, err = Text("en_word", c.EnWord, true, MaxWordLen); err != nil {
		return c, err
	}
	if c.RuExample, err = Text("ru_example", c.RuExample, false, MaxExampleLen); err != nil {
		return c, err
	}
	if c.EnExample, err = Text("en_example", c.EnExample, false, MaxExampleLen); err != nil {
		return c, err
	}
	return c, nil
}

type CardPatch struct {
	RuWord    *string
	RuExample *string
	EnWord    *string
	EnExample *string
	FolderID  *int64
}

type ResourceType string

const (
	ResourceWorkspace ResourceType = "workspace"
	ResourceFolder    ResourceType = "folder"
	ResourceCard      ResourceType = "card"
)

func (t ResourceType) Valid() bool {
	return t == ResourceWorkspace || t == ResourceFolder || t == ResourceCard
}

type ShareLink struct {
	Token        string
	OwnerID      int64
	ResourceType ResourceType
	ResourceID   int64
	ResourceName string
	CreatedAt    time.Time
}

type SharedFolder struct {
	ID          int64
	Name        string
	Description string
	Cards       []Card
}

type SharedWorkspace struct {
	ID          int64
	Name        string
	Description string
	Color       Color
	Folders     []SharedFolder
}

// Заполнено одно из полей Workspace, Folder, Card — в зависимости от типа ссылки.
type SharedContent struct {
	Link      ShareLink
	OwnerName string
	Workspace *SharedWorkspace
	Folder    *SharedFolder
	Card      *Card
}

type CopyResult struct {
	ResourceType ResourceType
	ID           int64
	CardsCopied  int
}

type StudyMode string

const (
	ModeLearn     StudyMode = "learn"
	ModeDictation StudyMode = "dictation"
	ModeMatch     StudyMode = "match"
)

var StudyModes = []StudyMode{ModeLearn, ModeDictation, ModeMatch}

func (m StudyMode) Valid() bool {
	return m == ModeLearn || m == ModeDictation || m == ModeMatch
}

type StudySession struct {
	ID          int64
	Mode        StudyMode
	WorkspaceID *int64
	FolderID    *int64
	Title       string
	TotalCards  int
	Mistakes    int
	Rounds      int
	Duration    time.Duration
	Completed   bool
	CreatedAt   time.Time
}

const (
	MaxSessionCards    = 100_000
	MaxSessionMistakes = 1_000_000
	MaxSessionDuration = 24 * time.Hour
)

type DayActivity struct {
	Day      string // YYYY-MM-DD в часовом поясе пользователя
	Sessions int
	Cards    int
}

type Stats struct {
	Workspaces     int
	Folders        int
	Cards          int
	Sessions       int
	CardsStudied   int
	MinutesStudied int
	Accuracy       float64 // доля от 0 до 1 по завершённым тренировкам
	StreakDays     int
	ByMode         map[StudyMode]int
	LastDays       []DayActivity
}
