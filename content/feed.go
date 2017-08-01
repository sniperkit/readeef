package content

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/urandom/readeef/parser"
)

type FeedID int64

type Feed struct {
	ID             FeedID          `json:"id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Link           string          `json:"link"`
	SiteLink       string          `db:"site_link" json:"-"`
	HubLink        string          `db:"hub_link" json:"-"`
	UpdateError    string          `db:"update_error" json:"updateError"`
	SubscribeError string          `db:"subscribe_error" json:"subscribeError"`
	TTL            time.Duration   `json:"-"`
	SkipHours      map[int]bool    `json:"-"`
	SkipDays       map[string]bool `json:"-"`

	parsedArticles []Article
}

func (f Feed) Validate() error {
	if f.data.Link == "" {
		return NewValidationError(errors.New("Feed has no link"))
	}

	if u, err := url.Parse(f.data.Link); err != nil || !u.IsAbs() {
		return NewValidationError(errors.New("Feed has no link"))
	}

	return nil
}

func (f *Feed) Refresh(pf parser.Feed) {
	f.Title = pf.Title
	f.Description = pf.Description
	f.SiteLink = pf.SiteLink
	f.HubLink = pf.HubLink

	f.parsedArticles = make([]Article, len(pf.Articles))

	for i := range pf.Articles {
		a := Article{
			Title:       pf.Articles[i].Title,
			Description: pf.Articles[i].Description,
			Link:        pf.Articles[i].Link,
			Date:        pf.Articles[i].Date,
		}
		a.FeedID = d.ID

		if pf.Articles[i].Guid != "" {
			a.Guid.Valid = true
			a.Guid.String = pf.Articles[i].Guid
		}

		f.parsedArticles[i] = a
	}
}

func (f Feed) ParsedArticles() (a []Article) {
	return f.parsedArticles
}

func (f Feed) String() string {
	return fmt.Sprintf("%d: %s", f.ID, f.Title)
}

/*
type Feed interface {
	Error

	fmt.Stringer

	Data(data ...data.Feed) data.Feed

	Validate() error

	Users() []User

	// Updates the in-memory feed information using the RSS data
	Refresh(pf parser.Feed)
	// Returns the []content.Article created from the RSS data
	ParsedArticles() []Article

	// Returns any new articles since the previous Update
	NewArticles() []Article

	Update()
	Delete()

	SetNewArticlesUnread()

	AllArticles() []Article
	LatestArticles() []Article

	AddArticles([]Article)

	Subscription() Subscription
}

type UserFeed interface {
	Feed
	ArticleSearch
	ArticleRepo
	RepoRelated

	// Detaches from the current user
	Detach()
}

type TaggedFeed interface {
	UserFeed

	Tags(tags ...[]Tag) []Tag

	UpdateTags()
}
*/
