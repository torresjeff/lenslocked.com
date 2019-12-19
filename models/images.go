package models

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ImageService interface {
	Create(galleryID uint, r io.Reader, filename string) error
	ByGalleryID(galleryID uint) ([]Image, error)
	Delete(i *Image) error
}

// Image is used to represent images stored in a Gallery.
// Image is NOT stored in the database, and instead
// references data stored on disk.
type Image struct {
	GalleryID uint
	Filename  string
}

type imageService struct {
}

func NewImageService() ImageService {
	return &imageService{}
}

func (is *imageService) Create(galleryID uint, r io.Reader, filename string) error {
	path, err := is.mkImagePath(galleryID)
	if err != nil {
		return err
	}
	dst, err := os.Create(filepath.Join(path, filename))
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, r)
	if err != nil {
		return err
	}
	return nil
}

func (is *imageService) ByGalleryID(galleryID uint) ([]Image, error) {
	path := is.imagePath(galleryID)
	paths, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		return nil, err
	}
	images := make([]Image, len(paths))
	for i, imgStr := range paths {
		images[i] = Image{
			Filename:  filepath.Base(imgStr), // Base returns the last element of a path. Eg: file name
			GalleryID: galleryID,
		}
	}
	return images, nil
}

func (is *imageService) Delete(image *Image) error {
	return os.Remove(image.RelativePath())
}

func (is *imageService) imagePath(galleryID uint) string {
	return filepath.Join("images", "galleries", fmt.Sprintf("%v", galleryID))
}

func (is *imageService) mkImagePath(galleryID uint) (string, error) {
	galleryPath := is.imagePath(galleryID)
	err := os.MkdirAll(galleryPath, 0755)
	if err != nil {
		return "", err
	}
	return galleryPath, nil
}

// Path is used to build the absolute path used to reference this image
// via a web request.
func (i *Image) Path() string {
	return "/" + i.RelativePath()
}

// RelativePath is used to build the path to this image on our local
// disk, relative to where our Go application is run from.
func (i *Image) RelativePath() string {
	galleryID := fmt.Sprintf("%v", i.GalleryID)
	return filepath.ToSlash(filepath.Join("images", "galleries", galleryID, i.Filename))
}
