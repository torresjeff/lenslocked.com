package controllers

import (
	"github.com/torresjeff/gallery/views"
)

type Static struct {
	HomeView    *views.View
	FaqView     *views.View
	ContactView *views.View

	NotFoundView *views.View
}

func NewStatic() *Static {
	return &Static{
		HomeView:     views.NewView("bootstrap", "home"),
		FaqView:      views.NewView("bootstrap", "faq"),
		ContactView:  views.NewView("bootstrap", "contact"),
		NotFoundView: views.NewView("notfound", "404"),
	}
}
