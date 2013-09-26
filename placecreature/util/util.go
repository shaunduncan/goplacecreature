package util

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/urlfetch"

	"placecreature/model"
)

func CreatureKey(context appengine.Context) *datastore.Key {
	return datastore.NewKey(context, "Creatures", "creatures", 0, nil)
}

func ReadUploadedFile(request *http.Request, field string) ([]byte, error) {
	var contents []byte
	file, _, err := request.FormFile(field)

	if err != nil {
		return contents, err
	}

	defer file.Close()
	for {
		buf := make([]byte, 1024)
		n, err := file.Read(buf)
		if err != nil {
			return contents, err
		}
		contents = append(contents, buf[:n]...)
	}

	return contents, nil
}

func ImportFixture(contents []byte, context appengine.Context) {
	type proto struct {
		Name      string
		Is_public bool
		Aliases   []string
		Images    []map[string]string
	}
	var data []proto
	json.Unmarshal(contents, &data)

	for _, parsed := range data {
		creature := model.Creature{
			Name:     parsed.Name,
			IsPublic: parsed.Is_public,
			Aliases:  parsed.Aliases,
		}

		for _, img := range parsed.Images {
			// Just one image for now...
			creature.Source = img["source"]
			creature.License = img["license"]
			creature.AuthorName = img["author_name"]
			creature.AuthorURL = img["author_url"]
			creature.OriginalURL = img["original_url"]
			break
		}

		NewCreature(&creature, context)
	}
}

func NewCreature(creature *model.Creature, context appengine.Context) {
	var buf []model.Creature

	datastore.NewQuery("Creatures").
		Filter("Name =", creature.Name).
		Ancestor(CreatureKey(context)).
		Order("Name").
		Limit(1).
		GetAll(context, &buf)

	if len(buf) > 0 {
		return
	}

	// Fetch the source contents
	client := urlfetch.Client(context)
	resp, err := client.Get(creature.Source)
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
	creature.BlobKey, _ = writer.Key()

	key := datastore.NewIncompleteKey(context, "Creatures", CreatureKey(context))
	datastore.Put(context, key, creature)
}
