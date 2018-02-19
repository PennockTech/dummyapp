// Copyright © 2017,2018 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// This is a mostly-lightweight webserver as a dummy app; it imports an
// external package or two mostly to have an excuse to use dep, for the
// dummy demonstration.
//
// Real code might use better structured work, or even a real middleware
// setup.  This is in one file.  It could be made much simpler, but I
// wanted proper logging of startup, respawn loop delay protection,
// and something which can serve from a filesystem area, to highlight
// other parts of the build system.

package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

const metadataBase = "http://169.254.169.254/latest/meta-data/"

// AWS metadata service is very local, we can hard-timeout much sooner
const awsHTTPTimeout = 3 * time.Second

type awsGatherItem struct {
	path string
	body []byte
	err  error
}

// doAWSGather spawns go-routines to collect all the given relative paths
// from the AWS metadata service and waits for either them, or a context
// expiry.  The results collected before expiration are returned as a map.
func doAWSGather(ctx context.Context, sections ...string) map[string]*awsGatherItem {
	wg := &sync.WaitGroup{}
	items := make(map[string]*awsGatherItem, len(sections))
	results := make(chan *awsGatherItem, len(sections))

	for _, section := range sections {
		wg.Add(1)
		go func(sect string) {
			defer wg.Done()
			results <- awsCollect(ctx, sect)
		}(section)
	}

	waited := make(chan struct{})
	go func() {
		wg.Wait()
		close(waited)
	}()
	resultOrClosed := results
collect1:
	for {
		select {
		case r, ok := <-resultOrClosed:
			if ok {
				items[r.path] = r
			} else {
				resultOrClosed = nil
			}
		case <-waited:
			break collect1
		case <-ctx.Done():
			break collect1
		}
	}
	go func() {
		<-waited
		close(results)
	}()
	// We can collect all the results and close the channel before reading them,
	// while the select above will _randomly_ choose one of the branches, so we
	// can be left with items in the channel.  (Also, context could expire
	// badly timed, but that's much less likely).
	// The items are placed in the channel before the waitgroup is marked done,
	// so we can select on a read and only get items already in the channel,
	// relying upon the default (not in the random selection) to stop when the
	// channel has been empty of whatever's in there (or just squeaked in after
	// context expiry).
collect2:
	for {
		select {
		case r, ok := <-results:
			if !ok {
				break collect2
			}
			items[r.path] = r
		default:
			break collect2
		}
	}

	return items
}

func awsCollect(ctx context.Context, path string) *awsGatherItem {
	// This function is a place where named return values make sense, but using
	// them is controversial.  I probably upset purists enough by insisting on
	// using ALL_CAPS for package constants, so go with the flow this time.
	r := &awsGatherItem{
		path: path,
	}
	u := metadataBase + path

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		r.err = err
		return r
	}
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		r.err = err
		return r
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		r.err = err
		return r
	}
	r.body = body
	return r
}

func rawAWSAddSection(w io.Writer, item *awsGatherItem) error {
	_, err := fmt.Fprintf(w, "\n<h3>%s</h3>\n", template.HTMLEscapeString(item.path))
	if err == nil {
		template.HTMLEscape(w, item.body)
		// no error return, oops
	}
	return err
}

func renderErrorToHTML(w io.Writer, path string, err error) {
	fmt.Fprintf(w, "\n<h3 class=\"error\">%s</h3>\n<div class=\"error errmsg\">%s</div>\n",
		template.HTMLEscapeString(path), template.HTMLEscapeString(err.Error()))
}

func addAWSSection(w io.Writer, item *awsGatherItem) {
	if item.err != nil {
		renderErrorToHTML(w, item.path, item.err)
		return
	}
	// this handling is a little overkill, now that the data gathering is done ahead
	// of time in a go-routine
	if err := rawAWSAddSection(w, item); err != nil {
		renderErrorToHTML(w, item.path, err)
	}
}

func awsHandle(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "<html><head><title>AWS Info Dumper</title></head><body><h1>AWS Info Dumper</h1>\n")

	childCtx, cancel := context.WithTimeout(req.Context(), awsHTTPTimeout)
	defer cancel()

	if p := os.Getenv("ECS_CONTAINER_METADATA_FILE"); p != "" {
		io.WriteString(w, "<h2>ECS metadata from file</h2>\n")
		contents, err := ioutil.ReadFile(p)
		if err != nil {
			renderErrorToHTML(w, p, err)
		} else {
			fmt.Fprintf(w, "\n<h3>%s</h3>\n", template.HTMLEscapeString(p))
			template.HTMLEscape(w, contents)
		}

	} else {
		io.WriteString(w, "<h2>AWS metadata service (HTTP requests)</h2>\n")
		sections := []string{
			"hostname",
			"placement/availability-zone",
			"iam/info",
		}
		items := doAWSGather(childCtx, sections...)
		for _, section := range sections {
			// may have timed out before collecting them all
			if item, ok := items[section]; ok {
				addAWSSection(w, item)
			}
		}
		if childCtx.Err() != nil {
			// any context expiration has _almost_ certainly been shown in the output of the
			// addAWSSection error handling; there's a few nanoseconds race, so rather than
			// risk aborting early without saying so, just explicitly say "hey we're done".
			renderErrorToHTML(w, "timeout", fmt.Errorf("terminated early"))
		}

	}
}

func init() { addFirstLevelPageFunc("aws", awsHandle) }
