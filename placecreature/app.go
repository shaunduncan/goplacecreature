package main

import (
	"html/template"
	goimg "image"
	"image/jpeg"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"appengine"
	"appengine/datastore"
	"appengine/image"
	"appengine/urlfetch"
	"appengine/user"

	"placecreature/util"
	"placecreature/model"
	"placecreature/graphics"
)


type Context map[string] interface {}

var templates = make(map[string] *template.Template)

func init() {
	loadTemplates()

	router := mux.NewRouter()
	router.HandleFunc("/", index)
	router.HandleFunc("/attribution", attribution)
	router.HandleFunc("/creatures", creatures)
	router.HandleFunc("/admin", admin)
	router.HandleFunc("/logout", logout)

	// Creature wildcard
	router.HandleFunc("/{creature}/{width:[0-9]+}/{height:[0-9]+}", creature)

	http.Handle("/", router)
}

func loadTemplates() {
	os.Chdir("placecreature/templates")

	// Really?!?
	f := template.FuncMap{
		"eq":    func(a int, b int) bool { return a == b },
		"lower": func(a string) string { return strings.ToLower(a) },
		"mod":   func(a int, b int) int { return a % b },
	}

	// Load in a loop
	tpls := []string{"index", "admin", "attribution", "creatures"}
	for _, name := range tpls {
		tpl := template.New(name).Funcs(f)
		templates[name] = template.Must(tpl.ParseFiles(name + ".html", "base.html"))
	}

	os.Chdir("../..")
}

func creature(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)

	dim := make([]int, 2)
	name := vars["creature"]

	for idx, key := range []string{"width", "height"} {
		val, err := strconv.ParseInt(vars[key], 10, 0)
		if err != nil {
			http.NotFound(response, request)
		}
		dim[idx] = int(val)
	}

	width, height := dim[0], dim[1]
	max := int(math.Max(float64(width), float64(height)))

	// Check if creature exists
	var buf []model.Creature
	var creature model.Creature

	ctx := appengine.NewContext(request)
	datastore.NewQuery("Creatures").Ancestor(util.CreatureKey(ctx)).GetAll(ctx, &buf)

	// Iterate all creatures, checking if name or any alias matches
	for _, obj := range buf {
		if obj.Name == name {
			creature = obj
			break
		} else {
			aliases := sort.StringSlice(obj.Aliases)
			aliases.Sort()
			idx := aliases.Search(name)

			if idx < len(obj.Aliases) && obj.Aliases[idx] == name {
				creature = obj
				break
			}
		}
	}

	// Use AppEngine image service to start out
	url, _ := image.ServingURL(ctx, creature.BlobKey, &image.ServingURLOptions{Size: max, Crop: true})

	// Grab the image
	client := urlfetch.Client(ctx)
	resp, _ := client.Get(url.String())
	defer resp.Body.Close()

	// Just in case
	if resp.StatusCode != 200 {
		http.NotFound(response, request)
	}

	// Do we need further cropping?
	if width == height {
		data, _ := ioutil.ReadAll(resp.Body)
		response.Write(data)
		response.Header().Set("Content-Type", "image/jpeg")
		return
	}

	src, err := jpeg.Decode(resp.Body)
	dest := goimg.NewRGBA(goimg.Rect(0, 0, width, height))
	if err != nil {
		panic(err)
	}

	graphics.Thumbnail(dest, src)

	response.Header().Set("Content-Type", "image/jpeg")
	jpeg.Encode(response, dest, nil)
}

func admin(response http.ResponseWriter, request *http.Request) {
	context := appengine.NewContext(request)
	u := user.Current(context)
	alert := ""

	if u == nil {
		url, _ := user.LoginURL(context, request.URL.String())
		response.Header().Set("Location", url)
		response.WriteHeader(http.StatusFound)
		return
	}

	if request.Method == "POST" {
		contents, _ := util.ReadUploadedFile(request, "fixture")
		util.ImportFixture(contents, context)
		alert = "Uploaded Successfully"
	}

	templates["admin"].ExecuteTemplate(response, "base", Context{
		"Title": "Admin",
		"Alert": alert,
	})
}

func logout(response http.ResponseWriter, request *http.Request) {
	context := appengine.NewContext(request)
	u := user.Current(context)

	if u != nil {
		url, _ := user.LogoutURL(context, request.URL.String())
		response.Header().Set("Location", url)
		response.WriteHeader(http.StatusFound)
	} else {
		url := "/"
		response.Header().Set("Location", url)
		response.WriteHeader(http.StatusFound)
	}
}

func index(response http.ResponseWriter, request *http.Request) {
	templates["index"].ExecuteTemplate(response, "base", Context{
		"Title": "placecreature",
	})
}

func creatures(response http.ResponseWriter, request *http.Request) {
	var creatures []model.Creature

	context := appengine.NewContext(request)
	query := datastore.NewQuery("Creatures").
		Filter("IsPublic =", true).
		Ancestor(util.CreatureKey(context)).
		Order("Name")

	if _, err := query.GetAll(context, &creatures); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	templates["creatures"].ExecuteTemplate(response, "base", Context{
		"Title": "Creatures List",
		"Creatures": creatures,
	})
}

func attribution(response http.ResponseWriter, request *http.Request) {
	var creatures []model.Creature

	context := appengine.NewContext(request)
	query := datastore.NewQuery("Creatures").
		Filter("IsPublic =", true).
		Ancestor(util.CreatureKey(context)).
		Order("Name")

	if _, err := query.GetAll(context, &creatures); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	templates["attribution"].ExecuteTemplate(response, "base", Context{
		"Title": "Image Attribution",
		"Creatures": creatures,
	})
}
