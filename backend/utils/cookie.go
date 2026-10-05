package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthCookieName(role string) string {
	return role + "_token"
}

func SetAuthCookie(c *gin.Context, role, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AuthCookieName(role), token, int(jwtManager().expiry.Seconds()), "/", "", false, true)
}

func ClearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	for _, role := range []string{"admin", "coach", "student", "super_admin"} {
		c.SetCookie(AuthCookieName(role), "", -1, "/", "", false, true)
	}
}
