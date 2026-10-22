package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthRoles is the complete set of session roles the app can issue. Keep it in
// one place so session discovery and cookie clearing cannot drift.
var AuthRoles = []string{"admin", "coach", "super_admin", "student"}

func AuthCookieName(role string) string {
	return role + "_token"
}

func SetAuthCookie(c *gin.Context, role, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AuthCookieName(role), token, int(jwtManager().expiry.Seconds()), "/", "", false, true)
}

func ClearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	for _, role := range AuthRoles {
		c.SetCookie(AuthCookieName(role), "", -1, "/", "", false, true)
	}
}
