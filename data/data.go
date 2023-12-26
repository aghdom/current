package data

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gorilla/feeds"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

type Post struct {
	Time    time.Time
	Content []byte
	BskyURI []byte
}

// runDBMigrationTX runs SQL database migration queries in a single transaction.
// It starts a new transaction in 'db' and executes the query strings in 'queries' one by one.
// Lastly, it updates "user_version" to the next one after the current 'dbVersion'.
func runDBMigrationTx(db *sql.DB, dbVersion int, queries []string) {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Defer transaction rollback in case anything goes wrong
	defer tx.Rollback()

	//Execute migration queries
	for _, query := range queries {
		_, qErr := tx.ExecContext(ctx, query)
		if qErr != nil {
			log.Fatal(qErr)
		}
	}
	// Increment DB version
	newVersion := dbVersion + 1
	_, err = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", newVersion))
	if err != nil {
		log.Fatal(err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}
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

	// Check current DB version
	rows, err := db.Query("PRAGMA user_version")
	if err != nil {
		log.Fatal(err)
	}
	rows.Next()
	var dbVersion int
	if err := rows.Scan(&dbVersion); err != nil {
		log.Fatal(err)
	}
	rows.Close()

	// If necessary, execute all missed migrations for the current DB
	// When adding a new migration, remember to add 'fallthrough' to the previous one
	switch dbVersion {
	case 0:
		runDBMigrationTx(db, 0, []string{"CREATE TABLE IF NOT EXISTS posts(ts INTEGER PRIMARY KEY, content TEXT)"})
		fallthrough
	case 1:
		// Introduced a 'bsky_uri' field to store BlueSky uri for deletion purposes in case of federation
		runDBMigrationTx(db, 1, []string{"ALTER TABLE posts ADD bsky_uri TEXT"})
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

func queryPostsWithURI(query string, args ...any) []Post {
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
		if err := rows.Scan(&ts, &post.Content, &post.BskyURI); err != nil {
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
	posts := queryPostsWithURI("SELECT ts,content,bsky_uri FROM posts WHERE ts == ? ORDER BY ts DESC", tm.Unix())
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

	_, err = db.Exec("INSERT INTO posts(ts, content, bsky_uri) VALUES (?, ?, ?);", post.Time.Unix(), post.Content, post.BskyURI)
	if err != nil {
		log.Fatal(err)
	}
}

func CreatePost(content string, bskyFed bool) {
	var bskyUri string
	t := time.Now().UTC()
	if bskyFed {
		uri, err := BskyCreatePost(content, t)
		if err != nil {
			log.Fatal(err)
		}
		bskyUri = uri
	}
	insertPost(Post{
		Time:    t.Truncate(time.Second),
		Content: []byte(content),
		BskyURI: []byte(bskyUri),
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

func DeletePostByTime(tm time.Time, bskyDel bool) {
	if bskyDel {
		post, ok := GetPostByTime(tm)
		if !ok {
			return
		}
		if err := BskyDeletePost(string(post.BskyURI)); err != nil {
			log.Fatal(err)
		}
	}
	deletePost(tm)
}

func getFeed() *feeds.Feed {
	feed := &feeds.Feed{
		Title:       "aghdom's current",
		Link:        &feeds.Link{Href: "https://current.aghdom.eu/"},
		Description: "My personal micro-blog",
		Author:      &feeds.Author{Name: "Dominik Ágh", Email: "agh.dominik@gmail.com"},
	}

	posts := queryPosts("SELECT ts,content FROM posts ORDER BY ts DESC")
	feed.Created = posts[0].Time

	for _, post := range posts {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:   post.Time.Format("2006/01/02 15:04"),
			Link:    &feeds.Link{Href: fmt.Sprintf("https://current.aghdom.eu/posts/%d", post.Time.Unix())},
			Author:  &feeds.Author{Name: "Dominik Ágh", Email: "agh.dominik@gmail.com"},
			Content: string(markdown.ToHTML(markdown.NormalizeNewlines(post.Content), nil, nil)),
			Created: post.Time,
		})
	}

	return feed
}

func GetAtomFeed() []byte {
	feed := getFeed()
	atom, err := feed.ToAtom()
	if err != nil {
		log.Fatal(err)
	}
	return []byte(atom)
}

func GetRssFeed() []byte {
	feed := getFeed()
	rss, err := feed.ToRss()
	if err != nil {
		log.Fatal(err)
	}
	return []byte(rss)
}

func GetJsonFeed() []byte {
	feed := getFeed()
	json, err := feed.ToJSON()
	if err != nil {
		log.Fatal(err)
	}
	return []byte(json)
}
