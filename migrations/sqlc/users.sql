-- name: CreateUser :one
INSERT INTO users (email, password_hash, first_name, last_name, phone, role, auth_provider)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND is_active = TRUE;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND is_active = TRUE;

-- name: GetUserByGoogleSub :one
SELECT * FROM users WHERE google_sub = $1 AND is_active = TRUE;

-- name: UpdateUser :one
UPDATE users
SET first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    phone = COALESCE($4, phone),
    avatar_url = COALESCE($5, avatar_url),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: AttachGoogleSub :exec
UPDATE users SET google_sub = $2, auth_provider = 'google', updated_at = NOW() WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPasswordResetToken :one
SELECT * FROM password_reset_tokens WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW();

-- name: MarkPasswordResetTokenUsed :exec
UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1;

-- name: AddOrgMember :one
INSERT INTO organization_members (org_id, user_id, role, invited_by, joined_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (org_id, user_id) DO UPDATE SET role = EXCLUDED.role, is_active = TRUE, updated_at = NOW()
RETURNING *;

-- name: GetOrgMember :one
SELECT * FROM organization_members WHERE org_id = $1 AND user_id = $2 AND is_active = TRUE;

-- name: ListOrgMembers :many
SELECT om.*, u.email, u.first_name, u.last_name, u.avatar_url
FROM organization_members om
JOIN users u ON u.id = om.user_id
WHERE om.org_id = $1 AND om.is_active = TRUE
ORDER BY om.created_at ASC;

-- name: ListUserOrgs :many
SELECT o.*, om.role
FROM organizations o
JOIN organization_members om ON om.org_id = o.id
WHERE om.user_id = $1 AND om.is_active = TRUE AND o.is_active = TRUE
ORDER BY o.name ASC;

-- name: RemoveOrgMember :exec
UPDATE organization_members SET is_active = FALSE, updated_at = NOW()
WHERE org_id = $1 AND user_id = $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users WHERE is_active = TRUE;
