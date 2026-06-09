package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/pkg/jwt"
	"github.com/readygeneration/readygeneration-backend/pkg/response"
)

const (
	CtxUserID = "user_id"
	CtxOrgID  = "org_id"
	CtxRole   = "role"
)

// Auth validates the Bearer JWT and sets user claims in context.
func Auth(jwtMgr *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Unauthorized(c, "invalid authorization header format")
			c.Abort()
			return
		}

		claims, err := jwtMgr.Verify(parts[1])
		if err != nil {
			response.Unauthorized(c, err.Error())
			c.Abort()
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxRole, claims.Role)
		if claims.OrgID != nil {
			c.Set(CtxOrgID, *claims.OrgID)
		}
		c.Next()
	}
}

// RequireRole enforces a minimum role level.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		if _, ok := allowed[role.(string)]; !ok {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetUserID extracts the authenticated user UUID from gin context.
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get(CtxUserID)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// GetOrgID extracts the active org UUID from gin context.
func GetOrgID(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get(CtxOrgID)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}
