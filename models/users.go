package models

import (
	"regexp"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/torresjeff/gallery/hash"
	"github.com/torresjeff/gallery/rand"
	"golang.org/x/crypto/bcrypt"
)

const (
	// ErrNotFound is returned when a resource cannot be found in the DB.
	ErrNotFound modelError = "models: resource not found"
	// ErrIDInvalid is returned when an invalid ID is provided to a method like DELETE.
	ErrIDInvalid modelError = "models: ID provided was invalid"
	// ErrEmailPasswordIncorrect is returned when an invalid password is supplied
	ErrEmailPasswordIncorrect modelError = "models: invalid email and/or password"
	// ErrEmailRequired is returned when an email address is not provided when creating/updating a user
	ErrEmailRequired modelError = "models: email address is required"
	// ErrEmailNotValid is returned when an email address is not provided when creating/updating a user
	ErrEmailNotValid modelError = "models: email address is not valid"
	// ErrEmailTaken is returned when an update or create is attempted with an email address that is already in use.
	ErrEmailTaken modelError = "models: email address is already taken"
	// ErrPasswordTooShort is returned when a user tries to set a password that is less than 8 characters long
	ErrPasswordTooShort modelError = "models: password must be at least 8 characters long"
	// ErrPasswordRequired is returned when a create is attempted without a user password provided.
	ErrPasswordRequired modelError = "models: password is required"
	// ErrRememberTokenHashRequired is returned when a create or update is attempted without a user remember token hash
	ErrRememberTokenHashRequired modelError = "models: remember token is required"
	// ErrRememberTokenTooShort is returned when a remember token is not at least 32 bytes
	ErrRememberTokenTooShort modelError = "models: remember token must be at least 32 bytes"
)

type UserDB interface {
	// Querying single users
	ById(id uint) (*User, error)
	ByEmail(email string) (*User, error)
	ByRememberToken(token string) (*User, error)

	// Methods for altering users
	Create(*User) error
	Update(*User) error
	Delete(id uint) error
}

type modelError string

func (e modelError) Error() string {
	return string(e)
}

func (e modelError) Public() string {
	s := strings.Replace(string(e), "models: ", "", 1)
	split := strings.Split(string(s), " ")
	split[0] = strings.Title(split[0])

	return strings.Join(split, " ")
}

// userGorm represents our database interaction layer
// and implements the UserDB interface fully.
type userGorm struct {
	db *gorm.DB
}

type User struct {
	gorm.Model
	Name              string
	Email             string `gorm:"not null;unique_index"`
	Password          string `gorm:"-"`
	PasswordHash      string `gorm:"not null"`
	RememberToken     string `gorm:"-"`
	RememberTokenHash string `gorm:"not null; unique_index"`
}

type userValidator struct {
	UserDB
	hmac       hash.HMAC
	pepper     string
	emailRegex *regexp.Regexp
}

type userValidatorFunction func(*User) error

type UserService interface {
	UserDB
	Authenticate(string, string) (*User, error)
}

type userService struct {
	UserDB
	pepper string
}

func NewUserService(db *gorm.DB, pepper, hmacKey string) UserService {
	ug := &userGorm{db}
	hmac := hash.NewHMAC(hmacKey)
	uv := newUserValidator(ug, hmac, pepper)
	return &userService{
		UserDB: uv,
		pepper: pepper,
	}
}

func newUserValidator(udb UserDB, hmac hash.HMAC, pepper string) *userValidator {
	return &userValidator{
		UserDB:     udb,
		hmac:       hmac,
		pepper:     pepper,
		emailRegex: regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,16}$`),
	}
}

func (us *userService) Authenticate(email, password string) (*User, error) {
	foundUser, err := us.ByEmail(email)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(password+us.pepper))
	if err == nil {
		return foundUser, nil
	}
	switch err {
	case bcrypt.ErrMismatchedHashAndPassword:
		return nil, ErrEmailPasswordIncorrect
	default:
		return nil, err
	}
}

func (ug *userGorm) Create(u *User) error {
	return ug.db.Create(u).Error
}

func (ug *userGorm) ById(id uint) (*User, error) {
	var user User
	err := first(ug.db.Where("id = ?", id), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (ug *userGorm) ByEmail(email string) (*User, error) {
	var user User
	err := first(ug.db.Where("email = ?", email), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (ug *userGorm) ByRememberToken(rememberHash string) (*User, error) {
	var user User
	err := first(ug.db.Where("remember_token_hash = ?", rememberHash), &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (ug *userGorm) Update(u *User) error {
	return ug.db.Save(u).Error
}

func (ug *userGorm) Delete(id uint) error {
	return ug.db.Delete(&User{Model: gorm.Model{ID: id}}).Error
}

// USER VALIDATOR METHODS
func (uv *userValidator) Create(user *User) error {
	err := runUserValidatorFunctions(user,
		uv.passwordRequired,
		uv.passwordMinLength,
		uv.bcryptPassword,
		uv.passwordHashRequired,
		uv.setRememberTokenIfUnset,
		uv.rememberTokenMinLength,
		uv.hmacRememberToken,
		uv.rememberTokenHashRequired,
		uv.normalizeEmail,
		uv.requireEmail,
		uv.emailFormat,
		uv.emailIsAvailable)

	if err != nil {
		return err
	}

	return uv.UserDB.Create(user)
}

func (uv *userValidator) ByEmail(email string) (*User, error) {
	user := User{
		Email: email,
	}
	if err := runUserValidatorFunctions(&user, uv.normalizeEmail); err != nil {
		return nil, err
	}
	return uv.UserDB.ByEmail(user.Email)

}

func (uv *userValidator) Update(user *User) error {
	err := runUserValidatorFunctions(user,
		uv.passwordMinLength,
		uv.bcryptPassword,
		uv.passwordHashRequired,
		uv.rememberTokenMinLength,
		uv.hmacRememberToken,
		uv.rememberTokenHashRequired,
		uv.normalizeEmail,
		uv.requireEmail,
		uv.emailFormat,
		uv.emailIsAvailable)
	if err != nil {
		return err
	}

	return uv.UserDB.Update(user)
}

func (uv *userValidator) Delete(id uint) error {
	user := &User{}
	user.ID = id
	if err := runUserValidatorFunctions(user, uv.idGreaterThan(0)); err != nil {
		return err
	}
	return uv.UserDB.Delete(id)
}

func (uv *userValidator) ByRememberToken(token string) (*User, error) {
	user := &User{
		RememberToken: token,
	}
	if err := runUserValidatorFunctions(user, uv.hmacRememberToken); err != nil {
		return nil, err
	}
	return uv.UserDB.ByRememberToken(user.RememberTokenHash)
}

func (uv *userValidator) bcryptPassword(user *User) error {
	if user.Password == "" {
		// No need to run bcrypt if password hasn't changed
		return nil
	}
	pwBytes := []byte(user.Password + uv.pepper)
	hashedBytes, err := bcrypt.GenerateFromPassword(pwBytes, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashedBytes)
	user.Password = ""
	return nil
}

func (uv *userValidator) hmacRememberToken(user *User) error {
	if user.RememberToken == "" {
		return nil
	}
	user.RememberTokenHash = uv.hmac.Hash(user.RememberToken)
	return nil
}

func (uv *userValidator) setRememberTokenIfUnset(user *User) error {
	if user.RememberToken != "" {
		return nil
	}
	token, err := rand.RememberToken()
	if err != nil {
		return err
	}
	user.RememberToken = token
	return nil
}

func (uv *userValidator) idGreaterThan(n uint) userValidatorFunction {
	return func(user *User) error {
		if user.ID <= n {
			return ErrIDInvalid
		}
		return nil
	}
}

func (uv *userValidator) normalizeEmail(user *User) error {
	user.Email = strings.ToLower(user.Email)
	user.Email = strings.TrimSpace(user.Email)
	return nil
}

func (uv *userValidator) requireEmail(user *User) error {
	if user.Email == "" {
		return ErrEmailRequired
	}
	return nil
}

func (uv *userValidator) emailFormat(user *User) error {
	if user.Email == "" {
		return nil
	}
	if !uv.emailRegex.MatchString(user.Email) {
		return ErrEmailNotValid
	}
	return nil
}

func (uv *userValidator) emailIsAvailable(user *User) error {
	existing, err := uv.ByEmail(user.Email)
	if err == ErrNotFound {
		// Email address is available if we don't find
		// a user with that email address.
		return nil
	}
	// We can't continue our validation without a successful
	// query, so if we get any error other than ErrNotFound we
	// should return it.
	if err != nil {
		return err
	}
	// If we get here that means we found a user w/ this email
	// address, so we need to see if this is the same user we
	// are updating, or if we have a conflict.
	if user.ID != existing.ID {
		return ErrEmailTaken
	}
	return nil
}

func (uv *userValidator) passwordMinLength(user *User) error {
	if user.Password == "" {
		return nil
	}
	if len(user.Password) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}

func (uv *userValidator) passwordRequired(user *User) error {
	if user.Password == "" {
		return ErrPasswordRequired
	}
	return nil
}

func (uv *userValidator) passwordHashRequired(user *User) error {
	if user.PasswordHash == "" {
		return ErrPasswordRequired
	}
	return nil
}

func (uv *userValidator) rememberTokenHashRequired(user *User) error {
	if user.RememberTokenHash == "" {
		return ErrRememberTokenHashRequired
	}
	return nil
}

func (uv *userValidator) rememberTokenMinLength(user *User) error {
	if user.RememberToken == "" {
		return nil
	}
	n, err := rand.NumberOfBytes(user.RememberToken)
	if err != nil {
		return err
	}
	if n < 32 {
		return ErrRememberTokenTooShort
	}
	return nil
}

func runUserValidatorFunctions(user *User, validators ...userValidatorFunction) error {
	for _, fn := range validators {
		if err := fn(user); err != nil {
			return err
		}
	}

	return nil
}
