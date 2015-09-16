package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/urandom/readeef"
	"github.com/urandom/readeef/api/processor"
	"github.com/urandom/readeef/content"
	"github.com/urandom/readeef/content/base/search"
	"github.com/urandom/readeef/content/data"
	"github.com/urandom/webfw"
	"github.com/urandom/webfw/context"
	"github.com/urandom/webfw/util"
)

const (
	TTRSS_API_STATUS_OK  = 0
	TTRSS_API_STATUS_ERR = 1
	TTRSS_VERSION        = "1.8.0"
	TTRSS_API_LEVEL      = 12

	TTRSS_FAVORITE_ID = -1
	TTRSS_FRESH_ID    = -3
	TTRSS_ALL_ID      = -4

	TTRSS_FRESH_DURATION = -24 * time.Hour
)

type TtRss struct {
	webfw.BasePatternController
	fm *readeef.FeedManager
	sp content.SearchProvider
	ap []ArticleProcessor
}

type ttRssRequest struct {
	Op            string           `json:"op"`
	Sid           string           `json:"sid"`
	Seq           int              `json:"seq"`
	User          string           `json:"user"`
	Password      string           `json:"password"`
	OutputMode    string           `json:"output_mode"`
	UnreadOnly    bool             `json:"unread_only"`
	Limit         int              `json:"limit"`
	Offset        int              `json:"offset"`
	CatId         int              `json:"cat_id"`
	FeedId        data.FeedId      `json:"feed_id"`
	Skip          int              `json:"skip"`
	IsCat         bool             `json:"is_cat"`
	ShowContent   bool             `json:"show_content"`
	ShowExcerpt   bool             `json:"show_excerpt"`
	ViewMode      string           `json:"view_mode"`
	SinceId       data.ArticleId   `json:"since_id"`
	Sanitize      bool             `json:"sanitize"`
	HasSandbox    bool             `json:"has_sandbox"`
	IncludeHeader bool             `json:"include_header"`
	OrderBy       string           `json:"order_by"`
	Search        string           `json:"search"`
	ArticleIds    []data.ArticleId `json:"article_ids"`
	Mode          int              `json:"mode"`
	Field         int              `json:"field"`
	Data          string           `json:"data"`
	ArticleId     []data.ArticleId `json:"article_id"`
	PrefName      string           `json:"pref_name"`
	FeedUrl       string           `json:"feed_url"`
}

type ttRssResponse struct {
	Seq     int             `json:"seq"`
	Status  int             `json:"status"`
	Content json.RawMessage `json:"content"`
}

type ttRssGenericContent struct {
	Error     string      `json:"error,omitempty"`
	Level     int         `json:"level,omitempty"`
	ApiLevel  int         `json:"api_level,omitempty"`
	Version   string      `json:"version,omitempty"`
	SessionId string      `json:"session_id,omitempty"`
	Status    interface{} `json:"status,omitempty"`
	Unread    int64       `json:"unread,omitempty"`
	Updated   int64       `json:"updated,omitempty"`
	Value     interface{} `json:"value,omitempty"`
}

type ttRssCountersContent []ttRssCounter

type ttRssCounter struct {
	Id         string `json:"id"`
	Counter    int64  `json:"counter"`
	AuxCounter int64  `json:"auxcounter,omitempty"`
}

type ttRssFeedsContent []ttRssFeed

type ttRssFeed struct {
	Id          data.FeedId `json:"id"`
	Title       string      `json:"title"`
	Unread      int64       `json:"unread"`
	CatId       int         `json:"cat_id"`
	FeedUrl     string      `json:"feed_url,omitempty"`
	LastUpdated int64       `json:"last_updated,omitempty"`
	OrderId     int         `json:"order_id,omitempty"`
}

type ttRssHeadlinesHeaderContent []interface{}
type ttRssHeadlinesContent []ttRssHeadline

type ttRssHeadline struct {
	Id        data.ArticleId `json:"id"`
	Unread    bool           `json:"unread"`
	Marked    bool           `json:"marked"`
	Updated   int64          `json:"updated"`
	IsUpdated bool           `json:"is_updated"`
	Title     string         `json:"title"`
	Link      string         `json:"link"`
	FeedId    data.FeedId    `json:"feed_id"`
	Excerpt   string         `json:"excerpt,omitempty"`
	Content   string         `json:"content,omitempty"`
	FeedTitle string         `json:"feed_title"`

	Tags   []string `json:"tags,omitempty"`
	Labels []string `json:"labels,omitempty"`
}

type ttRssHeadlinesHeader struct {
	Id      data.FeedId    `json:"id"`
	FirstId data.ArticleId `json:"first_id"`
	IsCat   bool           `json:"is_cat"`
}

type ttRssConfigContent struct {
	IconsDir        string `json:"icons_dir"`
	IconsUrl        string `json:"icons_url"`
	DaemonIsRunning bool   `json:"daemon_is_running"`
	NumFeeds        int    `json:"num_feeds"`
}

type ttRssSubscribeContent struct {
	Status struct {
		Code int `json:"code"`
	} `json:"status"`
}

type ttRssSession struct {
	login     data.Login
	lastVisit time.Time
}

var (
	ttRssSessions = map[string]ttRssSession{}
)

func NewTtRss(fm *readeef.FeedManager, sp content.SearchProvider, processors []ArticleProcessor) TtRss {
	ap := make([]ArticleProcessor, 0, len(processors))
	for _, p := range processors {
		if _, ok := p.(processor.ProxyHTTP); !ok {
			ap = append(ap, p)
		}
	}

	go func() {
		fiveDaysAgo := time.Now().AddDate(0, 0, -5)
		for id, sess := range ttRssSessions {
			if sess.lastVisit.Before(fiveDaysAgo) {
				delete(ttRssSessions, id)
			}
		}
	}()

	return TtRss{
		webfw.NewBasePatternController(fmt.Sprintf("/v%d/tt-rss/api/", TTRSS_API_LEVEL), webfw.MethodPost, ""),
		fm, sp, ap,
	}
}

func (controller TtRss) Handler(c context.Context) http.Handler {
	repo := readeef.GetRepo(c)
	logger := webfw.GetLogger(c)
	config := readeef.GetConfig(c)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := ttRssRequest{}
		dec := json.NewDecoder(r.Body)

		resp := ttRssResponse{}

		var err error
		var errType string
		var user content.User
		var con interface{}

		switch {
		default:
			in := map[string]interface{}{}
			err = dec.Decode(&in)
			if err != nil {
				err = fmt.Errorf("Error decoding JSON request: %v", err)
				break
			}

			req = ttRssConvertRequest(in)

			logger.Debugf("Request: %#v\n", req)

			resp.Seq = req.Seq

			if req.Op != "login" && req.Op != "isLoggedIn" {
				if sess, ok := ttRssSessions[req.Sid]; ok {
					user = repo.UserByLogin(data.Login(sess.login))
					if repo.Err() != nil {
						errType = "NOT_LOGGED_IN"
					} else {
						sess.lastVisit = time.Now()
						ttRssSessions[req.Sid] = sess
					}
				} else {
					errType = "NOT_LOGGED_IN"
				}
			}

			if errType != "" {
				logger.Debugf("TT-RSS Sessions: %#v\n", ttRssSessions)
				con = ttRssGenericContent{}
				break
			}

			logger.Debugf("TT-RSS OP: %s\n", req.Op)
			switch req.Op {
			case "getApiLevel":
				con = ttRssGenericContent{Level: TTRSS_API_LEVEL}
			case "getVersion":
				con = ttRssGenericContent{Version: TTRSS_VERSION}
			case "login":
				user = repo.UserByLogin(data.Login(req.User))
				if repo.Err() != nil {
					errType = "LOGIN_ERROR"
					break
				}

				if !user.Authenticate(req.Password, []byte(config.Auth.Secret)) {
					errType = "LOGIN_ERROR"
					break
				}

				var sessId string

				login := user.Data().Login

				for id, sess := range ttRssSessions {
					if sess.login == login {
						sessId = id
					}
				}

				if sessId == "" {
					sessId = util.UUID()
					ttRssSessions[sessId] = ttRssSession{login: login, lastVisit: time.Now()}
				}

				con = ttRssGenericContent{
					ApiLevel:  TTRSS_API_LEVEL,
					SessionId: sessId,
				}
			case "logout":
				delete(ttRssSessions, req.Sid)
				con = ttRssGenericContent{Status: "OK"}
			case "isLoggedIn":
				if _, ok := ttRssSessions[req.Sid]; ok {
					con = ttRssGenericContent{Status: true}
				} else {
					con = ttRssGenericContent{Status: false}
				}
			case "getUnread":
				if !req.IsCat {
					var ar content.ArticleRepo
					o := data.ArticleCountOptions{UnreadOnly: true}

					switch req.FeedId {
					case TTRSS_FAVORITE_ID:
						ar = user
						o.FavoriteOnly = true
					case TTRSS_FRESH_ID:
						ar = user
						o.AfterDate = time.Now().Add(TTRSS_FRESH_DURATION)
					case TTRSS_ALL_ID:
						ar = user
					default:
						if req.FeedId > 0 {
							feed := user.FeedById(req.FeedId)
							if feed.HasErr() {
								err = feed.Err()
								break
							}

							ar = feed
						}

					}

					if ar != nil {
						con = ttRssGenericContent{Unread: ar.Count(o)}
					}

				}

				if con == nil {
					con = ttRssGenericContent{Unread: 0}
				}
			case "getCounters":
				if req.OutputMode == "" {
					req.OutputMode = "flc"
				}
				cContent := ttRssCountersContent{}

				o := data.ArticleCountOptions{UnreadOnly: true}
				unreadCount := user.Count(o)
				cContent = append(cContent,
					ttRssCounter{Id: "global-unread", Counter: unreadCount})

				feeds := user.AllFeeds()
				cContent = append(cContent,
					ttRssCounter{Id: "subscribed-feeds", Counter: int64(len(feeds))})

				cContent = append(cContent,
					ttRssCounter{Id: strconv.Itoa(TTRSS_FAVORITE_ID),
						Counter:    user.Count(data.ArticleCountOptions{UnreadOnly: true, FavoriteOnly: true}),
						AuxCounter: user.Count(data.ArticleCountOptions{FavoriteOnly: true})})

				freshTime := time.Now().Add(TTRSS_FRESH_DURATION)
				cContent = append(cContent,
					ttRssCounter{Id: strconv.Itoa(TTRSS_FRESH_ID),
						Counter:    user.Count(data.ArticleCountOptions{UnreadOnly: true, AfterDate: freshTime}),
						AuxCounter: user.Count(data.ArticleCountOptions{AfterDate: freshTime})})

				cContent = append(cContent,
					ttRssCounter{Id: strconv.Itoa(TTRSS_ALL_ID),
						Counter:    user.Count(),
						AuxCounter: unreadCount})

				if strings.Contains(req.OutputMode, "f") {
					for _, f := range feeds {
						cContent = append(cContent,
							ttRssCounter{Id: strconv.FormatInt(int64(f.Data().Id), 10), Counter: f.Count(o)},
						)

					}
				}

				if user.HasErr() {
					err = fmt.Errorf("Error getting user counters: %v\n", user.Err())
				}

				con = cContent
			case "getFeeds":
				fContent := ttRssFeedsContent{}

				if req.CatId == TTRSS_FAVORITE_ID || req.CatId == TTRSS_ALL_ID {
					unreadFav := user.Count(data.ArticleCountOptions{UnreadOnly: true, FavoriteOnly: true})

					if unreadFav > 0 || !req.UnreadOnly {
						fContent = append(fContent, ttRssFeed{
							Id:     TTRSS_FAVORITE_ID,
							Title:  "Starred articles",
							Unread: unreadFav,
							CatId:  TTRSS_FAVORITE_ID,
						})
					}

					freshTime := time.Now().Add(TTRSS_FRESH_DURATION)
					unreadFresh := user.Count(data.ArticleCountOptions{UnreadOnly: true, AfterDate: freshTime})

					if unreadFresh > 0 || !req.UnreadOnly {
						fContent = append(fContent, ttRssFeed{
							Id:     TTRSS_FRESH_ID,
							Title:  "Fresh articles",
							Unread: unreadFresh,
							CatId:  TTRSS_FAVORITE_ID,
						})
					}

					unreadAll := user.Count(data.ArticleCountOptions{UnreadOnly: true})

					if unreadAll > 0 || !req.UnreadOnly {
						fContent = append(fContent, ttRssFeed{
							Id:     TTRSS_ALL_ID,
							Title:  "All articles",
							Unread: unreadAll,
							CatId:  TTRSS_FAVORITE_ID,
						})
					}
				}

				if req.CatId == 0 || req.CatId == TTRSS_FRESH_ID || req.CatId == TTRSS_ALL_ID {
					feeds := user.AllFeeds()
					o := data.ArticleCountOptions{UnreadOnly: true}
					for i := range feeds {
						if req.Limit > 0 {
							if i < req.Offset || i >= req.Limit+req.Offset {
								continue
							}
						}

						d := feeds[i].Data()
						unread := feeds[i].Count(o)

						if unread > 0 || !req.UnreadOnly {
							fContent = append(fContent, ttRssFeed{
								Id:          d.Id,
								Title:       d.Title,
								FeedUrl:     d.Link,
								CatId:       0,
								Unread:      unread,
								LastUpdated: time.Now().Unix(),
								OrderId:     0,
							})
						}
					}
				}

				if user.HasErr() {
					err = fmt.Errorf("Error getting user feeds: %v\n", user.Err())
				}

				con = fContent
			case "getCategories":
				con = ttRssFeedsContent{}
			case "getHeadlines":
				if req.FeedId == 0 {
					errType = "INCORRECT_USAGE"
					break
				}

				limit := req.Limit
				if limit == 0 {
					limit = 200
				}

				var articles []content.UserArticle
				var articleRepo content.ArticleRepo
				var feedTitle string
				firstId := data.ArticleId(0)
				o := data.ArticleQueryOptions{Limit: limit, Offset: req.Skip, UnreadFirst: true}

				if req.FeedId == TTRSS_FAVORITE_ID {
					ttRssSetupSorting(req, user)
					o.FavoriteOnly = true
					articleRepo = user
					feedTitle = "Starred articles"
				} else if req.FeedId == TTRSS_FRESH_ID {
					ttRssSetupSorting(req, user)
					o.AfterDate = time.Now().Add(TTRSS_FRESH_DURATION)
					articleRepo = user
					feedTitle = "Fresh articles"
				} else if req.FeedId == TTRSS_ALL_ID {
					ttRssSetupSorting(req, user)
					articleRepo = user
					feedTitle = "All articles"
				} else if req.FeedId > 0 {
					feed := user.FeedById(req.FeedId)

					ttRssSetupSorting(req, feed)
					articleRepo = feed
					feedTitle = feed.Data().Title
				}

				if req.SinceId > 0 {
					o.AfterId = req.SinceId
				}

				if articleRepo != nil {
					if req.Search != "" {
						if controller.sp != nil {
							if as, ok := articleRepo.(content.ArticleSearch); ok {
								articles = as.Query(req.Search, controller.sp, limit, req.Skip)
							}
						}
					} else {
						var skip bool

						switch req.ViewMode {
						case "all_articles":
						case "adaptive":
						case "unread":
							o.UnreadOnly = true
						case "marked":
							o.FavoriteOnly = true
						default:
							skip = true
						}

						if !skip {
							articles = articleRepo.Articles(o)
						}
					}
				}

				if len(articles) > 0 {
					firstId = articles[0].Data().Id

					for _, p := range controller.ap {
						articles = p.ProcessArticles(articles)
					}
				}

				headlines := ttRssHeadlinesFromArticles(req, articles, feedTitle, nil)
				if req.IncludeHeader {
					header := ttRssHeadlinesHeader{Id: req.FeedId, FirstId: firstId}
					hContent := ttRssHeadlinesHeaderContent{}

					hContent = append(hContent, header)
					hContent = append(hContent, headlines)

					con = hContent
				} else {
					con = headlines
				}
			case "updateArticle":
				articles := user.ArticlesById(req.ArticleIds)
				updateCount := int64(0)

				switch req.Field {
				case 0, 2:
					for _, a := range articles {
						d := a.Data()
						updated := false

						switch req.Field {
						case 0:
							switch req.Mode {
							case 0:
								if d.Favorite {
									updated = true
									d.Favorite = false
								}
							case 1:
								if !d.Favorite {
									updated = true
									d.Favorite = true
								}
							case 2:
								updated = true
								d.Favorite = !d.Favorite
							}
							if updated {
								a.Favorite(d.Favorite)
							}
						case 2:
							switch req.Mode {
							case 0:
								if !d.Read {
									updated = true
									d.Read = true
								}
							case 1:
								if d.Read {
									updated = true
									d.Read = false
								}
							case 2:
								updated = true
								d.Read = !d.Read
							}
							if updated {
								a.Read(d.Read)
							}
						}

						if updated {
							if a.HasErr() {
								err = a.Err()
								break
							}

							updateCount++
						}
					}

					if err != nil {
						break
					}

					con = ttRssGenericContent{Status: "OK", Updated: updateCount}
				}
			case "getArticle":
				articles := user.ArticlesById(req.ArticleId)
				feedTitles := map[data.FeedId]string{}

				for _, a := range articles {
					d := a.Data()
					if _, ok := feedTitles[d.FeedId]; !ok {
						f := repo.FeedById(d.FeedId)
						feedTitles[d.FeedId] = f.Data().Title
					}
				}

				if len(controller.ap) > 0 {
					for _, p := range controller.ap {
						articles = p.ProcessArticles(articles)
					}
				}

				con = ttRssHeadlinesFromArticles(req, articles, "", feedTitles)
			case "getConfig":
				con = ttRssConfigContent{DaemonIsRunning: true, NumFeeds: len(user.AllFeeds())}
			case "updateFeed":
				con = ttRssGenericContent{Status: "OK"}
			case "catchupFeed":
				if !req.IsCat {
					f := user.FeedById(req.FeedId)
					f.ReadState(true, data.ArticleUpdateStateOptions{BeforeDate: time.Now()})

					if f.HasErr() {
						err = f.Err()
						break
					}

					con = ttRssGenericContent{Status: "OK"}
				}
			case "getPref":
				switch req.PrefName {
				case "DEFAULT_UPDATE_INTERVAL":
					con = ttRssGenericContent{Value: int(config.FeedManager.Converted.UpdateInterval.Minutes())}
				case "DEFAULT_ARTICLE_LIMIT":
					con = ttRssGenericContent{Value: 200}
				case "SHOW_CONTENT_PREVIEW":
					con = ttRssGenericContent{Value: true}
				case "HIDE_READ_FEEDS":
					con = ttRssGenericContent{Value: user.Data().ProfileData["unreadOnly"]}
				case "FEEDS_SORT_BY_UNREAD":
					con = ttRssGenericContent{Value: true}
				case "FRESH_ARTICLE_MAX_AGE":
					con = ttRssGenericContent{Value: (-1 * TTRSS_FRESH_DURATION).Hours()}
				}
			case "getLabels":
				con = []interface{}{}
			case "setArticleLabel":
				con = ttRssGenericContent{Status: "OK", Updated: 0}
			case "shareToPublished":
				errType = "Publishing failed"
			case "subscribeToFeed":
				f := repo.FeedByLink(req.FeedUrl)
				for _, u := range f.Users() {
					if u.Data().Login == user.Data().Login {
						con = ttRssSubscribeContent{Status: struct {
							Code int `json:"code"`
						}{0}}
						break
					}
				}

				if f.HasErr() {
					err = f.Err()
					break
				}

				f, err := controller.fm.AddFeedByLink(req.FeedUrl)
				if err != nil {
					errType = "INCORRECT_USAGE"
					break
				}

				uf := user.AddFeed(f)
				if uf.HasErr() {
					err = uf.Err()
					break
				}

				con = ttRssSubscribeContent{Status: struct {
					Code int `json:"code"`
				}{1}}
			case "unsubscribeFeed":
				f := user.FeedById(req.FeedId)
				f.Detach()
				users := f.Users()

				if f.HasErr() {
					err = f.Err()
					if err == content.ErrNoContent {
						errType = "FEED_NOT_FOUND"
					}
					break
				}

				if len(users) == 0 {
					controller.fm.RemoveFeed(f)
				}

				con = ttRssGenericContent{Status: "OK"}
			}
		}

		if err == nil && errType == "" {
			resp.Status = TTRSS_API_STATUS_OK
		} else {
			logger.Infof("Error processing TT-RSS API request: %s %v\n", errType, err)
			resp.Status = TTRSS_API_STATUS_ERR
			if v, ok := con.(ttRssGenericContent); ok {
				v.Error = errType
				con = v
			}
		}

		var b []byte
		b, err = json.Marshal(con)
		if err == nil {
			resp.Content = json.RawMessage(b)
		}

		b, err = json.Marshal(&resp)

		if err == nil {
			w.Header().Set("Content-Type", "text/json")
			w.Write(b)

			logger.Debugf("Output for %s: %s\n", req.Op, string(b))
		} else {
			logger.Print(fmt.Errorf("TT-RSS error %s: %v", req.Op, err))

			w.WriteHeader(http.StatusInternalServerError)
		}

	})
}

func ttRssSetupSorting(req ttRssRequest, sorting content.ArticleSorting) {
	switch req.OrderBy {
	case "date_reverse":
		sorting.SortingByDate()
		sorting.Order(data.AscendingOrder)
	default:
		sorting.SortingByDate()
		sorting.Order(data.DescendingOrder)
	}
}

func ttRssHeadlinesFromArticles(req ttRssRequest, articles []content.UserArticle, feedTitle string, feedTitles map[data.FeedId]string) (c ttRssHeadlinesContent) {
	c = ttRssHeadlinesContent{}
	for _, a := range articles {
		d := a.Data()
		title := feedTitle
		if feedTitles != nil {
			title = feedTitles[d.FeedId]
		}
		h := ttRssHeadline{
			Id:        d.Id,
			Unread:    !d.Read,
			Marked:    d.Favorite,
			Updated:   d.Date.Unix(),
			IsUpdated: !d.Read,
			Title:     d.Title,
			Link:      d.Link,
			FeedId:    d.FeedId,
			FeedTitle: title,
		}

		if req.ShowExcerpt {
			excerpt := search.StripTags(d.Description)
			if len(excerpt) > 100 {
				excerpt = excerpt[:100]
			}

			h.Excerpt = excerpt
		}

		if req.ShowContent {
			h.Content = d.Description
		}

		c = append(c, h)
	}
	return
}

func ttRssConvertRequest(in map[string]interface{}) (req ttRssRequest) {
	for key, v := range in {
		switch key {
		case "op":
			req.Op = ttRssParseString(v)
		case "sid":
			req.Sid = ttRssParseString(v)
		case "seq":
			req.Seq = ttRssParseInt(v)
		case "user":
			req.User = ttRssParseString(v)
		case "password":
			req.Password = ttRssParseString(v)
		case "output_mode":
			req.OutputMode = ttRssParseString(v)
		case "unread_only":
			req.UnreadOnly = ttRssParseBool(v)
		case "limit":
			req.Limit = ttRssParseInt(v)
		case "offset":
			req.Offset = ttRssParseInt(v)
		case "cat_id":
			req.CatId = ttRssParseInt(v)
		case "feed_id":
			req.FeedId = data.FeedId(ttRssParseInt64(v))
		case "skip":
			req.Skip = ttRssParseInt(v)
		case "is_cat":
			req.IsCat = ttRssParseBool(v)
		case "show_content":
			req.ShowContent = ttRssParseBool(v)
		case "show_excerpt":
			req.ShowExcerpt = ttRssParseBool(v)
		case "view_mode":
			req.ViewMode = ttRssParseString(v)
		case "since_id":
			req.SinceId = data.ArticleId(ttRssParseInt64(v))
		case "sanitize":
			req.Sanitize = ttRssParseBool(v)
		case "has_sandbox":
			req.HasSandbox = ttRssParseBool(v)
		case "include_header":
			req.IncludeHeader = ttRssParseBool(v)
		case "order_by":
			req.OrderBy = ttRssParseString(v)
		case "search":
			req.Search = ttRssParseString(v)
		case "article_ids":
			req.ArticleIds = ttRssParseArticleIds(v)
		case "mode":
			req.Mode = ttRssParseInt(v)
		case "field":
			req.Field = ttRssParseInt(v)
		case "data":
			req.Data = ttRssParseString(v)
		case "article_id":
			req.ArticleId = ttRssParseArticleIds(v)
		case "pref_name":
			req.PrefName = ttRssParseString(v)
		case "feed_url":
			req.FeedUrl = ttRssParseString(v)
		}
	}

	return
}

func ttRssParseString(vv interface{}) string {
	if v, ok := vv.(string); ok {
		return v
	}
	return fmt.Sprintf("%v", vv)
}

func ttRssParseBool(vv interface{}) bool {
	switch v := vv.(type) {
	case string:
		return v == "t" || v == "true" || v == "1"
	case float64:
		return v == 1
	case bool:
		return v
	}
	return false
}

func ttRssParseInt(vv interface{}) int {
	switch v := vv.(type) {
	case string:
		i, _ := strconv.Atoi(v)
		return i
	case float64:
		return int(v)
	}
	return 0
}

func ttRssParseInt64(vv interface{}) int64 {
	switch v := vv.(type) {
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		return i
	case float64:
		return int64(v)
	}
	return 0
}

func ttRssParseArticleIds(vv interface{}) (ids []data.ArticleId) {
	switch v := vv.(type) {
	case string:
		parts := strings.Split(v, ",")
		for _, p := range parts {
			if i, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
				ids = append(ids, data.ArticleId(i))
			}
		}
	case []float64:
		for _, p := range v {
			ids = append(ids, data.ArticleId(int64(p)))
		}
	}
	return
}