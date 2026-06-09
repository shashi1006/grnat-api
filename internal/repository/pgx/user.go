package pgxrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

type userRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo creates a pgx-backed UserRepo.
func NewUserRepo(db *pgxpool.Pool) repository.UserRepo {
	return &userRepo{db: db}
}

const userCols = `id, email, password_hash, first_name, last_name, phone, avatar_url,
                  role, google_sub, auth_provider, email_verified, is_active, last_login_at,
                  created_at, updated_at`

func (r *userRepo) Create(ctx context.Context, p repository.CreateUserParams) (*domain.User, error) {
	const q = `INSERT INTO users (email, password_hash, first_name, last_name, phone, role, auth_provider)
	           VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING ` + userCols
	return scanUser(r.db.QueryRow(ctx, q,
		p.Email, p.PasswordHash, p.FirstName, p.LastName, p.Phone,
		string(p.Role), string(p.AuthProvider),
	))
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1 AND is_active=TRUE`, id))
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email=$1 AND is_active=TRUE`, email))
}

func (r *userRepo) GetByGoogleSub(ctx context.Context, sub string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE google_sub=$1 AND is_active=TRUE`, sub))
}

func (r *userRepo) Update(ctx context.Context, p repository.UpdateUserParams) (*domain.User, error) {
	const q = `UPDATE users SET first_name=COALESCE($2,first_name), last_name=COALESCE($3,last_name),
	           phone=COALESCE($4,phone), avatar_url=COALESCE($5,avatar_url), updated_at=NOW()
	           WHERE id=$1 RETURNING ` + userCols
	return scanUser(r.db.QueryRow(ctx, q, p.ID, p.FirstName, p.LastName, p.Phone, p.AvatarURL))
}

func (r *userRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash=$2, updated_at=NOW() WHERE id=$1`, id, hash)
	return err
}

func (r *userRepo) AttachGoogleSub(ctx context.Context, id uuid.UUID, sub string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET google_sub=$2, auth_provider='google', updated_at=NOW() WHERE id=$1`, id, sub)
	return err
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_login_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *userRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *userRepo) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, ttlSeconds int) error {
	expiresAt := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	_, err := r.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1,$2,$3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *userRepo) GetPasswordResetToken(ctx context.Context, tokenHash string) (*repository.PasswordResetToken, error) {
	var t repository.PasswordResetToken
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash FROM password_reset_tokens WHERE token_hash=$1 AND used_at IS NULL AND expires_at > NOW()`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash)
	if err != nil {
		return nil, fmt.Errorf("get reset token: %w", err)
	}
	return &t, nil
}

func (r *userRepo) MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE password_reset_tokens SET used_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *userRepo) AddOrgMember(ctx context.Context, orgID, userID uuid.UUID, role domain.OrgMemberRole, invitedBy *uuid.UUID) (*domain.OrganizationMember, error) {
	const q = `INSERT INTO organization_members (org_id, user_id, role, invited_by, joined_at)
	           VALUES ($1,$2,$3,$4,NOW())
	           ON CONFLICT (org_id, user_id) DO UPDATE SET role=EXCLUDED.role, is_active=TRUE, updated_at=NOW()
	           RETURNING id, org_id, user_id, role, is_active, invited_by, joined_at, created_at, updated_at`
	var m domain.OrganizationMember
	var roleStr string
	err := r.db.QueryRow(ctx, q, orgID, userID, string(role), invitedBy).
		Scan(&m.ID, &m.OrgID, &m.UserID, &roleStr, &m.IsActive, &m.InvitedBy, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("add org member: %w", err)
	}
	m.Role = domain.OrgMemberRole(roleStr)
	return &m, nil
}

func (r *userRepo) GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrganizationMember, error) {
	var m domain.OrganizationMember
	var roleStr string
	err := r.db.QueryRow(ctx,
		`SELECT id, org_id, user_id, role, is_active, invited_by, joined_at, created_at, updated_at
		 FROM organization_members WHERE org_id=$1 AND user_id=$2 AND is_active=TRUE`,
		orgID, userID,
	).Scan(&m.ID, &m.OrgID, &m.UserID, &roleStr, &m.IsActive, &m.InvitedBy, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get org member: %w", err)
	}
	m.Role = domain.OrgMemberRole(roleStr)
	return &m, nil
}

func (r *userRepo) ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]*repository.OrgMemberWithUser, error) {
	rows, err := r.db.Query(ctx,
		`SELECT om.id, om.org_id, om.user_id, om.role, om.is_active, om.invited_by, om.joined_at, om.created_at, om.updated_at,
		        u.email, u.first_name, u.last_name, u.avatar_url
		 FROM organization_members om JOIN users u ON u.id=om.user_id
		 WHERE om.org_id=$1 AND om.is_active=TRUE ORDER BY om.created_at ASC`, orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list org members: %w", err)
	}
	defer rows.Close()
	var members []*repository.OrgMemberWithUser
	for rows.Next() {
		var m repository.OrgMemberWithUser
		var roleStr string
		err := rows.Scan(
			&m.ID, &m.OrgID, &m.UserID, &roleStr, &m.IsActive, &m.InvitedBy, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
			&m.Email, &m.FirstName, &m.LastName, &m.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		m.Role = domain.OrgMemberRole(roleStr)
		members = append(members, &m)
	}
	return members, rows.Err()
}

func (r *userRepo) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*repository.OrgWithRole, error) {
	rows, err := r.db.Query(ctx,
		`SELECT o.id, o.name, o.slug, o.ein, o.org_type, o.mission, o.address_line1, o.address_line2,
		        o.city, o.state, o.zip, o.county, o.website, o.phone, o.logo_url, o.plan, o.plan_expires_at,
		        o.is_active, o.created_at, o.updated_at, om.role
		 FROM organizations o JOIN organization_members om ON om.org_id=o.id
		 WHERE om.user_id=$1 AND om.is_active=TRUE AND o.is_active=TRUE ORDER BY o.name ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list user orgs: %w", err)
	}
	defer rows.Close()
	var orgs []*repository.OrgWithRole
	for rows.Next() {
		var o repository.OrgWithRole
		var orgType, plan, roleStr string
		err := rows.Scan(
			&o.ID, &o.Name, &o.Slug, &o.EIN, &orgType, &o.Mission,
			&o.AddressLine1, &o.AddressLine2, &o.City, &o.State, &o.Zip, &o.County,
			&o.Website, &o.Phone, &o.LogoURL, &plan, &o.PlanExpiresAt,
			&o.IsActive, &o.CreatedAt, &o.UpdatedAt, &roleStr,
		)
		if err != nil {
			return nil, err
		}
		o.OrgType = domain.OrgType(orgType)
		o.Plan = domain.Plan(plan)
		o.Role = domain.OrgMemberRole(roleStr)
		orgs = append(orgs, &o)
	}
	return orgs, rows.Err()
}

func (r *userRepo) RemoveOrgMember(ctx context.Context, orgID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE organization_members SET is_active=FALSE, updated_at=NOW() WHERE org_id=$1 AND user_id=$2`,
		orgID, userID,
	)
	return err
}

func (r *userRepo) SaveQuizDraft(ctx context.Context, userID uuid.UUID, data []byte) error {
	const q = `INSERT INTO quiz_drafts (user_id, data, updated_at)
	           VALUES ($1, $2, NOW())
	           ON CONFLICT (user_id) DO UPDATE SET data = $2, updated_at = NOW()`
	_, err := r.db.Exec(ctx, q, userID, data)
	return err
}

func (r *userRepo) GetQuizDraft(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	const q = `SELECT data FROM quiz_drafts WHERE user_id = $1`
	var data []byte
	err := r.db.QueryRow(ctx, q, userID).Scan(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *userRepo) DeleteQuizDraft(ctx context.Context, userID uuid.UUID) error {
	const q = `DELETE FROM quiz_drafts WHERE user_id = $1`
	_, err := r.db.Exec(ctx, q, userID)
	return err
}

func scanUser(row scannable) (*domain.User, error) {
	var u domain.User
	var role, authProvider string
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Phone, &u.AvatarURL,
		&role, &u.GoogleSub, &authProvider, &u.EmailVerified, &u.IsActive, &u.LastLoginAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Role = domain.UserRole(role)
	u.AuthProvider = domain.AuthProvider(authProvider)
	return &u, nil
}
