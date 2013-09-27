package util

import (
	"encoding/json"
	"net/http"
	"strconv"

	"appengine"

	"placecreature/model"
)

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

		creature.Import(context)
	}
}

func ParseInt(val string) int {
	if num, err := strconv.ParseInt(val, 10, 0); err == nil {
		return int(num)
	} else {
		return 0
	}
}
