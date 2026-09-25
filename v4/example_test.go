package raptor_test

import (
	"encoding/json/v2"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/errs"
	"github.com/go-raptor/raptor/v4/router"
)

// The code in this file is shown in README.md; keep the two in sync.

type Note struct {
	ID    int    `json:"id"`
	Owner string `json:"-"`
	Text  string `json:"text"`
}

type NotesService struct {
	raptor.Service

	mu    sync.Mutex
	notes []Note
}

func (s *NotesService) Create(owner, text string) Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	note := Note{ID: len(s.notes) + 1, Owner: owner, Text: text}
	s.notes = append(s.notes, note)
	return note
}

// Find scopes the lookup to owner, so another user's note is simply not found.
func (s *NotesService) Find(owner string, id int) (Note, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range s.notes {
		if n.ID == id && n.Owner == owner {
			return n, true
		}
	}
	return Note{}, false
}

type NotesController struct {
	raptor.Controller

	Notes *NotesService
}

func (c *NotesController) Create(ctx *raptor.Context) error {
	var in struct {
		Text string `json:"text"`
	}
	if err := ctx.Bind(&in); err != nil {
		return errs.NewErrorBadRequest("Invalid JSON")
	}
	return ctx.Data(c.Notes.Create(ctx.Get("user").(string), in.Text), http.StatusCreated)
}

func (c *NotesController) Show(ctx *raptor.Context) error {
	id, _ := strconv.Atoi(ctx.Param("id"))
	note, ok := c.Notes.Find(ctx.Get("user").(string), id)
	if !ok {
		return errs.NewErrorNotFound("Note not found")
	}
	return ctx.Data(note)
}

// TokenAuthMiddleware stands in for real authentication: the bearer token is the user name.
type TokenAuthMiddleware struct {
	raptor.Middleware
}

func (m *TokenAuthMiddleware) Handle(ctx *raptor.Context, next func(*raptor.Context) error) error {
	user, ok := strings.CutPrefix(ctx.Request().Header.Get("Authorization"), "Bearer ")
	if !ok || user == "" {
		return errs.ErrUnauthorized
	}
	ctx.Set("user", user)
	return next(ctx)
}

// FilesController is the README's file-serving example; it only needs to compile.
type FilesController struct {
	raptor.Controller
}

func (c *FilesController) Download(ctx *raptor.Context) error {
	name := ctx.Param("name")
	ctx.Response().Header().Set("Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	return ctx.FileFromDir("storage/uploads", name)
}

func TestNotesAreScopedToTheirOwner(t *testing.T) {
	app := raptor.NewTestApp(&raptor.Components{
		Services:    raptor.Services{&NotesService{}},
		Controllers: raptor.Controllers{&NotesController{}},
		Middlewares: raptor.Middlewares{raptor.Use(&TokenAuthMiddleware{})},
	}, router.CollectRoutes(
		router.Post("/notes", "Notes.Create"),
		router.Get("/notes/{id}", "Notes.Show"),
	))
	alice := raptor.WithHeader("Authorization", "Bearer alice")
	bob := raptor.WithHeader("Authorization", "Bearer bob")

	rec := app.TestPost("/notes", strings.NewReader(`{"text":"hi"}`), alice)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d %s", rec.Code, rec.Body)
	}
	var note struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &note); err != nil {
		t.Fatal(err)
	}

	path := fmt.Sprintf("/notes/%d", note.ID)
	if rec := app.TestGet(path, alice); rec.Code != http.StatusOK {
		t.Fatalf("owner read: got %d", rec.Code)
	}
	if rec := app.TestGet(path, bob); rec.Code != http.StatusNotFound {
		t.Fatalf("another user's note must be a 404, got %d", rec.Code)
	}
	if svc := raptor.GetService[NotesService](app); svc == nil || len(svc.notes) != 1 {
		t.Fatal("GetService must return the live NotesService")
	}
}
