package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/JLarky/strike-notes/server/db"
	. "github.com/JLarky/strike/pkg/h"
	"github.com/JLarky/strike/pkg/strike"
)

func main() {
	http.Handle("/favicon.ico", http.FileServer(http.Dir("public")))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("public"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		flush := func() {
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
				time.Sleep(1000 * time.Millisecond)
			}
		}

		page := Page(r.URL)

		rsc := r.Header.Get("RSC")
		if rsc == "1" {
			jsonData, err := json.Marshal(page)
			if err != nil {
				fmt.Printf("Error serializing data: %v", err)
				return
			}

			w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
			w.Write(jsonData)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<!doctype html>"))
		// flush()
		err := strike.RenderToString(w, page)
		if err != nil {
			fmt.Printf("Error rendering page: %v", err)
			return
		}
		flush()
		// w.Write([]byte("Hello, World!"))
		jsonData, err := json.Marshal(page)

		if err != nil {
			fmt.Printf("Error serializing data: %v", err)
			return
		}

		const tpl = `<script>self.__rsc=self.__rsc||[];__rsc.push({{.}})</script>`

		t, err := template.New("webpage").Parse(tpl)

		if err != nil {
			fmt.Printf("Error parsing template: %v", err)
			return
		}

		err = t.Execute(w, string(jsonData))

		if err != nil {
			fmt.Printf("Error parsing template: %v", err)
			return
		}
	})
	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func Page(url *url.URL) Component {
	// time.Sleep(1000 * time.Millisecond)
	fmt.Println("Page", url)
	nav := H("nav",
		H("a", Props{"href": "/"}, "Home"), " ",
		H("a", Props{"href": "/about"}, "About"),
	)

	return H("html", Props{"lang": "en"},
		H("head",
			H("meta", Props{"charset": "utf-8"}),
			H("meta", Props{"name": "description", "content": "React with Server Components demo"}),
			H("meta", Props{"name": "viewport", "content": "width=device-width, initial-scale=1"}),
			H("link", Props{"rel": "stylesheet", "href": "/static/style.css"}),
			H("title", "React Notes"),
			H("style", Props{"type": "text/css"}, []template.HTML{`
			strike-slot {
				display: none;
			}
			strike-island {
				display: contents;
			}
			`}),
		),
		H("body",
			App(url),
			H("div", Props{"id": "root"}, nav, "Loading... "+url.Path),
			H("script", Props{"type": "importmap"}, []template.HTML{`
			{
				"imports": {
					"react": "https://esm.sh/react@canary?dev",
					"react-dom/client": "https://esm.sh/react-dom@canary/client?dev",
					"react/jsx-runtime": "https://esm.sh/react@canary/jsx-runtime?dev",
					"react-error-boundary": "https://esm.sh/react-error-boundary"
				}
			}`}),
			H("script", Props{"src": "/static/strike/bootstrap.js", "type": "module"}),
		),
	)
}

func App(url *url.URL) Component {
	return H("div", Props{"class": "main"},
		H("section", Props{"class": "col sidebar"},
			H("section", Props{"class": "sidebar-header"},
				H("img", Props{"class": "logo", "src": "/static/logo.svg", "width": "22px", "height": "20px", "alt": "", "role": "presentation"}),
				H("strong", "React Notes"),
			),
			H("section", Props{"class": "sidebar-menu", "role": "menubar"},
				SearchField(),
				EditButton(nil, "New"),
			),
			H("nav", NodeList(url)), // <Suspense fallback={<NoteListSkeleton />}>
			// 	<NoteList searchText={searchText} />
			// </Suspense>

		),
		H("section", Props{"class": "col note-viewer"}), // 	<Suspense fallback={<NoteSkeleton isEditing={isEditing} />}>
	// 	<Note selectedId={selectedId} isEditing={isEditing} />
	// </Suspense>

	)
}

func SearchField() Component {
	return Island("SearchField", nil,
		H("form", Props{"class": "search", "role": "search"},
			H("label", Props{"class": "offscreen"}),
			H("input", Props{"placeholder": "Search", "disabled": "disabled"}),
		),
	)
}

func EditButton(noteId *string, title string) Component {
	return Island("EditButton", Props{"noteId": noteId, "title": title},
		H("button", Props{"class": "edit-button edit-button--solid", "role": "menuitem"}, "New"),
	)
}

func NodeList(url *url.URL) Component {
	q := url.Query().Get("q")
	notes, err := db.SearchNotes(q)
	if err != nil {
		panic(fmt.Sprintf("Error searching notes: %v", err))
	}
	if (len(notes)) == 0 {
		text := "No notes created yet!"
		if q != "" {
			text = fmt.Sprintf(`Couldn't find any notes titled "%s".`, q)
		}
		return H("div", Props{"class": "notes-empty"}, text)
	}
	noteComponents := make([]Component, len(notes))
	for i, note := range notes {
		noteComponents[i] = H("li", Props{"key": note.Id}, SidebarNote(note))
	}
	return H("ul", Props{"class": "notes-list"}, noteComponents)
}

func SidebarNote(note db.Note) Component {
	isToday := func(t time.Time) bool {
		now := time.Now()
		return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
	}

	lastEdited := ""

	if isToday(note.UpdatedAt) {
		lastEdited = note.UpdatedAt.Format("3:04 PM")
	} else {
		lastEdited = note.UpdatedAt.Format("1/_2/06")
	}

	return Island("SidebarNoteContent", Props{"id": note.Id, "title": note.Title},
		H("div", Props{"class": "sidebar-note-list-item"},
			H("header", Props{"class": "sidebar-note-header"},
				H("strong", note.Title),
				H("small", lastEdited),
			),
			H("button", Props{"class": "sidebar-note-open"}),
		),
		H("strike-slot",
			H("p", Props{"class": "sidebar-note-excerpt"}, H("i", "(No content)")),
		),
	)
}

func Island(componentName string, props Props, children ...any) Component {
	return H("strike-island", props, Props{"component-export": componentName}, children)
}
