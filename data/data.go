package data

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

type Post struct {
	Time    time.Time
	Content []byte
}

func InitDB() {
	fp := viper.GetString("sqlite.filepath")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		_, createErr := os.Create(fp)
		if createErr != nil {
			log.Fatal(err)
		}
		fmt.Println("Created DB file: " + fp)
	}
	db, err := sql.Open("sqlite3", fp)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS posts(ts INTEGER PRIMARY KEY, content TEXT)")
	if err != nil {
		log.Fatal(err)
	}
}

func CountPosts(query string) int {
	fp := viper.GetString("sqlite.filepath")
	db, err := sql.Open("sqlite3", fp)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT COUNT(ts) AS count FROM posts WHERE content LIKE ?", "%"+query+"%")
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	rows.Next()
	var count int
	if err := rows.Scan(&count); err != nil {
		log.Fatal(err)
	}
	return count
}

func queryPosts(query string, args ...any) []Post {
	fp := viper.GetString("sqlite.filepath")
	var result []Post
	db, err := sql.Open("sqlite3", fp)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		var post Post
		var ts int64
		if err := rows.Scan(&ts, &post.Content); err != nil {
			log.Fatal(err)
		}
		post.Time = time.Unix(ts, 0).Truncate(time.Second).UTC()
		result = append(result, post)
	}

	return result
}

func GetPosts(page, count int, query string) []Post {
	if query != "" {
		return queryPosts("SELECT ts,content FROM posts WHERE content LIKE ? ORDER BY ts DESC LIMIT ?,?", "%"+query+"%", count*(page-1), count)
	}
	return queryPosts("SELECT ts,content FROM posts ORDER BY ts DESC LIMIT ?,?", count*(page-1), count)
}

func GetPostByTime(tm time.Time) (Post, bool) {
	posts := queryPosts("SELECT ts,content FROM posts WHERE ts == ? ORDER BY ts DESC", tm.Unix())
	if len(posts) > 0 {
		return posts[0], true
	}
	return Post{}, false
}

func GetPostOnDate(dt time.Time) []Post {
	return queryPosts("SELECT ts,content FROM posts WHERE ? <= ts AND ts < ? ORDER BY ts DESC", dt.Unix(), dt.Add(24*time.Hour).Unix())
}

func insertPost(post Post) {
	fp := viper.GetString("sqlite.filepath")
	db, err := sql.Open("sqlite3", fp)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO posts(ts, content) VALUES (?, ?);", post.Time.Unix(), post.Content)
	if err != nil {
		log.Fatal(err)
	}
}

func CreatePost(content string) {
	insertPost(Post{
		Time:    time.Now().Truncate(time.Second).UTC(),
		Content: []byte(content),
	})
}

func deletePost(tm time.Time) {
	fp := viper.GetString("sqlite.filepath")
	db, err := sql.Open("sqlite3", fp)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM posts WHERE ts == ?", tm.Unix())
	if err != nil {
		log.Fatal(err)
	}
}

func DeletePostByTime(tm time.Time) {
	deletePost(tm)
}
