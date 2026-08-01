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

func faqItems() []any {
	return []any{
		map[string]any{
			"id":      "shipping",
			"title":   "How fast is shipping?",
			"content": "<p>Orders ship within two business days.</p>",
		},
		map[string]any{
			"id":      "returns",
			"title":   "What is your return policy?",
			"content": "<p>Unused items may be returned within 30 days.</p>",
		},
	}
}

func moreFaqItems() []any {
	return []any{
		map[string]any{
			"id":      "warranty",
			"title":   "Is there a warranty?",
			"content": "<p>Every product carries a one-year limited warranty.</p>",
		},
	}
}

func tabItems() []any {
	return []any{
		map[string]any{
			"id":      "overview",
			"label":   "Overview",
			"content": "<p>Radix-inspired, headless, zero-JS-baseline components for htmlc.</p>",
		},
		map[string]any{
			"id":      "usage",
			"label":   "Usage",
			"content": "<p>Mount the package, then reference its components by any documented form.</p>",
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
