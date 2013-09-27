package model

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"strconv"

	"code.google.com/p/graphics-go/graphics"

	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	aeimg "appengine/image"
	"appengine/memcache"
	"appengine/urlfetch"
)

const CREATURE_REL = "Creatures"

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

func NewCreature() *Creature {
	return new(Creature)
}

func (c Creature) Key(context appengine.Context) *datastore.Key {
	return datastore.NewKey(context, CREATURE_REL, "creatures", 0, nil)
}

func (c Creature) Serve(response http.ResponseWriter, request *http.Request, ctx appengine.Context, width int, height int) {
	max := int(math.Max(float64(width), float64(height)))
	key := c.Name + ":" + strconv.Itoa(width) + "x" + strconv.Itoa(height)

	if width == 0 || height == 0 {
		http.NotFound(response, request)
	}

	// In cache?
	if cached, err := memcache.Get(ctx, key); err == nil {
		response.Header().Set("Content-Type", "image/jpeg")
		response.Write(cached.Value)
		return
	}

	if image, err := c.CropImage(width, height, max, ctx); err == nil {
		response.Header().Set("Content-Type", "image/jpeg")

		// Get the raw data and cache it
		writer := bytes.NewBuffer(make([]byte, 0))
		jpeg.Encode(writer, image, nil)

		item := memcache.Item {
			Key:   key,
			Value: writer.Bytes(),
		}

		memcache.Set(ctx, &item)
		response.Write(item.Value)
		return
	}

	http.NotFound(response, request)
}

func (c Creature) CropImage(width, height, max int, ctx appengine.Context) (image.Image, error) {
	url, _ := aeimg.ServingURL(ctx, c.BlobKey, &aeimg.ServingURLOptions{Size: max, Crop: true})

	client := urlfetch.Client(ctx)
	resp, _ := client.Get(url.String())

	// Just in case
	if resp.StatusCode != 200 {
		return nil, errors.New("Blob not found")
	}

	// Do we need further cropping?
	if width == height {
		return jpeg.Decode(resp.Body)
	}

	src, err := jpeg.Decode(resp.Body)
	dest := image.NewRGBA(image.Rect(0, 0, width, height))
	if err != nil {
		return nil, err
	}

	graphics.Thumbnail(dest, src)

	return dest, nil
}

func FindCreatureByNameOrAlias(lookup string, ctx appengine.Context) (Creature, error) {
	var buf []Creature
	creature := NewCreature()

	datastore.NewQuery(CREATURE_REL).Ancestor(creature.Key(ctx)).GetAll(ctx, &buf)

	// Iterate all creatures, checking if name or any alias matches
	for _, obj := range buf {
		if obj.Name == lookup {
			return obj, nil
		} else {
			aliases := sort.StringSlice(obj.Aliases)
			aliases.Sort()
			idx := aliases.Search(lookup)

			if idx < len(obj.Aliases) && obj.Aliases[idx] == lookup {
				return obj, nil
			}
		}
	}

	return Creature{}, errors.New("Not Found")
}

func (c Creature) Exists(context appengine.Context) bool {
	var buf []Creature

	datastore.NewQuery("Creatures").
		Filter("Name =", c.Name).
		Ancestor(c.Key(context)).
		Order("Name").
		Limit(1).
		GetAll(context, &buf)

	return len(buf) > 0
}

func (c Creature) Save(context appengine.Context) {
	key := datastore.NewIncompleteKey(context, "Creatures", c.Key(context))
	datastore.Put(context, key, c)
}

func (c Creature) Import(context appengine.Context) {
	if c.Exists(context) {
		return
	}

	// Fetch the source contents
	client := urlfetch.Client(context)
	resp, err := client.Get(c.Source)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Store the Blob
	writer, _ := blobstore.Create(context, "image/jpeg")
	writer.Write(data)
	writer.Close()
	c.BlobKey, _ = writer.Key()

	c.Save(context)
}
