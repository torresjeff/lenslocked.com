package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/torresjeff/gallery/context"
	"github.com/torresjeff/gallery/models"
)

// User middleware will lookup the current user via their
// remember_token cookie using the UserService. If the user
// is found, they will be set on the request context.
// Regardless, the next handler is always called.
type User struct {
	models.UserService
}

// RequireUser will redirect a user to the /login page
// if they are not logged in. This middleware assumes
// that User middleware has already been run, otherwise
// it will always redirect users.
type RequireUser struct{}

// Checks to see if a user is logged in and then either call next(w, r) if they are,
// or redirect them to the login page if they're not
func (mw *User) ApplyFn(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("remember_token")
		if err != nil {
			next(w, r)
			return
		}
		user, err := mw.UserService.ByRememberToken(cookie.Value)
		if err != nil {
			next(w, r)
			return
		}

		ctx := r.Context()
		ctx = context.WithUser(ctx, user)
		r = r.WithContext(ctx)

		next(w, r)
	})
}

// Apply is used for http.Handler, while ApplyFn is used for http.HandlerFunc
func (mw *User) Apply(next http.Handler) http.HandlerFunc {
	return mw.ApplyFn(next.ServeHTTP)
}

func (mw *RequireUser) Apply(next http.Handler) http.HandlerFunc {
	return mw.ApplyFn(next.ServeHTTP)
}

func (mw *RequireUser) ApplyFn(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// If the user is requesting a static asset or image we will not need to lookup the current user so we skip doing that.
		// Keep in mind this means that any user with the URl to an image can view it even if he's not logged in
		if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/images/") {
			next(w, r)
			return
		}
		user := context.User(r.Context())
		if user == nil {
			http.Redirect(w, r, "/login?redirect="+url.QueryEscape(r.URL.Path), http.StatusFound)
			return
		}
		next(w, r)
	})
}
