package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/torresjeff/gallery/context"
	"github.com/torresjeff/gallery/models"
	"github.com/torresjeff/gallery/rand"
	"github.com/torresjeff/gallery/views"
)

type Users struct {
	SignUpView *views.View
	LoginView  *views.View
	us         models.UserService
}

type SignUpForm struct {
	Name     string `schema:"name"`
	Email    string `schema:"email"`
	Password string `schema:"password"`
}

type LoginForm struct {
	Email    string `schema:"email"`
	Password string `schema:"password"`
}

func NewUsers(us models.UserService) *Users {
	return &Users{
		SignUpView: views.NewView("bootstrap", "users/signup"),
		LoginView:  views.NewView("bootstrap", "users/login"),
		us:         us,
	}
}

func (u *Users) RenderSignUp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	u.SignUpView.Render(w, r, nil)
}

func (u *Users) RenderLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	redirectURI := r.URL.Query().Get("redirect")
	var data views.Data
	if redirectURI != "" {
		data = views.Data{
			Yield: redirectURI,
		}
	}
	u.LoginView.Render(w, r, data)
}

func (u *Users) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	var form LoginForm
	var vd views.Data
	if err := parseForm(r, &form); err != nil {
		vd.SetAlert(err)
		u.LoginView.Render(w, r, vd)
		return
	}

	// Since the plain text Remember Token is not saved in the database, whenever we retrieve a user, the RememberToken field will be empty
	user, err := u.us.Authenticate(form.Email, form.Password)
	if err != nil {
		switch err {
		case models.ErrNotFound:
			vd.AlertError("No user exists with that email address")
		default:
			vd.SetAlert(err)
		}
		u.LoginView.Render(w, r, vd)
		return
	}

	err = u.signIn(w, user)
	if err != nil {
		vd.SetAlert(err)
		u.LoginView.Render(w, r, vd)
		return
	}
	redirectURI := r.URL.Query().Get("redirect")
	if redirectURI != "" {
		redirectURI, err = url.QueryUnescape(redirectURI)
		if err != nil {
			http.Redirect(w, r, "/cookie", http.StatusFound)
			return
		}
		http.Redirect(w, r, redirectURI, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/galleries", http.StatusFound)

}

func (u *Users) signIn(w http.ResponseWriter, user *models.User) error {
	// Since the plain text Remember Token is not saved in the database, whenever we retrieve a user, the RememberToken field will be empty
	if user.RememberToken == "" {
		fmt.Println("Generating new remember token")
		token, err := rand.RememberToken()
		if err != nil {
			return err
		}
		user.RememberToken = token
		err = u.us.Update(user)
		if err != nil {
			return err
		}
	}
	cookie := http.Cookie{
		Name:     "remember_token",
		Value:    user.RememberToken,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	return nil
}

func (u *Users) Create(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var vd views.Data
	var form SignUpForm
	if err := parseForm(r, &form); err != nil {
		vd.SetAlert(err)
		u.SignUpView.Render(w, r, vd)
		return
	}

	user := models.User{
		Name:     form.Name,
		Email:    form.Email,
		Password: form.Password,
	}
	err := u.us.Create(&user)
	if err != nil {
		vd.SetAlert(err)
		u.SignUpView.Render(w, r, vd)
		return
	}

	// If we reached the signIn method, then we know the user was created successfully
	err = u.signIn(w, &user)
	if err != nil {
		// Since the user was created successfully, but we weren't able to sign him in, then just redirect him to the login page
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/galleries", http.StatusFound)
}

func (u *Users) CookieTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	cookie, err := r.Cookie("remember_token")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user, err := u.us.ByRememberToken(cookie.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, user)
}

func (u *Users) Logout(w http.ResponseWriter, r *http.Request) {
	// Expire the user's sessions cookie
	cookie := http.Cookie{
		Name:     "remember_token",
		Value:    "",
		Expires:  time.Now(),
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)

	// Update the user with a new remember token
	user := context.User(r.Context())
	token, _ := rand.RememberToken()
	user.RememberToken = token
	u.us.Update(user)

	// Send the user to the home pag
	http.Redirect(w, r, "/", http.StatusFound)
}
