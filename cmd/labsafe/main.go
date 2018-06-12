package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/hallazzang/labsafe"
	"github.com/pkg/errors"
)

var id, pw string

func doProgress(c *labsafe.Client, p *labsafe.Progress, interval time.Duration) error {
	switch p.Type {
	case labsafe.NormalContent:
		Info("%s - started", p.Name)

		pages, err := c.GetTotalPages(p.No)
		if err != nil {
			return errors.Wrap(err, "cannot get total pages")
		}

		wg := &sync.WaitGroup{}
		for page := 1; page <= pages; page++ {
			time.Sleep(600 * time.Millisecond)

			cc, err := labsafe.NewClient()
			if err != nil {
				return errors.Wrap(err, "cannot create new client")
			}
			if _, err := cc.Login(id, pw); err != nil {
				return errors.Wrap(err, "cannot login")
			}

			wg.Add(1)
			go func(page int) {
				defer wg.Done()

				for {
					suc, _, err := cc.ViewNormal(p.No, page, interval)
					if err != nil {
						Error("%s page %d - cannot bypass: %v", p.Name, page, err)
					} else if !suc {
						Error("%s page %d - failed", p.Name, page)
					} else {
						break
					}

					Debug("%s page %d - retrying", p.Name, page)
				}
			}(page)
		}

		wg.Wait()
		Info("%s - finished", p.Name)
	case labsafe.VideoContent:
		Info("%s - started", p.Name)

		if suc, err := c.ViewVideo(p.No); err != nil {
			return errors.Wrap(err, "cannot bypass")
		} else if !suc {
			return errors.New("failed")
		}

		Info("%s - finished", p.Name)
	}

	return nil
}

func init() {
	if len(os.Args) != 3 {
		fmt.Fprintln(color.Output, color.GreenString("Usage:"), os.Args[0], "ID PW")
		os.Exit(0)
	}

	id, pw = os.Args[1], os.Args[2]
}

func main() {
	c, err := labsafe.NewClient()
	if err != nil {
		Fatal("cannot create new client: %v", err)
	}

	if suc, err := c.Login(id, pw); err != nil {
		Fatal("cannot login: %v", err)
	} else if !suc {
		Fatal("login failed - wrong id or pw")
	}

	Info("bypassing started")

	ps, err := c.GetProgresses()
	if err != nil {
		Fatal("cannot get progresses: %v", err)
	}

	for _, p := range ps {
		var typ string
		switch p.Type {
		case labsafe.NormalContent:
			typ = "normal"
		case labsafe.VideoContent:
			typ = "video"
		}
		if p.Taken {
			Debug("[%s] %s - done", typ, p.Name)
		} else if p.No == "" {
			Debug("[%s] %s - not opened yet", typ, p.Name)
		} else {
			Debug("[%s] %s - in progress", typ, p.Name)
		}
	}

	for {
		done := true

		for _, p := range ps {
			if !p.Taken && p.No != "" {
				err := doProgress(c, &p, 10*time.Second)
				if err != nil {
					Error("%s - failed: %v", err)
				}
				done = false
			}
		}

		ps, err = c.GetProgresses()
		if err != nil {
			Fatal("cannot get progresses: %v", err)
		}

		if done {
			break
		}
	}

	Info("expoiting exam")

	if suc, err := c.ExamExploit(); err != nil {
		Fatal("cannot exploit exam: %v", err)
	} else if !suc {
		Fatal("exploiting exam failed - unknown reason")
	}

	Info("all done")
}
