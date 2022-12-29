package server

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
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
		Date:    post.Time.Format("2006/01/02"),
		Time:    post.Time.Format("15:04"),
		Unix:    post.Time.Unix(),
		Content: template.HTML(markdown.ToHTML(post.Content, nil, nil)),
	}
}

//go:embed static/*
var staticFS embed.FS

func inTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}

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

	r.Get("/posts/{timestamp}", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/page.html"))
		var tm time.Time
		ts := chi.URLParam(r, "timestamp")
		if ts != "" {
			unix, err := strconv.ParseInt(ts, 10, 64)
			if err != nil {
				log.Fatal(err)
			}
			tm = time.Unix(unix, 0).UTC()
		}

		pd := PageData{
			Title: "Dom's current",
		}

		for _, p := range getPosts() {
			if p.Time == tm {
				pd.Feed = append(pd.Feed, transformPost(p))
				pd.Title = p.Time.Format("2006/02/01 15:04")
			}
		}
		tmpl.ExecuteTemplate(w, "page", pd)
	})

	r.Get("/on/{year}/{month}/{day}", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/page.html"))
		var year, month, day int64
		var err error
		y, m, d := chi.URLParam(r, "year"), chi.URLParam(r, "month"), chi.URLParam(r, "day")

		if y != "" {
			year, err = strconv.ParseInt(y, 10, 0)
			if err != nil {
				log.Fatal(err)
			}
		}
		if m != "" {
			month, err = strconv.ParseInt(m, 10, 0)
			if err != nil {
				log.Fatal(err)
			}
		}
		if d != "" {
			day, err = strconv.ParseInt(d, 10, 0)
			if err != nil {
				log.Fatal(err)
			}
		}

		tm := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
		pd := PageData{
			Title: "Posts on " + tm.Format("2006/01/02"),
		}

		for _, p := range getPosts() {
			if inTimeSpan(tm, tm.Add(24*time.Hour), p.Time) {
				pd.Feed = append(pd.Feed, transformPost(p))
			}
		}
		tmpl.ExecuteTemplate(w, "page", pd)
	})

	// serve embeded static files
	sFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(http.FS(sFS))
	r.Handle("/s/*", http.StripPrefix("/s/", fs))

	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Println("Listening on " + addr)
	http.ListenAndServe(addr, r)
}
