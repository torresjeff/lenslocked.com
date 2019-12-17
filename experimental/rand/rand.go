package main

import (
	"fmt"

	"github.com/torresjeff/gallery/controllers"
	"github.com/torresjeff/gallery/models"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "lenslocked_dev"
)

var (
	usersController  *controllers.Users
	staticController *controllers.Static
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	us, err := models.NewUserService(psqlInfo)
	if err != nil {
		panic(err)
	}
	defer us.Close()
	// For dev environment only
	us.DestructiveReset()
	us.AutoMigrate()

	user := models.User{
		Name:  "Michael Scott",
		Email: "michael@dundermifflin.com",
	}

	err = us.Create(&user)
	if err != nil {
		panic(err)
	}
	// Verify that the user has a RememberToken and RememberTokenHash
	fmt.Printf("%v+\n", user)
	if user.RememberToken == "" {
		panic("Invalid remember token")
	}

	// Verify that we can lookup a user with that RememberToken
	user2, err := us.ByRememberToken(user.RememberToken)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v+\n", user2)

}
