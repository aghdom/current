package data

import (
	"time"
)

type Post struct {
	Time    time.Time
	Content []byte
}

var posts = []Post{
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

func GetPosts() *[]Post {
	return &posts
}

func GetPostByTime(tm time.Time) (Post, bool) {
	for _, p := range *GetPosts() {
		if p.Time == tm {
			return p, true
		}
	}
	return Post{}, false
}

func inTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}

func GetPostOnDate(dt time.Time) []Post {
	var result []Post
	for _, p := range *GetPosts() {
		if inTimeSpan(dt, dt.Add(24*time.Hour), p.Time) {
			result = append(result, p)
		}
	}
	return result
}

func CreatePost(content string) {
	posts := GetPosts()
	*posts = append(*posts, Post{
		Time:    time.Now().Truncate(time.Second).UTC(),
		Content: []byte(content),
	})
}

func removePost(index int) {
	posts = append(posts[:index], posts[index+1:]...)
}

func DeletePostByTime(tm time.Time) {
	var to_delete int
	for i, p := range *GetPosts() {
		if p.Time == tm {
			to_delete = i
		}
	}
	removePost(to_delete)
}
