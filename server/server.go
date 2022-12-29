package server

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gomarkdown/markdown"
	"github.com/spf13/viper"
)

func getPosts() []Post {
	return []Post{
		{
			Time:    time.Date(2022, 12, 28, 8, 30, 0, 0, time.UTC),
			Content: []byte(`First **post**`),
		},
		{
			Time:    time.Date(2022, 12, 28, 8, 31, 0, 0, time.UTC),
			Content: []byte(`Second *post*`),
		},
		{
			Time:    time.Date(2022, 12, 28, 8, 32, 0, 0, time.UTC),
			Content: []byte("Third `post`"),
		},
	}
}

type ServerConfig struct {
	Host string
	Port int
}

type Post struct {
	Time    time.Time
	Content []byte
}

type FeedPost struct {
	Date    string
	Time    string
	Unix    int64
	Content template.HTML
}

type PageData struct {
	Title string
	Feed  []FeedPost
}

func initConfig() ServerConfig {
	cfg := ServerConfig{
		Host: viper.GetString("server.host"),
		Port: viper.GetInt("server.port"),
	}
	return cfg
}

func transformPost(post Post) FeedPost {
	return FeedPost{
		Date:    post.Time.Format("2006/02/01"),
		Time:    post.Time.Format("15:04"),
		Unix:    post.Time.Unix(),
		Content: template.HTML(markdown.ToHTML(post.Content, nil, nil)),
	}
}

//go:embed static/*
var staticFS embed.FS

func Run() {
	cfg := initConfig()
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/page.html"))

		pd := PageData{
			Title: "Dom's current",
		}
		for _, p := range getPosts() {
			pd.Feed = append(pd.Feed, transformPost(p))
		}
		tmpl.ExecuteTemplate(w, "page", pd)
	})

	r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/about.html", "templates/page.html"))
		tmpl.ExecuteTemplate(w, "page", nil)
	})

	// serve embeded static files
	sFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(http.FS(sFS))
	r.Handle("/s/*", http.StripPrefix("/s/", fs))

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	fmt.Println("Serving on http://" + addr + " ...")
	http.ListenAndServe(addr, r)
}
