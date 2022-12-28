package server

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gomarkdown/markdown"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Host string
	Port int
}

type StatusData struct {
	Feed []template.HTML
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
		tmpl := template.Must(template.ParseFiles("templates/layout.html"))

		md := []byte("## Testing \nStatus **update**")
		md2 := []byte("Another _status_: [link](https://example.com)")
		sd := StatusData{
			Feed: []template.HTML{
				template.HTML(markdown.ToHTML(md, nil, nil)),
				template.HTML(markdown.ToHTML(md2, nil, nil)),
			},
		}
		tmpl.Execute(w, sd)
	})

	fs := http.FileServer(http.Dir("static/"))
	r.Handle("/s/*", http.StripPrefix("/s/", fs))

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	http.ListenAndServe(addr, r)
}
