// Command site serves lovyou.ai — the home of the hive's products.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/lovyou-ai/site/content"
	"github.com/lovyou-ai/site/views"
	"github.com/lovyou-ai/site/work"
)

func main() {
	port := flag.String("port", "", "HTTP port (default: $PORT or 8080)")
	flag.Parse()

	p := *port
	if p == "" {
		p = os.Getenv("PORT")
	}
	if p == "" {
		p = "8080"
	}

	// Load content.
	posts, err := content.LoadPosts()
	if err != nil {
		log.Fatalf("load posts: %v", err)
	}
	log.Printf("loaded %d blog posts", len(posts))

	layers := content.LoadLayers()
	agentPrims := content.LoadAgentPrimitives()

	// Build lookups for individual pages.
	primsBySlug := map[string]views.Primitive{}
	layersByNum := map[int]views.Layer{}
	totalPrims := 0
	for _, layer := range layers {
		layersByNum[layer.Number] = layer
		totalPrims += len(layer.Primitives)
		for _, prim := range layer.Primitives {
			primsBySlug[prim.Slug] = prim
		}
	}
	for _, prim := range agentPrims {
		primsBySlug[prim.Slug] = prim
	}
	log.Printf("loaded %d layers, %d primitives, %d agent primitives",
		len(layers), totalPrims, len(agentPrims))

	grammars, err := content.LoadGrammars()
	if err != nil {
		log.Fatalf("load grammars: %v", err)
	}
	log.Printf("loaded %d grammars", len(grammars))

	// Blog handlers.
	handleHome, handleBlogIndex, handleBlogPost := makeHandlers(posts)

	mux := http.NewServeMux()

	// Static files.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Pages.
	mux.HandleFunc("GET /{$}", handleHome)
	mux.HandleFunc("GET /blog", handleBlogIndex)
	mux.HandleFunc("GET /blog/{slug}", handleBlogPost)

	// Reference.
	mux.HandleFunc("GET /reference", func(w http.ResponseWriter, r *http.Request) {
		views.ReferenceIndex(layers, agentPrims).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/layers/{num}", func(w http.ResponseWriter, r *http.Request) {
		num, err := strconv.Atoi(r.PathValue("num"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if layer, ok := layersByNum[num]; ok {
			views.LayerPage(layer, layers).Render(r.Context(), w)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /reference/agents", func(w http.ResponseWriter, r *http.Request) {
		views.AgentPrimitivesPage(agentPrims).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/primitives/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if prim, ok := primsBySlug[slug]; ok {
			views.PrimitivePage(prim).Render(r.Context(), w)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /reference/grammars", func(w http.ResponseWriter, r *http.Request) {
		views.GrammarIndex(grammars).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/grammars/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		for _, g := range grammars {
			if g.Slug == slug {
				views.GrammarPage(g, grammars).Render(r.Context(), w)
				return
			}
		}
		http.NotFound(w, r)
	})

	// Work product.
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		workStore, err := work.New(dsn)
		if err != nil {
			log.Fatalf("work store: %v", err)
		}
		defer workStore.Close()
		workHandlers := work.NewHandlers(workStore)
		workHandlers.Register(mux)
		log.Println("work product enabled (DATABASE_URL set)")
	} else {
		log.Println("work product disabled (no DATABASE_URL)")
		mux.HandleFunc("GET /work", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Work product requires DATABASE_URL", http.StatusServiceUnavailable)
		})
	}

	// Health check for Fly.io.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	addr := ":" + p
	log.Printf("lovyou.ai listening on %s", addr)
	if err := http.ListenAndServe(addr, noCache(mux)); err != nil {
		log.Fatal(err)
	}
}

// ────────────────────────────────────────────────────────────────────
// Middleware
// ────────────────────────────────────────────────────────────────────

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		next.ServeHTTP(w, r)
	})
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func makeHandlers(posts []views.Post) (home, blogIndex, blogPost http.HandlerFunc) {
	home = func(w http.ResponseWriter, r *http.Request) {
		views.Home().Render(r.Context(), w)
	}
	blogIndex = func(w http.ResponseWriter, r *http.Request) {
		views.BlogIndex(posts).Render(r.Context(), w)
	}
	blogPost = func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		for i, post := range posts {
			if post.Slug == slug {
				var nav views.PostNav
				if i > 0 {
					nav.Prev = &posts[i-1]
				}
				if i < len(posts)-1 {
					nav.Next = &posts[i+1]
				}
				views.BlogPost(post, nav).Render(r.Context(), w)
				return
			}
		}
		http.NotFound(w, r)
	}
	return
}
