package main

import (
	`context`
	`crypto/sha1`
	"fmt"
	"github.com/gocolly/colly"
	"github.com/sirupsen/logrus"
	`log`
	`net/url`
	`os`
	`os/signal`
	`regexp`
	`sync`
	`syscall`
	`time`
)

var (
	urls          = make(chan *url.URL,1000)
	manager       = make(chan *url.URL,1000)
	paragraphChan = make(chan *paragraph,1000)
	peg = regexp.MustCompile("[\u0600-\u06FF\u0698\u067E\u0686\u06AF]+")
	words = make(map[string]int)
	lock = sync.Mutex{}

	visited = make(map[string]bool)
	vlock = sync.Mutex{}
)

func wait() {
	w := make(chan os.Signal, 10)
	signal.Notify(w, syscall.SIGKILL, syscall.SIGTERM,syscall.SIGQUIT, syscall.SIGINT)
	<-w
	fmt.Println("ops")
}

func extractor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			break
		case p := <-paragraphChan:
			lw :=make(map[string]int)
			for _, v := range peg.FindAll([]byte(p.text), -1) {
				lw[string(v)]+=1
			}
			lock.Lock()
			for k, v := range lw {
				words[k]+=v
			}
			lock.Unlock()
		}
	}
}
func main() {
	ctx,cl :=context.WithCancel( context.Background())
	for i := 0; i < 30; i++ {
		go worker(ctx, urls)
	}
	for i:=0;i<100;i++ {
		go extractor(ctx)
	}
	// u, err := url.Parse("http://www.clickyab.com")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// urls <- u
	//
	u1, err := url.Parse("http://gadgetnews.net")
	if err != nil {
		log.Fatal(err)
	}
	urls <- u1

	u2, err := url.Parse("http://www.irna.ir/")
	if err != nil {
		log.Fatal(err)
	}
	urls <- u2
	fmt.Println("sss")
	wait()
	cl()
	fmt.Println("sss")

	lock.Lock()
	for k, v := range words {
		fmt.Println(fmt.Sprintf("%10s-%d",k,v))
	}
	fmt.Println("len: ", len(words))
	fmt.Println("len: ", len(visited))
	lock.Unlock()
	time.Sleep(time.Second)
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
			c := colly.NewCollector()
			c.OnHTML("a[href]", func(e *colly.HTMLElement) {
				l, err := url.Parse(e.Request.AbsoluteURL(e.Attr("href")))
				if err == nil {
					vlock.Lock()
					if _,ok := visited[l.String()]; !ok {
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
			fmt.Println("VISITING: ",u.String())
			err := c.Visit(u.String())
			if err != nil {
				logrus.Error("VISIT: ",u.String(),err)
			}
		}
	}
}
