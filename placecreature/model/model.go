package model

import "appengine"

type Creature struct {
	Name        string
	Aliases     []string
	IsPublic    bool
	Source      string
	License     string
	AuthorName  string
	AuthorURL   string
	OriginalURL string
	BlobKey     appengine.BlobKey
}
