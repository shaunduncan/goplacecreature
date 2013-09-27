package main

import (
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"appengine"
	"appengine/datastore"
	"appengine/user"

	"placecreature/util"
	"placecreature/model"
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
	ctx := appengine.NewContext(request)
	vars := mux.Vars(request)

	name := vars["creature"]
	width, height := util.ParseInt(vars["width"]), util.ParseInt(vars["height"])

	if creature, err := model.FindCreatureByNameOrAlias(name, ctx); err == nil {
		creature.Serve(response, request, ctx, width, height)
	} else {
		http.NotFound(response, request)
	}
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
	creature := model.NewCreature()

	context := appengine.NewContext(request)
	query := datastore.NewQuery("Creatures").
		Filter("IsPublic =", true).
		Ancestor(creature.Key(context)).
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
	creature := model.NewCreature()

	context := appengine.NewContext(request)
	query := datastore.NewQuery("Creatures").
		Filter("IsPublic =", true).
		Ancestor(creature.Key(context)).
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
