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

func main() {
	ctx, cl := context.WithCancel(context.Background())
	for i := 0; i < 20; i++ {
		go worker(ctx, urls)
	}
	for i := 0; i < 100; i++ {
		go extractor(ctx)
	}

	webs := []string{
		"http://gadgetnews.net",
		"http://www.irna.ir/",
		"https://www.farsnews.com/",
		"https://dictionary.abadis.ir/",
		"https://www.khabaronline.ir/",
		"https://www.varzesh3.com/",
		"https://www.aparat.com/",
	}
	for _, v := range webs {
		u, err := url.Parse(v)
		if err != nil {
			log.Fatal(err)
		}
		urls <- u
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

func worker(ctx context.Context, urls chan *url.URL) {
	for {
		select {
		case <-ctx.Done():
			break
		case u := <-urls:
			time.Sleep(time.Millisecond * 250)
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
