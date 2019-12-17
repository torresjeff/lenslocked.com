package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/torresjeff/gallery/models"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "lenslocked_dev"
)

type User struct {
	gorm.Model
	Name   string
	Email  string `gorm:"not null;unique_index"`
	Orders []Order
}

type Order struct {
	gorm.Model
	UserID      uint
	Amount      int
	Description string
}

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	// db, err := gorm.Open("postgres", psqlInfo)
	// if err != nil {
	// 	panic(err)
	// }
	// defer db.Close()
	// // If you want to enable detailed logging
	// db.LogMode(true)

	// // Creates/updates the model in the database
	// db.AutoMigrate(&User{}, &Order{})

	/*Name, Email := getInfo()
	u := &User{
		Name:  Name,
		Email: Email,
	}

	// Create a record
	if err = db.Create(u).Error; err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", u)*/

	// Query a single record
	// var user User
	// db.First(&user)

	// if db.Error != nil {
	// 	panic(db.Error)
	// }
	// fmt.Println(user)

	// var user2 User
	// id := 3
	// db.First(&user2, id)
	// if db.Error != nil {
	// 	panic(db.Error)
	// }
	// fmt.Println(user2)

	// var user3 User
	// maxID := 3

	// db.Where("id <= ?", maxID).First(&user3)
	// if db.Error != nil {
	// 	panic(db.Error)
	// }
	// fmt.Println(user3)

	// Querying with object values
	// var user4 User
	// user4.Email = "sid@hotmail.com"

	// db.Where(user4).First(&user4)
	// if db.Error != nil {
	// 	panic(db.Error)
	// }
	// fmt.Println(user4)

	// Querying multiple records
	/*var users []User
	db.Find(&users)
	if db.Error != nil {
		panic(db.Error)
	}
	fmt.Println("Retrieved", len(users), "users.")
	fmt.Println(users)*/

	// Associate orders with a user
	// var user5 User
	// db.First(&user5)

	// if db.Error != nil {
	// 	panic(db.Error)
	// }
	// fmt.Println(user5)

	// db.Preload("Orders").First(&user5)
	// if db.Error != nil {
	// 	panic(db.Error)
	// }
	// fmt.Println("Email:", user5.Email)
	// fmt.Println("Number of orders:", len(user5.Orders))
	// fmt.Println("Orders:", user5.Orders)
	// createOrder(db, user5, 1001, "Fake description #1")
	// createOrder(db, user5, 9999, "Fake description #2")
	// createOrder(db, user5, 8800, "Fake description #3")

	us, err := models.NewUserService(psqlInfo)
	if err != nil {
		panic(err)
	}
	defer us.Close()
	// For dev environment only
	us.DestructiveReset()
	us.Create(&models.User{
		Name:  "Michael Scott",
		Email: "michael@dundermifflin.com",
	})
	user, err := us.ByEmail("michael@dundermifflin.com")
	if err != nil {
		panic(err)
	}
	fmt.Println(user)

	user.Name = "Jeff"
	err = us.Update(user)
	if err != nil {
		panic(err)
	} else {
		fmt.Println("User updated correctly")
	}

	fmt.Println("Attempting to delete user with ID:", user.ID)
	err = us.Delete(user.ID)
	if err != nil {
		panic(err)
	} else {
		fmt.Println("User deleted")
	}

	fmt.Println("Attempting to get user that was just deleted")
	_, err = us.ById(user.ID)
	if err != models.ErrNotFound {
		fmt.Println("User NOT deleted!")
	} else {
		fmt.Println("User was deleted.")
	}
}

func getInfo() (name, email string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Name: ")
	name, _ = reader.ReadString('\n')
	name = strings.TrimSpace(name)
	fmt.Print("Email: ")
	email, _ = reader.ReadString('\n')
	email = strings.TrimSpace(email)
	return name, email
}

func createOrder(db *gorm.DB, user User, amount int, desc string) {
	db.Create(&Order{
		UserID:      user.ID,
		Amount:      amount,
		Description: desc,
	})
	if db.Error != nil {
		panic(db.Error)
	}
}
