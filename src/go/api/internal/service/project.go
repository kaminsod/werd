package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrSlugTaken          = errors.New("slug already taken")
	ErrInvalidSlug        = errors.New("slug must be lowercase alphanumeric with hyphens, 2-64 chars")
	ErrNotProjectMember   = errors.New("not a project member")
	ErrInsufficientRole   = errors.New("insufficient role")
	ErrCannotDemoteOwner  = errors.New("cannot demote or remove the project owner")
	ErrAlreadyMember      = errors.New("user is already a project member")
	ErrUserNotFound       = errors.New("user not found")
	ErrCannotLeaveAsOwner = errors.New("owner cannot leave; transfer ownership first")
)

var slugRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`)

type ProjectInfo struct {
	ID        string
	Name      string
	Slug      string
	Settings  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MemberInfo struct {
	ProjectID string
	UserID    string
	Role      string
	Email     string
	Name      string
	CreatedAt time.Time
}

type Project struct {
	pool *pgxpool.Pool
	q    *storage.Queries
}

func NewProject(pool *pgxpool.Pool, q *storage.Queries) *Project {
	return &Project{pool: pool, q: q}
}

// Create creates a new project and makes the caller the owner (atomic).
func (s *Project) Create(ctx context.Context, userID, name, slug string) (*ProjectInfo, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if !slugRegexp.MatchString(slug) {
		return nil, ErrInvalidSlug
	}

	taken, err := s.q.ProjectExistsBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("checking slug: %w", err)
	}
	if taken {
		return nil, ErrSlugTaken
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	proj, err := qtx.CreateProject(ctx, storage.CreateProjectParams{
		Name:     name,
		Slug:     slug,
		Settings: []byte(`{}`),
	})
	if err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	_, err = qtx.CreateProjectMember(ctx, storage.CreateProjectMemberParams{
		ProjectID: proj.ID,
		UserID:    uid,
		Role:      storage.ProjectRoleOwner,
	})
	if err != nil {
		return nil, fmt.Errorf("adding owner membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return storageProjectToInfo(proj), nil
}

// List returns all projects the user is a member of.
func (s *Project) List(ctx context.Context, userID string) ([]ProjectInfo, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	projects, err := s.q.ListProjectsForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	result := make([]ProjectInfo, len(projects))
	for i, p := range projects {
		result[i] = *storageProjectToInfo(p)
	}
	return result, nil
}

// Get returns a single project by ID.
func (s *Project) Get(ctx context.Context, projectID string) (*ProjectInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	proj, err := s.q.GetProjectByID(ctx, pid)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	return storageProjectToInfo(proj), nil
}

// Update modifies a project's name, slug, and settings.
func (s *Project) Update(ctx context.Context, projectID, name, slug string, settings map[string]any) (*ProjectInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if !slugRegexp.MatchString(slug) {
		return nil, ErrInvalidSlug
	}

	taken, err := s.q.ProjectExistsBySlugExcludingID(ctx, storage.ProjectExistsBySlugExcludingIDParams{
		Slug: slug,
		ID:   pid,
	})
	if err != nil {
		return nil, fmt.Errorf("checking slug: %w", err)
	}
	if taken {
		return nil, ErrSlugTaken
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("marshaling settings: %w", err)
	}

	proj, err := s.q.UpdateProject(ctx, storage.UpdateProjectParams{
		ID:       pid,
		Name:     name,
		Slug:     slug,
		Settings: settingsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("updating project: %w", err)
	}

	return storageProjectToInfo(proj), nil
}

// Delete removes a project.
func (s *Project) Delete(ctx context.Context, projectID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrProjectNotFound
	}
	return s.q.DeleteProject(ctx, pid)
}

// ListMembers returns all members of a project with user details.
func (s *Project) ListMembers(ctx context.Context, projectID string) ([]MemberInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	rows, err := s.q.ListProjectMembers(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}

	result := make([]MemberInfo, len(rows))
	for i, r := range rows {
		result[i] = MemberInfo{
			ProjectID: r.ProjectID.String(),
			UserID:    r.UserID.String(),
			Role:      string(r.Role),
			Email:     r.UserEmail,
			Name:      r.UserName,
			CreatedAt: r.CreatedAt.Time,
		}
	}
	return result, nil
}

// AddMember adds a user to the project with the given role.
func (s *Project) AddMember(ctx context.Context, projectID, targetUserID, role string) (*MemberInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	tuid, err := uuid.Parse(targetUserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	pRole := storage.ProjectRole(role)
	if pRole != storage.ProjectRoleMember && pRole != storage.ProjectRoleViewer && pRole != storage.ProjectRoleAdmin {
		return nil, ErrInsufficientRole
	}

	targetUser, err := s.q.GetUserByID(ctx, tuid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	_, err = s.q.GetProjectMember(ctx, storage.GetProjectMemberParams{
		ProjectID: pid,
		UserID:    tuid,
	})
	if err == nil {
		return nil, ErrAlreadyMember
	}

	member, err := s.q.CreateProjectMember(ctx, storage.CreateProjectMemberParams{
		ProjectID: pid,
		UserID:    tuid,
		Role:      pRole,
	})
	if err != nil {
		return nil, fmt.Errorf("adding member: %w", err)
	}

	return &MemberInfo{
		ProjectID: member.ProjectID.String(),
		UserID:    member.UserID.String(),
		Role:      string(member.Role),
		Email:     targetUser.Email,
		Name:      targetUser.Name,
		CreatedAt: member.CreatedAt.Time,
	}, nil
}

// UpdateMemberRole changes a member's role.
func (s *Project) UpdateMemberRole(ctx context.Context, projectID, callerRole, targetUserID, newRole string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrProjectNotFound
	}

	tuid, err := uuid.Parse(targetUserID)
	if err != nil {
		return ErrNotProjectMember
	}

	newPRole := storage.ProjectRole(newRole)

	target, err := s.q.GetProjectMember(ctx, storage.GetProjectMemberParams{
		ProjectID: pid,
		UserID:    tuid,
	})
	if err != nil {
		return ErrNotProjectMember
	}

	if target.Role == storage.ProjectRoleOwner && callerRole != string(storage.ProjectRoleOwner) {
		return ErrCannotDemoteOwner
	}

	if newPRole == storage.ProjectRoleOwner && callerRole != string(storage.ProjectRoleOwner) {
		return ErrInsufficientRole
	}

	switch newPRole {
	case storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember, storage.ProjectRoleViewer:
	default:
		return fmt.Errorf("invalid role: %s", newRole)
	}

	return s.q.UpdateProjectMemberRole(ctx, storage.UpdateProjectMemberRoleParams{
		ProjectID: pid,
		UserID:    tuid,
		Role:      newPRole,
	})
}

// RemoveMember removes a member from the project.
func (s *Project) RemoveMember(ctx context.Context, projectID, callerID, callerRole, targetUserID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrProjectNotFound
	}

	tuid, err := uuid.Parse(targetUserID)
	if err != nil {
		return ErrNotProjectMember
	}

	isSelfLeave := callerID == targetUserID

	target, err := s.q.GetProjectMember(ctx, storage.GetProjectMemberParams{
		ProjectID: pid,
		UserID:    tuid,
	})
	if err != nil {
		return ErrNotProjectMember
	}

	if target.Role == storage.ProjectRoleOwner {
		if isSelfLeave {
			return ErrCannotLeaveAsOwner
		}
		return ErrCannotDemoteOwner
	}

	if !isSelfLeave && callerRole != string(storage.ProjectRoleOwner) && callerRole != string(storage.ProjectRoleAdmin) {
		return ErrInsufficientRole
	}

	return s.q.DeleteProjectMember(ctx, storage.DeleteProjectMemberParams{
		ProjectID: pid,
		UserID:    tuid,
	})
}

func storageProjectToInfo(p storage.Project) *ProjectInfo {
	var settings map[string]any
	if len(p.Settings) > 0 {
		json.Unmarshal(p.Settings, &settings)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	return &ProjectInfo{
		ID:        p.ID.String(),
		Name:      p.Name,
		Slug:      p.Slug,
		Settings:  settings,
		CreatedAt: p.CreatedAt.Time,
		UpdatedAt: p.UpdatedAt.Time,
	}
}
