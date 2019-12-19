package models

import "github.com/jinzhu/gorm"

const (
	ErrUserIDRequired modelError = "models: user ID is required"
	ErrTitleRequired  modelError = "models: title is required"
)

type Gallery struct {
	gorm.Model
	UserID uint    `gorm:"not null;index"`
	Title  string  `gorm:"not null"`
	Images []Image `gorm:"-"`
}

type GalleryDB interface {
	Create(*Gallery) error
	ById(uint) (*Gallery, error)
	ByUserId(uint) ([]Gallery, error)
	Update(*Gallery) error
	Delete(uint) error
}

type GalleryService interface {
	GalleryDB
}

type galleryGorm struct {
	db *gorm.DB
}

type galleryService struct {
	GalleryDB
}

type galleryValidator struct {
	GalleryDB
}

type galleryValidatorFunction func(*Gallery) error

var _ GalleryDB = &galleryGorm{}

func NewGalleryService(db *gorm.DB) GalleryService {
	return &galleryService{
		GalleryDB: &galleryValidator{
			GalleryDB: &galleryGorm{
				db: db,
			},
		},
	}
}

func (g *Gallery) ImagesSplitN(n int) [][]Image {
	// Create our 2D slice
	ret := make([][]Image, n)
	for i := 0; i < n; i++ {
		ret[i] = make([]Image, 0)
	}

	for i, img := range g.Images {
		ret[i%n] = append(ret[i%n], img)
	}

	return ret
}

func (gg *galleryGorm) Create(gallery *Gallery) error {
	return gg.db.Create(gallery).Error
}

func (gg *galleryGorm) ById(id uint) (*Gallery, error) {
	var gallery Gallery
	err := first(gg.db.Where("id = ?", id), &gallery)
	if err != nil {
		return nil, err
	}

	return &gallery, nil
}

func (gg *galleryGorm) ByUserId(userId uint) ([]Gallery, error) {
	var galleries []Gallery
	db := gg.db.Where("user_id = ?", userId)
	if err := db.Find(&galleries).Error; err != nil {
		return nil, err
	}

	return galleries, nil
}

func (gg *galleryGorm) Update(gallery *Gallery) error {
	return gg.db.Save(gallery).Error
}

func (gg *galleryGorm) Delete(id uint) error {
	gallery := Gallery{Model: gorm.Model{ID: id}}
	return gg.db.Delete(&gallery).Error
}

func runGalleryValidatorFunctions(gallery *Gallery, validators ...galleryValidatorFunction) error {
	for _, fn := range validators {
		if err := fn(gallery); err != nil {
			return err
		}
	}
	return nil
}

func (gv *galleryValidator) Create(gallery *Gallery) error {
	err := runGalleryValidatorFunctions(gallery,
		gv.userIDRequired,
		gv.titleRequired)
	if err != nil {
		return err
	}
	return gv.GalleryDB.Create(gallery)
}

func (gv *galleryValidator) Update(gallery *Gallery) error {
	err := runGalleryValidatorFunctions(gallery,
		gv.userIDRequired,
		gv.titleRequired)
	if err != nil {
		return err
	}
	return gv.GalleryDB.Update(gallery)
}

func (gv *galleryValidator) Delete(id uint) error {
	var gallery Gallery
	gallery.ID = id
	if err := runGalleryValidatorFunctions(&gallery, gv.nonZeroID); err != nil {
		return err
	}
	return gv.GalleryDB.Delete(gallery.ID)
}

func (gv *galleryValidator) userIDRequired(g *Gallery) error {
	if g.UserID <= 0 {
		return ErrUserIDRequired
	}
	return nil
}

func (gv *galleryValidator) titleRequired(g *Gallery) error {
	if g.Title == "" {
		return ErrTitleRequired
	}
	return nil
}

func (gv *galleryValidator) nonZeroID(gallery *Gallery) error {
	if gallery.ID <= 0 {
		return ErrIDInvalid
	}
	return nil
}
