package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/torresjeff/gallery/context"
	"github.com/torresjeff/gallery/models"
	"github.com/torresjeff/gallery/views"
)

const (
	IndexGalleries = "index_galleries"
	ShowGallery    = "show_gallery"
	EditGallery    = "edit_gallery"

	maxMultipartMemory = 1 << 20 // 1 MB
)

type Galleries struct {
	CreateGalleryView *views.View
	ShowView          *views.View
	EditView          *views.View
	IndexView         *views.View
	gs                models.GalleryService
	is                models.ImageService
	r                 *mux.Router
}

type NewGalleryForm struct {
	Title string `schema:"title"`
}

func NewGalleries(gs models.GalleryService, is models.ImageService, r *mux.Router) *Galleries {
	return &Galleries{
		CreateGalleryView: views.NewView("bootstrap", "galleries/new"),
		ShowView:          views.NewView("bootstrap", "galleries/show"),
		EditView:          views.NewView("bootstrap", "galleries/edit"),
		IndexView:         views.NewView("bootstrap", "galleries/index"),
		gs:                gs,
		is:                is,
		r:                 r,
	}
}

func (g *Galleries) RenderCreateGallery(w http.ResponseWriter, r *http.Request) {
	g.CreateGalleryView.Render(w, r, nil)
}

func (g *Galleries) RenderIndex(w http.ResponseWriter, r *http.Request) {
	user := context.User(r.Context())
	galleries, err := g.gs.ByUserId(user.ID)
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	var vd views.Data
	vd.Yield = galleries
	g.IndexView.Render(w, r, vd)
}

func (g *Galleries) Create(w http.ResponseWriter, r *http.Request) {
	var vd views.Data
	var form NewGalleryForm
	if err := parseForm(r, &form); err != nil {
		vd.SetAlert(err)
		g.CreateGalleryView.Render(w, r, vd)
		return
	}

	user := context.User(r.Context())
	gallery := models.Gallery{
		Title:  form.Title,
		UserID: user.ID,
	}
	if err := g.gs.Create(&gallery); err != nil {
		vd.SetAlert(err)
		g.CreateGalleryView.Render(w, r, vd)
		return
	}

	url, err := g.r.Get(EditGallery).URL("id", strconv.Itoa(int(gallery.ID)))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Redirect(w, r, url.Path, http.StatusFound)

}

func (g *Galleries) Show(w http.ResponseWriter, r *http.Request) {

	gallery, err := g.galleryByID(w, r)
	if err != nil {
		// The galleryByID would have already rendered the error, so simply return
		return
	}

	var vd views.Data
	vd.Yield = gallery
	g.ShowView.Render(w, r, vd)
}

func (g *Galleries) RenderEdit(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r)
	if err != nil {
		// The galleryByID would have already rendered the error, so simply return
		return
	}

	user := context.User(r.Context())
	if gallery.UserID != user.ID {
		http.Error(w, "You do not have permission to edit this gallery.", http.StatusForbidden)
		return
	}
	var vd views.Data
	vd.Yield = gallery
	g.EditView.Render(w, r, vd)
}

func (g *Galleries) Edit(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r)
	if err != nil {
		// The galleryByID would have already rendered the error, so simply return
		return
	}

	user := context.User(r.Context())
	if gallery.UserID != user.ID {
		http.Error(w, "Gallery not found", http.StatusNotFound)
		return
	}

	var vd views.Data
	vd.Yield = gallery

	var form NewGalleryForm
	if err := parseForm(r, &form); err != nil {
		vd.SetAlert(err)
		g.EditView.Render(w, r, vd)
		return
	}
	gallery.Title = form.Title
	err = g.gs.Update(gallery)
	if err != nil {
		vd.SetAlert(err)
	} else {
		vd.Alert = &views.Alert{
			Level:   views.AlertLvlSuccess,
			Message: "Gallery updated successfully.",
		}
	}
	g.EditView.Render(w, r, vd)
}

func (g *Galleries) Delete(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r)
	if err != nil {
		// The galleryByID would have already rendered the error, so simply return
		return
	}

	user := context.User(r.Context())
	if gallery.UserID != user.ID {
		http.Error(w, "You do not have permission to delete this gallery", http.StatusForbidden)
		return
	}

	var vd views.Data
	err = g.gs.Delete(gallery.ID)
	if err != nil {
		vd.SetAlert(err)
		vd.Yield = gallery
		g.EditView.Render(w, r, vd)
		return
	}

	url, err := g.r.Get(IndexGalleries).URL()
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Redirect(w, r, url.Path, http.StatusFound)
}

func (g *Galleries) ImageUpload(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r)
	if err != nil {
		return
	}
	user := context.User(r.Context())
	if gallery.UserID != user.ID {
		http.Error(w, "Gallery not found", http.StatusNotFound)
		return
	}

	var vd views.Data
	vd.Yield = gallery
	err = r.ParseMultipartForm(maxMultipartMemory)
	if err != nil {
		// Couldn't parse form, set alert
		vd.SetAlert(err)
		g.EditView.Render(w, r, vd)
		return
	}

	// Iterate over uploaded files to process them
	files := r.MultipartForm.File["images"]
	for _, f := range files {
		// Open the uploaded file
		file, err := f.Open()
		if err != nil {
			vd.SetAlert(err)
			g.EditView.Render(w, r, vd)
			return
		}
		defer file.Close() // Always make sure to close the file to avoid memory leaks

		err = g.is.Create(gallery.ID, file, f.Filename)
		if err != nil {
			vd.SetAlert(err)
			g.EditView.Render(w, r, vd)
			return
		}

	}
	vd.Alert = &views.Alert{
		Level:   views.AlertLvlSuccess,
		Message: "Images successfully loaded.",
	}
	g.EditView.Render(w, r, vd)
}

func (g *Galleries) ImageDelete(w http.ResponseWriter, r *http.Request) {
	gallery, err := g.galleryByID(w, r)
	if err != nil {
		return
	}
	user := context.User(r.Context())
	if gallery.UserID != user.ID {
		http.Error(w, "You do not have permission to edit this gallery or image.", http.StatusForbidden)
		return
	}

	// Get the file name from the path
	filename := mux.Vars(r)["filename"]
	image := models.Image{
		Filename:  filename,
		GalleryID: gallery.ID,
	}

	// Try to delete the image
	err = g.is.Delete(&image)
	if err != nil {
		// Render edit page with any errors
		var vd views.Data
		vd.Yield = gallery
		vd.SetAlert(err)
		g.EditView.Render(w, r, vd)
		return
	}

	// If all goes well redirect to the edit page
	url, err := g.r.Get(EditGallery).URL("id", fmt.Sprintf("%v", gallery.ID))
	if err != nil {
		http.Redirect(w, r, "/galleries", http.StatusFound)
		return
	}
	http.Redirect(w, r, url.Path, http.StatusFound)
}

func (g *Galleries) galleryByID(w http.ResponseWriter, r *http.Request) (*models.Gallery, error) {
	// Gets all path parameters
	vars := mux.Vars(r)
	// Get the path parameter with name "id"
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid gallery ID", http.StatusNotFound)
		return nil, err
	}
	gallery, err := g.gs.ById(uint(id))
	if err != nil {
		switch err {
		case models.ErrNotFound:
			http.Error(w, "Gallery not found", http.StatusNotFound)
		default:
			http.Error(w, "Whoops! Something went wrong.", http.StatusInternalServerError)
		}
		return nil, err
	}

	images, _ := g.is.ByGalleryID(gallery.ID)
	gallery.Images = images
	return gallery, nil
}
