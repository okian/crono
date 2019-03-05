package main

import (
	`context`
	`crypto/sha1`
	`encoding/csv`
	"fmt"
	"github.com/gocolly/colly"
	"github.com/sirupsen/logrus"
	`log`
	`net/url`
	`os`
	`os/signal`
	`regexp`
	`sort`
	`strings`
	`sync`
	`syscall`
	`time`
)

var (
	urls          = make(chan *url.URL, 100000)
	manager       = make(chan *url.URL, 100000)
	paragraphChan = make(chan *paragraph, 100000)
	peg           = regexp.MustCompile("[\u0600-\u06FF\u0698\u067E\u0686\u06AF]+")
	space         = regexp.MustCompile(`(\s+)`)
	words         = make(map[string]int)
	lock          = sync.Mutex{}

	visited = make(map[string]bool)
	vlock   = sync.Mutex{}
)

func wait() {
	w := make(chan os.Signal, 10)
	signal.Notify(w, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	<-w
	fmt.Println("ops")
}

func extractor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			break
		case p := <-paragraphChan:
			p.text = strings.Replace(p.text, "،", " ", -1)
			p.text = strings.Replace(p.text, "؟", " ", -1)
			p.text = strings.Replace(p.text, "ـ", " ", -1)
			p.text = strings.Replace(p.text, "٪", " ", -1)
			p.text = strings.Replace(p.text, "ً", " ", -1)
			p.text = strings.Replace(p.text, "َ", " ", -1)
			p.text = strings.Replace(p.text, "ُ", " ", -1)
			p.text = strings.Replace(p.text, "ُ،", " ", -1)
			p.text = strings.Replace(p.text, "ِ", " ", -1)
			p.text = strings.Replace(p.text, "ٰ", " ", -1)
			p.text = strings.Replace(p.text, "٠", " ", -1)
			p.text = strings.Replace(p.text, "؛", " ", -1)
			p.text = strings.Replace(p.text, "٬", " ", -1)
			p.text = strings.Replace(p.text, "؛", " ", -1)
			p.text = strings.Replace(p.text, "؛", " ", -1)

			p.text = strings.Replace(p.text, "۰", " ", -1)
			p.text = strings.Replace(p.text, "۱", " ", -1)
			p.text = strings.Replace(p.text, "۲", " ", -1)
			p.text = strings.Replace(p.text, "۳", " ", -1)
			p.text = strings.Replace(p.text, "۴", " ", -1)
			p.text = strings.Replace(p.text, "۵", " ", -1)
			p.text = strings.Replace(p.text, "۶", " ", -1)
			p.text = strings.Replace(p.text, "۷", " ", -1)
			p.text = strings.Replace(p.text, "۸", " ", -1)
			p.text = strings.Replace(p.text, "۹", " ", -1)

			p.text = strings.Replace(p.text, "0", " ", -1)
			p.text = strings.Replace(p.text, "1", " ", -1)
			p.text = strings.Replace(p.text, "2", " ", -1)
			p.text = strings.Replace(p.text, "3", " ", -1)
			p.text = strings.Replace(p.text, "4", " ", -1)
			p.text = strings.Replace(p.text, "5", " ", -1)
			p.text = strings.Replace(p.text, "6", " ", -1)
			p.text = strings.Replace(p.text, "7", " ", -1)
			p.text = strings.Replace(p.text, "8", " ", -1)
			p.text = strings.Replace(p.text, "9", " ", -1)

			p.text = strings.Replace(p.text, "؟", " ", -1)
			p.text = strings.Replace(p.text, "!", " ", -1)
			p.text = strings.Replace(p.text, "@", " ", -1)
			p.text = strings.Replace(p.text, "#", " ", -1)
			p.text = strings.Replace(p.text, "&", " ", -1)

			p.text = string(space.ReplaceAll([]byte(p.text), []byte(" ")))
			lw := make(map[string]int)
			for _, v := range peg.FindAll([]byte(p.text), -1) {

				lw[string(v)] += 1
			}
			lock.Lock()
			for k, v := range lw {
				words[k] += v
			}
			lock.Unlock()
		}
	}
}

func ptmaker(s string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf("https?://%s.*", s))
}

func clc() {
	c := colly.NewCollector(func(x *colly.Collector) {
		x.AllowedDomains = []string{
			"gadgetnews.net",
			"www.irna.ir",
			"www.farsnews.com",
			"dictionary.abadis.ir",
			"www.khabaronline.ir",
			"www.varzesh3.com",
			"www.aparat.com",
			"www.entekhab.ir",
			"www.bbc.com",
			"fa.wikipedia.org",
			"fa.wiktionary.org",
			"www.vajehyab.com",
		}
		x.URLFilters = []*regexp.Regexp{
			ptmaker("gadgetnews.net"),
			ptmaker("www.irna.ir"),
			ptmaker("www.isna.ir"),
			ptmaker("www.ilna.ir/fa"),
			ptmaker("www.digikala.com"),
			ptmaker("www.farsnews.com"),
			ptmaker("dictionary.abadis.ir"),
			ptmaker("www.khabaronline.ir"),
			ptmaker("www.varzesh3.com"),
			ptmaker("www.aparat.com"),
			ptmaker("www.entekhab.ir"),
			ptmaker("www.bbc.com/persian/"),
			ptmaker("fa.wikipedia.org/wiki/"),
			ptmaker("fa.wiktionary.org/wiki/"),
			ptmaker("www.vajehyab.com/"),
		}
		x.AllowURLRevisit = false

	})
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		l, err := url.Parse(e.Request.AbsoluteURL(e.Attr("href")))
		if err == nil {
			vlock.Lock()
			if _, ok := visited[l.String()]; !ok {
				visited[l.String()] = true
			}
			vlock.Unlock()
			_ = c.Visit(l.String())
		}
	})
	c.OnHTML("p", func(e *colly.HTMLElement) {
		paragraphChan <- &paragraph{
			url:  e.Request.URL,
			text: e.Text,
		}
	})

	c.OnHTML("h1", func(e *colly.HTMLElement) {
		paragraphChan <- &paragraph{
			url:  e.Request.URL,
			text: e.Text,
		}
	})

	c.OnHTML("h2", func(e *colly.HTMLElement) {
		paragraphChan <- &paragraph{
			url:  e.Request.URL,
			text: e.Text,
		}
	})

	c.OnHTML("h3", func(e *colly.HTMLElement) {
		paragraphChan <- &paragraph{
			url:  e.Request.URL,
			text: e.Text,
		}
	})

	c.OnHTML("h4", func(e *colly.HTMLElement) {
		paragraphChan <- &paragraph{
			url:  e.Request.URL,
			text: e.Text,
		}
	})

	c.OnHTML("h5", func(e *colly.HTMLElement) {
		paragraphChan <- &paragraph{
			url:  e.Request.URL,
			text: e.Text,
		}
	})

	c.OnHTML("h6", func(e *colly.HTMLElement) {
		paragraphChan <- &paragraph{
			url:  e.Request.URL,
			text: e.Text,
		}
	})

	_ = c.Visit("http://gadgetnews.net")
	_ = c.Visit("http://www.irna.ir/")
	_ = c.Visit("https://www.farsnews.com/")
	_ = c.Visit("https://dictionary.abadis.ir/")
	_ = c.Visit("https://www.khabaronline.ir/")
	_ = c.Visit("https://www.varzesh3.com/")
	_ = c.Visit("https://www.aparat.com/")
	_ = c.Visit("http://www.entekhab.ir/")
	_ = c.Visit("http://www.bbc.com/persian")
	_ = c.Visit("https://fa.wikipedia.org/wiki/")
	_ = c.Visit("https://fa.wiktionary.org/wiki/")
	_ = c.Visit("https://www.vajehyab.com/?q=%D8%A2%D8%A8")
	_ = c.Visit("www.isna.ir")
	_ = c.Visit("www.ilna.ir/fa")
	_ = c.Visit("www.digikala.com")
	_ = c.Visit("www.farsnews.com")
}

func main() {
	ctx, cl := context.WithCancel(context.Background())
	for i := 0; i < 1000; i++ {
		go extractor(ctx)
	}

	wait()
	cl()
	fmt.Println("sss")

	lock.Lock()

	ws := make([]word, 0)

	for k, v := range words {
		ws = append(ws, word{
			w: k,
			c: v,
		})
	}

	sort.SliceStable(ws, func(i, j int) bool {
		return ws[i].c > ws[i].c
	})
	fmt.Println("len: ", len(words))
	fmt.Println("len: ", len(visited))
	lock.Unlock()

	f, err := os.Create("words.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	wr := csv.NewWriter(f)
	defer wr.Flush()

	for _, v := range ws {
		err := wr.Write([]string{fmt.Sprint(v.c), v.w})
		if err != nil {
			log.Fatal(err)
		}
	}
	time.Sleep(time.Second)
}

type word struct {
	w string
	c int
}

type paragraph struct {
	url  *url.URL
	text string
}

type job struct {
	url.URL
}

func (j *paragraph) Hash() string {
	sh := sha1.New()
	return fmt.Sprintf("%x", sh.Sum([]byte(j.url.Host)))
}

func checkWiki(u *url.URL) bool {
	if strings.Contains(u.String(), ".wik") {
		if u.Host == "fa.wikiperdia.org" || u.Host == "fa.wiktionary.org" {
			return true
		}
		return false
	}
	return true
}

func worker(ctx context.Context, urls chan *url.URL) {
	for {
		select {
		case <-ctx.Done():
			break
		case u := <-urls:
			time.Sleep(time.Millisecond * 250)
			if !checkWiki(u) {
				continue
			}
			c := colly.NewCollector()
			c.OnHTML("a[href]", func(e *colly.HTMLElement) {
				l, err := url.Parse(e.Request.AbsoluteURL(e.Attr("href")))
				if err == nil {
					vlock.Lock()
					if _, ok := visited[l.String()]; !ok {
						visited[l.String()] = true
						urls <- l
					}
					vlock.Unlock()
				}
			})
			c.OnHTML("p", func(e *colly.HTMLElement) {
				paragraphChan <- &paragraph{
					url:  e.Request.URL,
					text: e.Text,
				}
			})
			fmt.Println("VISITING: ", u.String())
			err := c.Visit(u.String())
			if err != nil {
				logrus.Error("VISIT: ", u.String(), err)
			}
		}
	}
}
