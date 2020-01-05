package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/torresjeff/gallery/controllers"
	"github.com/torresjeff/gallery/middleware"
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
	staticController    *controllers.Static
	usersController     *controllers.Users
	galleriesController *controllers.Galleries
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	prod := flag.Bool("prod", false, "Provide this flag in production. This ensures that a config.json file is provided before the application starts.")
	flag.Parse()
	config := LoadConfig(*prod)
	dbConfig := config.Database
	services, err := models.NewServices(
		models.WithGorm(dbConfig.Dialect(), dbConfig.ConnectionInfo()),
		models.WithLogMode(true),
		models.WithUser(config.Pepper, config.HMACKey),
		models.WithGallery(),
		models.WithImage(),
	)
	// us, err := models.NewUserService(psqlInfo)
	if err != nil {
		panic(err)
	}
	defer services.Close()
	// For dev environment only
	// services.DestructiveReset()
	services.AutoMigrate()

	userMw := middleware.User{
		UserService: services.User,
	}
	requireUserMw := middleware.RequireUser{}

	b := []byte("32-byte-long-auth-key")
	csrfMw := csrf.Protect(b, csrf.Secure(config.IsProd()))

	r := mux.NewRouter()

	staticController = controllers.NewStatic()
	usersController = controllers.NewUsers(services.User)
	galleriesController = controllers.NewGalleries(services.Gallery, services.Image, r)

	// User related routes
	r.HandleFunc("/signup", usersController.RenderSignUp).Methods("GET")
	r.HandleFunc("/signup", usersController.Create).Methods("POST")
	r.HandleFunc("/login", usersController.RenderLogin).Methods("GET")
	r.HandleFunc("/login", usersController.Login).Methods("POST")
	r.HandleFunc("/cookie", usersController.CookieTest).Methods("GET")
	r.HandleFunc("/logout", usersController.Logout).Methods("POST")

	// Gallery related routes
	r.HandleFunc("/galleries", requireUserMw.ApplyFn(galleriesController.RenderIndex)).Methods("GET").Name(controllers.IndexGalleries)
	r.HandleFunc("/galleries/new", requireUserMw.ApplyFn(galleriesController.RenderCreateGallery)).Methods("GET")
	r.HandleFunc("/galleries", requireUserMw.ApplyFn(galleriesController.Create)).Methods("POST")
	r.HandleFunc("/galleries/{id:[0-9]+}", galleriesController.Show).Methods("GET").Name(controllers.ShowGallery)
	r.HandleFunc("/galleries/{id:[0-9]+}/edit", requireUserMw.ApplyFn(galleriesController.RenderEdit)).Methods("GET").Name(controllers.EditGallery)
	r.HandleFunc("/galleries/{id:[0-9]+}/edit", requireUserMw.ApplyFn(galleriesController.Edit)).Methods("POST")
	r.HandleFunc("/galleries/{id:[0-9]+}/delete", requireUserMw.ApplyFn(galleriesController.Delete)).Methods("POST")
	r.HandleFunc("/galleries/{id:[0-9]+}/images", requireUserMw.ApplyFn(galleriesController.ImageUpload)).Methods("POST")
	r.HandleFunc("/galleries/{id:[0-9]+}/images/{filename}/delete", requireUserMw.ApplyFn(galleriesController.ImageDelete)).Methods("POST")

	// Image routes
	imageHandler := http.FileServer(http.Dir("./images/"))
	r.PathPrefix("/images/").Handler(http.StripPrefix("/images/", imageHandler))

	// Asset routes
	assetHandler := http.FileServer(http.Dir("./assets/"))
	assetHandler = http.StripPrefix("/assets/", assetHandler)
	r.PathPrefix("/assets/").Handler(assetHandler)

	// Routes for static content
	r.Handle("/", staticController.HomeView).Methods("GET")
	r.Handle("/contact", staticController.ContactView).Methods("GET")
	r.Handle("/faq", staticController.FaqView).Methods("GET")
	r.NotFoundHandler = staticController.NotFoundView

	// Apply our user middleware before our router even routes a user to the appropriate page,
	// guaranteeing that the user is set in the request context if they are logged in.
	fmt.Println("Starting the server on port", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), csrfMw(userMw.Apply(r))))
}
