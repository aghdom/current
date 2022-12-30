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
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gomarkdown/markdown"
	"github.com/spf13/viper"

	"github.com/aghdom/current/data"
)

type ServerConfig struct {
	Host          string
	Port          int
	AdminUsername string
	AdminPassword string
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
		Host:          viper.GetString("server.host"),
		Port:          viper.GetInt("server.port"),
		AdminUsername: viper.GetString("server.admin_user"),
		AdminPassword: viper.GetString("server.admin_pass"),
	}
	if cfg.AdminPassword == "" || cfg.AdminUsername == "" {
		log.Fatal("Missing admin credentials!")
	}
	return cfg
}

func transformPost(post data.Post) FeedPost {
	return FeedPost{
		Date:    post.Time.Format("2006/01/02"),
		Time:    post.Time.Format("15:04"),
		Unix:    post.Time.Unix(),
		Content: template.HTML(markdown.ToHTML(post.Content, nil, nil)),
	}
}

//go:embed static/*
var staticFS embed.FS

func getAdminCreds(cfg ServerConfig) map[string]string {
	return map[string]string{cfg.AdminUsername: cfg.AdminPassword}
}

func Run() {
	cfg := initConfig()
	r := chi.NewRouter()
	tmpl := template.Must(template.ParseGlob("templates/*"))
	// TODO: This should be refactored to be only called once and not on every startup
	data.InitDB()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {

		pd := PageData{
			Title: "Dom's current",
		}
		for _, p := range data.GetPosts() {
			pd.Feed = append(pd.Feed, transformPost(p))
		}
		tmpl.ExecuteTemplate(w, "index", pd)
	})

	r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "about", nil)
	})

	r.Get("/posts/{timestamp}", func(w http.ResponseWriter, r *http.Request) {
		var tm time.Time
		ts := chi.URLParam(r, "timestamp")
		if ts != "" {
			unix, err := strconv.ParseInt(ts, 10, 64)
			if err != nil {
				//TODO: Handle properly
				log.Fatal(err)
			}
			tm = time.Unix(unix, 0).UTC()
		}
		p, ok := data.GetPostByTime(tm)
		if !ok {
			//TODO: Implement better 404 handling
			tmpl.ExecuteTemplate(w, "index", PageData{Title: "Post not found!"})
			return
		}
		pd := PageData{
			Title: p.Time.Format("2006/01/02 15:04"),
			Feed:  []FeedPost{transformPost(p)},
		}
		tmpl.ExecuteTemplate(w, "index", pd)
	})

	r.Get("/on/{year}/{month}/{day}", func(w http.ResponseWriter, r *http.Request) {
		var year, month, day int64
		var err error
		y, m, d := chi.URLParam(r, "year"), chi.URLParam(r, "month"), chi.URLParam(r, "day")

		if y != "" {
			year, err = strconv.ParseInt(y, 10, 0)
			if err != nil {
				//TODO: Handle properly
				log.Fatal(err)
			}
		}
		if m != "" {
			month, err = strconv.ParseInt(m, 10, 0)
			if err != nil {
				//TODO: Handle properly
				log.Fatal(err)
			}
		}
		if d != "" {
			day, err = strconv.ParseInt(d, 10, 0)
			if err != nil {
				//TODO: Handle properly
				log.Fatal(err)
			}
		}

		tm := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
		pd := PageData{
			Title: "Posts on " + tm.Format("2006/01/02"),
		}
		posts := data.GetPostOnDate(tm)
		for _, p := range posts {
			pd.Feed = append(pd.Feed, transformPost(p))
		}
		tmpl.ExecuteTemplate(w, "index", pd)
	})

	// admin endpoints
	r.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("author", getAdminCreds(cfg)))
		r.Get("/author", func(w http.ResponseWriter, r *http.Request) {
			tmpl.ExecuteTemplate(w, "author", nil)
		})

		r.Post("/author/post", func(w http.ResponseWriter, r *http.Request) {
			data.CreatePost(r.FormValue("content")) // Should empty content be allowed?
			// Redirect back to the admin portal
			w.Header().Add("Location", "/author")
			w.WriteHeader(http.StatusSeeOther)
		})

		r.Post("/author/delete", func(w http.ResponseWriter, r *http.Request) {
			ts, err := strconv.ParseInt(r.FormValue("time"), 10, 64)
			if err != nil {
				//TODO: Handle properly
				log.Fatal(err)
			}
			tm := time.Unix(int64(ts), 0).UTC()
			data.DeletePostByTime(tm)
			// Redirect back to the admin portal
			w.Header().Add("Location", "/author")
			w.WriteHeader(http.StatusSeeOther)
		})
	})

	// serve embedded static files
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
