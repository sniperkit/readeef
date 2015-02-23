package base

import "github.com/urandom/readeef/content"

type ArticleSearch struct {
	HighlightStyle string
}

func (s ArticleSearch) Highlight(highlight string) content.ArticleSearch {
	s.HighlightStyle = highlight

	return s
}

func (s ArticleSearch) Query(query string) (ua []content.UserArticle) {
	// TODO: move search_index.go here and in sql/article_search.go
	return
}
