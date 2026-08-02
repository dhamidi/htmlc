// Command radix-demo is a small, standalone example application proving
// out RFC 014's Options.Mounts mechanism end-to-end: it mounts the in-tree
// github.com/dhamidi/htmlc/ui/radix component package alongside its own
// local ComponentDir and renders a single page that references the mounted
// Accordion/Tabs/Dialog components via every form documented in RFC 014 §5's
// Syntax Summary table, followed by a full gallery showcasing every other
// component in ui/radix. See components/HomePage.vue's own header comment
// for the gallery's structure.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dhamidi/htmlc"
	radixui "github.com/dhamidi/htmlc/ui/radix"
)

// AccordionFAQ is one Accordion/HomePage FAQ entry — the shape rendered by
// Accordion.vue's own `v-for="item in items"` loop (item.id/item.title/
// item.content). Passed as a Go struct slice rather than []map[string]any;
// props.go's StructProps resolves each field against its `json` tag, the
// same convention this project's other example apps (e.g. examples/blog)
// already use for exposing Go structs as template data.
type AccordionFAQ struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// TabItem is one Tabs/HomePage tab entry — the shape rendered by Tabs.vue's
// own `v-for="item in items"` loop (item.id/item.label/item.content).
type TabItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Content string `json:"content"`
}

func faqItems() []AccordionFAQ {
	return []AccordionFAQ{
		{
			ID:      "shipping",
			Title:   "How fast is shipping?",
			Content: "<p>Orders ship within two business days.</p>",
		},
		{
			ID:      "returns",
			Title:   "What is your return policy?",
			Content: "<p>Unused items may be returned within 30 days.</p>",
		},
	}
}

func moreFaqItems() []AccordionFAQ {
	return []AccordionFAQ{
		{
			ID:      "warranty",
			Title:   "Is there a warranty?",
			Content: "<p>Every product carries a one-year limited warranty.</p>",
		},
	}
}

func tabItems() []TabItem {
	return []TabItem{
		{
			ID:      "overview",
			Label:   "Overview",
			Content: "<p>Radix-inspired, headless, zero-JS-baseline components for htmlc.</p>",
		},
		{
			ID:      "usage",
			Label:   "Usage",
			Content: "<p>Mount the package, then reference its components by any documented form.</p>",
		},
	}
}

func pageData() map[string]any {
	return map[string]any{
		"title":        "radix-demo",
		"faqItems":     faqItems(),
		"moreFaqItems": moreFaqItems(),
		"tabItems":     tabItems(),
	}
}

// newEngine constructs the real demo engine: a local "components" directory
// mounted alongside ui/radix under the "radix" prefix, exactly as shown in
// RFC 014 §4.1's "In-tree example and first real consumer" section.
func newEngine() (*htmlc.Engine, error) {
	return htmlc.New(htmlc.Options{
		ComponentDir: "components",
		Mounts: []htmlc.Mount{
			{Prefix: "radix", FS: radixui.FS(), Dir: "components"},
		},
	})
}

func main() {
	engine, err := newEngine()
	if err != nil {
		log.Fatalf("init engine: %v", err)
	}

	// Validate at startup, per this project's documented convention: a
	// cross-mount collision or any other registration problem must fail
	// fast, not surface later as a request-time surprise.
	if errs := engine.ValidateAll(); len(errs) > 0 {
		for _, e := range errs {
			log.Printf("validate: %v", e)
		}
		log.Fatalf("engine validation failed with %d error(s)", len(errs))
	}

	if len(os.Args) > 1 && os.Args[1] == "print" {
		out, err := engine.RenderPageString("HomePage", pageData())
		if err != nil {
			log.Fatalf("render error: %v", err)
		}
		fmt.Println(out)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := engine.RenderPage(r.Context(), w, "HomePage", pageData()); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	})

	addr := ":8080"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
