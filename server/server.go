package server

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gomarkdown/markdown"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Host string
	Port int
}

type Post struct {
	Time    time.Time
	Content []byte
}

type FeedData struct {
	Title string
	Feed  []template.HTML
}

func initConfig() ServerConfig {
	cfg := ServerConfig{
		Host: viper.GetString("server.host"),
		Port: viper.GetInt("server.port"),
	}
	return cfg
}

func Run() {
	cfg := initConfig()
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/page.html"))

		posts := []Post{
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
		fd := FeedData{
			Title: "Dom's current",
		}
		for _, p := range posts {
			fd.Feed = append(fd.Feed, template.HTML(markdown.ToHTML(p.Content, nil, nil)))
		}
		tmpl.ExecuteTemplate(w, "page", fd)
	})

	r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/about.html", "templates/page.html"))
		tmpl.ExecuteTemplate(w, "page", nil)
	})

	// server static files
	fs := http.FileServer(http.Dir("static/"))
	r.Handle("/s/*", http.StripPrefix("/s/", fs))

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	http.ListenAndServe(addr, r)
}
