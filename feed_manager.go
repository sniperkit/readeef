package readeef

import (
	"bytes"
	"crypto/md5"
	"errors"
	"io"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"net/http"
	"net/url"

	"github.com/urandom/readeef/config"
	"github.com/urandom/readeef/content"
	"github.com/urandom/readeef/content/data"
	"github.com/urandom/readeef/parser"
	"github.com/urandom/webfw/util"
)

// TODO: split up this struct and modify the api

type FeedManager struct {
	config           config.Config
	repo             content.Repo
	addFeed          chan content.Feed
	removeFeed       chan content.Feed
	done             chan bool
	client           *http.Client
	log              Logger
	activeFeeds      map[data.FeedId]bool
	lastUpdateHash   map[data.FeedId][md5.Size]byte
	hubbub           *Hubbub
	parserProcessors []parser.Processor
	feedMonitors     []content.FeedMonitor
}

var (
	commentPattern = regexp.MustCompile("<!--.*?-->")
	linkPattern    = regexp.MustCompile(`<link ([^>]+)>`)

	ErrNoAbsolute = errors.New("Feed link is not absolute")
	ErrNoFeed     = errors.New("Feed not found")

	httpStatusPrefix = "HTTP Status: "
)

func NewFeedManager(repo content.Repo, c config.Config, l Logger) *FeedManager {
	return &FeedManager{
		repo: repo, config: c, log: l,
		addFeed:        make(chan content.Feed, 2),
		removeFeed:     make(chan content.Feed, 2),
		done:           make(chan bool),
		activeFeeds:    map[data.FeedId]bool{},
		lastUpdateHash: map[data.FeedId][md5.Size]byte{},
		client:         NewTimeoutClient(c.Timeout.Converted.Connect, c.Timeout.Converted.ReadWrite),
	}
}

// TODO: stop using these kinds of apis
func (fm *FeedManager) Hubbub(hubbub ...*Hubbub) *Hubbub {
	if len(hubbub) > 0 {
		fm.hubbub = hubbub[0]
	}

	return fm.hubbub
}

func (fm *FeedManager) Client(c ...*http.Client) *http.Client {
	if len(c) > 0 {
		fm.client = c[0]
	}

	return fm.client
}

func (fm *FeedManager) ParserProcessors(p ...[]parser.Processor) []parser.Processor {
	if len(p) > 0 {
		fm.parserProcessors = p[0]
	}

	return fm.parserProcessors
}

func (fm *FeedManager) AddFeedMonitor(m content.FeedMonitor) {
	fm.feedMonitors = append(fm.feedMonitors, m)
}

func (fm FeedManager) Start() {
	fm.log.Infoln("Starting the feed manager")

	go fm.reactToChanges()

	go fm.scheduleFeeds()
}

func (fm *FeedManager) Stop() {
	fm.log.Infoln("Stopping the feed manager")

	fm.done <- true
}

func (fm *FeedManager) AddFeed(f content.Feed) {
	if f.Data().HubLink != "" && fm.hubbub != nil {
		err := fm.hubbub.Subscribe(f)

		if err == nil || err == ErrSubscribed {
			return
		}
	}

	fm.addFeed <- f
}

func (fm *FeedManager) RemoveFeed(f content.Feed) {
	if f.Data().HubLink != "" && fm.hubbub != nil {
		fm.hubbub.Unsubscribe(f)
	}
	fm.removeFeed <- f
}

func (fm *FeedManager) AddFeedByLink(link string) (content.Feed, error) {
	u, err := url.Parse(link)
	if err == nil {
		if !u.IsAbs() {
			return nil, ErrNoAbsolute
		}
		u.Fragment = ""
		link = u.String()
	} else {
		return nil, err
	}

	f := fm.repo.FeedByLink(link)
	err = f.Err()
	if err != nil && err != content.ErrNoContent {
		return f, err
	}

	if err != nil {
		fm.log.Infoln("Discovering feeds in " + link)

		feeds, err := fm.discoverSecureParserFeeds(u)

		if err != nil {
			return nil, err
		}

		f = feeds[0]

		f.Update()
		if f.HasErr() {
			return f, f.Err()
		}

		// Do not halt the adding process due to slow monitors
		go fm.processFeedUpdateMonitors(f)
	}

	fm.log.Infoln("Adding feed " + f.String() + " to manager")
	fm.AddFeed(f)

	return f, nil
}

func (fm *FeedManager) RemoveFeedByLink(link string) (content.Feed, error) {
	f := fm.repo.FeedByLink(link)
	if f.HasErr() {
		err := f.Err()
		if err == content.ErrNoContent {
			err = nil
		}
		return f, f.Err()
	}

	if f.Validate() != nil {
		return f, nil
	}

	fm.log.Infoln("Removing feed " + f.String() + " from manager")

	fm.removeFeed <- f

	return f, nil
}

func (fm *FeedManager) DiscoverFeeds(link string) ([]content.Feed, error) {
	feeds := []content.Feed{}

	u, err := url.Parse(link)
	if err == nil {
		if !u.IsAbs() {
			return feeds, ErrNoAbsolute
		}
		link = u.String()
	} else {
		return feeds, err
	}

	f := fm.repo.FeedByLink(link)
	err = f.Err()
	if err != nil && err != content.ErrNoContent {
		return feeds, f.Err()
	} else {
		if err != nil {
			fm.log.Debugln("Discovering feeds in " + link)

			discovered, err := fm.discoverSecureParserFeeds(u)

			if err != nil {
				return feeds, err
			}

			fm.log.Debugf("Discovered %d feeds in %s\n", len(discovered), link)
			feeds = append(feeds, discovered...)
		}
	}

	return feeds, nil
}

func (fm FeedManager) AddFeedChannel() chan<- content.Feed {
	return fm.addFeed
}

func (fm FeedManager) RemoveFeedChannel() chan<- content.Feed {
	return fm.removeFeed
}

func (fm *FeedManager) reactToChanges() {
	for {
		select {
		case f := <-fm.addFeed:
			fm.startUpdatingFeed(f)
		case f := <-fm.removeFeed:
			fm.stopUpdatingFeed(f)
		case <-fm.done:
			return
		}
	}
}

func (fm *FeedManager) startUpdatingFeed(f content.Feed) {
	if f == nil {
		fm.log.Infoln("No feed provided")
		return
	}

	data := f.Data()

	if data.Id == 0 || fm.activeFeeds[data.Id] {
		fm.log.Infoln("Feed " + data.Link + " already active")
		return
	}

	d := 30 * time.Minute
	if fm.config.FeedManager.Converted.UpdateInterval != 0 {
		if data.TTL != 0 && data.TTL > fm.config.FeedManager.Converted.UpdateInterval {
			d = data.TTL
		} else {
			d = fm.config.FeedManager.Converted.UpdateInterval
		}
	}

	fm.activeFeeds[data.Id] = true

	go func() {
		fm.requestFeedContent(f)

		ticker := time.After(d)

		fm.log.Infof("Starting feed scheduler for %s and duration %d\n", f, d)
	TICKER:
		for {
			select {
			case now := <-ticker:
				if !fm.activeFeeds[data.Id] {
					fm.log.Infof("Feed '%s' no longer active\n", data.Link)
					break TICKER
				}

				if !data.SkipHours[now.Hour()] && !data.SkipDays[now.Weekday().String()] {
					fm.requestFeedContent(f)
				}

				ticker = time.After(d)
				fm.log.Infof("New feed ticker for '%s' after %d\n", data.Link, d)
			case <-fm.done:
				fm.stopUpdatingFeed(f)
				return
			}
		}
	}()
}

func (fm *FeedManager) stopUpdatingFeed(f content.Feed) {
	if f == nil {
		fm.log.Infoln("No feed provided")
		return
	}

	data := f.Data()

	fm.log.Infoln("Stopping feed update for " + data.Link)
	delete(fm.activeFeeds, data.Id)

	users := f.Users()
	if f.HasErr() {
		fm.log.Printf("Error getting users for feed '%s': %v\n", f, f.Err())
	} else {
		if len(users) == 0 {
			fm.log.Infoln("Removing orphan feed " + f.String() + " from the database")

			for _, m := range fm.feedMonitors {
				if err := m.FeedDeleted(f); err != nil {
					fm.log.Printf(
						"Error invoking monitor '%s' on deleted feed '%s': %v\n",
						reflect.TypeOf(m), f, err)
				}
			}
			f.Delete()
			if f.HasErr() {
				fm.log.Printf("Error deleting feed '%s' from the repository: %v\n", f, f.Err())
			}
		}
	}
}

func (fm *FeedManager) requestFeedContent(f content.Feed) {
	if f == nil {
		fm.log.Infoln("No feed provided")
		return
	}

	data := f.Data()

	fm.log.Infoln("Requesting feed content for " + f.String())

	resp, err := fm.client.Get(data.Link)

	if err != nil {
		data.UpdateError = err.Error()
	} else if resp.StatusCode != http.StatusOK {
		defer func() {
			// Drain the body so that the connection can be reused
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}()
		data.UpdateError = httpStatusPrefix + strconv.Itoa(resp.StatusCode)
	} else {
		defer resp.Body.Close()
		data.UpdateError = ""

		buf := util.BufferPool.GetBuffer()
		defer util.BufferPool.Put(buf)

		if _, err := buf.ReadFrom(resp.Body); err == nil {
			hash := md5.Sum(buf.Bytes())
			if b, ok := fm.lastUpdateHash[data.Id]; ok && bytes.Equal(b[:], hash[:]) {
				fm.log.Infof("Content of feed %s is the same as the previous update\n", f)
				return
			}
			fm.lastUpdateHash[data.Id] = hash

			if pf, err := parser.ParseFeed(buf.Bytes(), parser.ParseRss2, parser.ParseAtom, parser.ParseRss1); err == nil {
				f.Refresh(fm.processParserFeed(pf))
			} else {
				data.UpdateError = err.Error()
			}
		} else {
			data.UpdateError = err.Error()
		}

	}

	if data.UpdateError != "" {
		fm.log.Printf("Error updating feed '%s': %s\n", f, data.UpdateError)
	}

	f.Data(data)

	select {
	case <-fm.done:
		return
	default:
		fm.updateFeed(f)
	}
}

func (fm *FeedManager) scheduleFeeds() {
	feeds := fm.repo.AllUnsubscribedFeeds()
	if fm.repo.HasErr() {
		fm.log.Printf("Error fetching unsubscribed feeds: %v\n", fm.repo.Err())
		return
	}

	for _, f := range feeds {
		fm.log.Infoln("Scheduling feed " + f.String())

		fm.AddFeed(f)
	}
}

func (fm FeedManager) discoverSecureParserFeeds(u *url.URL) (feeds []content.Feed, err error) {
	if u.Scheme == "http" {
		fm.log.Debugln("Testing secure link of", u)

		u.Scheme = "https"
		feeds, err = fm.discoverParserFeeds(u.String())
		u.Scheme = "http"
	}

	if u.Scheme != "http" || err != nil {
		feeds, err = fm.discoverParserFeeds(u.String())
	}

	return
}

func (fm FeedManager) discoverParserFeeds(link string) ([]content.Feed, error) {
	fm.log.Debugf("Fetching feed link body %s\n", link)
	resp, err := http.Get(link)
	if err != nil {
		return []content.Feed{}, err
	}
	defer resp.Body.Close()

	buf := util.BufferPool.GetBuffer()
	defer util.BufferPool.Put(buf)

	buf.ReadFrom(resp.Body)

	if parserFeed, err := parser.ParseFeed(buf.Bytes(), parser.ParseRss2, parser.ParseAtom, parser.ParseRss1); err == nil {
		fm.log.Debugf("Discovering link %s contains feed data\n", link)

		feed := fm.repo.Feed()

		feed.Data(data.Feed{Link: link})
		feed.Refresh(fm.processParserFeed(parserFeed))

		return []content.Feed{feed}, nil
	} else {
		fm.log.Debugf("Searching for html links within the discovering link %s\n", link)

		html := commentPattern.ReplaceAllString(buf.String(), "")
		links := linkPattern.FindAllStringSubmatch(html, -1)

		feeds := []content.Feed{}
		for _, l := range links {
			attrs := l[1]
			if strings.Contains(attrs, `"application/rss+xml"`) || strings.Contains(attrs, `'application/rss+xml'`) {
				index := strings.Index(attrs, "href=")
				attr := attrs[index+6:]
				index = strings.IndexByte(attr, attrs[index+5])
				href := attr[:index]

				if u, err := url.Parse(href); err != nil {
					return []content.Feed{}, ErrNoFeed
				} else {
					if !u.IsAbs() {
						l, _ := url.Parse(link)

						u.Scheme = l.Scheme

						if u.Host == "" {
							u.Host = l.Host
						}

						href = u.String()
					}

					fs, err := fm.discoverParserFeeds(href)
					if err != nil {
						return []content.Feed{}, err
					}

					feeds = append(feeds, fs[0])
				}
			}
		}

		if len(feeds) != 0 {
			return feeds, nil
		}
	}

	return []content.Feed{}, ErrNoFeed
}

func (fm FeedManager) updateFeed(f content.Feed) {
	f.Update()

	if f.HasErr() {
		fm.log.Printf("Error updating feed '%s' database record: %v\n", f, f.Err())
	} else {
		fm.processFeedUpdateMonitors(f)
	}
}

func (fm FeedManager) processFeedUpdateMonitors(f content.Feed) {
	if len(f.NewArticles()) > 0 {
		for _, m := range fm.feedMonitors {
			if err := m.FeedUpdated(f); err != nil {
				fm.log.Printf("Error invoking monitor '%s' on updated feed '%s': %v\n",
					reflect.TypeOf(m), f, err)
			}
		}
	} else {
		fm.log.Infoln("No new articles for " + f.String())
	}
}

func (fm FeedManager) processParserFeed(pf parser.Feed) parser.Feed {
	for _, p := range fm.parserProcessors {
		pf = p.Process(pf)
	}

	return pf
}
