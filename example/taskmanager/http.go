package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	gomust "github.com/larsartmann/go-must"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"
)

// routes builds the HTTP mux with all task management endpoints.
func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)

	mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	mux.HandleFunc("GET /api/tasks/", s.handleTaskSubresource)
	mux.HandleFunc("PATCH /api/tasks/", s.handleTaskSubresource)
	mux.HandleFunc("DELETE /api/tasks/", s.handleTaskSubresource)

	// SSE: real-time event stream for clients
	if s.sseBroker != nil {
		mux.Handle("GET /events", cqrshttp.SSEHandler(s.sseBroker))
	}

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Priority    Priority `json:"priority"`
	}

	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if body.Priority == "" {
		body.Priority = PriorityMedium
	}

	taskID := id.NewStreamID()

	if err := s.CmdDisp.Dispatch(r.Context(), CreateTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdCreateTask, taskID)),
		Title:        body.Title,
		Description:  body.Description,
		Priority:     body.Priority,
	}); err != nil {
		writeCQRSError(w, err)

		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": taskID.String()})
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")

	all, err := s.Mat.List(r.Context(), stack.ExcludeTombstoned)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list tasks: "+err.Error())

		return
	}

	result := make([]*TaskView, 0, len(all))

	for _, t := range all {
		if statusFilter != "" && string(t.Status) != statusFilter {
			continue
		}

		result = append(result, t)
	}

	writeJSON(w, http.StatusOK, map[string]any{"tasks": result, "count": len(result)})
}

func (s *Server) handleTaskSubresource(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.SplitN(path, "/", 2)
	taskIDStr := parts[0]

	taskID, err := id.ParseStreamID(taskIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task ID")

		return
	}

	if len(parts) == 1 || parts[1] == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleGetTask(w, r, taskID)
		case http.MethodPatch:
			s.handlePatchTask(w, r, taskID)
		case http.MethodDelete:
			s.dispatchSimple(
				w,
				r,
				taskID,
				DeleteTaskCmd{BasicCommand: gomust.Must(command.New(cmdDeleteTask, taskID))},
			)
		}

		return
	}

	action := parts[1]

	switch action {
	case "assign":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")

			return
		}

		var body struct {
			AssigneeID string `json:"assigneeId"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())

			return
		}

		s.dispatchSimple(w, r, taskID, AssignTaskCmd{
			BasicCommand: gomust.Must(
				command.New(cmdAssignTask, taskID),
			),
			AssigneeID: body.AssigneeID,
		})

	case "start":
		s.dispatchSimple(
			w,
			r,
			taskID,
			StartTaskCmd{BasicCommand: gomust.Must(command.New(cmdStartTask, taskID))},
		)

	case "complete":
		s.dispatchSimple(
			w,
			r,
			taskID,
			CompleteTaskCmd{BasicCommand: gomust.Must(command.New(cmdCompleteTask, taskID))},
		)

	case "archive":
		s.dispatchSimple(
			w,
			r,
			taskID,
			ArchiveTaskCmd{BasicCommand: gomust.Must(command.New(cmdArchiveTask, taskID))},
		)

	case "blockers":
		if r.Method == http.MethodPost {
			var body struct {
				DependencyID string `json:"dependencyId"`
			}
			if err := decodeJSON(r, &body); err != nil {
				writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())

				return
			}

			depID, dErr := id.ParseStreamID(body.DependencyID)
			if dErr != nil {
				writeError(w, http.StatusBadRequest, "invalid dependency ID")

				return
			}

			s.dispatchSimple(w, r, taskID, AddBlockerCmd{
				BasicCommand: gomust.Must(command.New(cmdAddBlocker, taskID)), DependencyID: depID,
			})
		}

	default:
		writeError(w, http.StatusNotFound, "unknown sub-resource: "+action)
	}
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, taskID id.StreamID) {
	view, err := s.Mat.View(r.Context(), taskID)
	if err != nil || view == nil {
		writeError(w, http.StatusNotFound, "task not found")

		return
	}

	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handlePatchTask(w http.ResponseWriter, r *http.Request, taskID id.StreamID) {
	var body struct {
		Title    string   `json:"title"`
		Priority Priority `json:"priority"`
		DueDate  string   `json:"dueDate"`
	}

	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())

		return
	}

	if body.Title != "" {
		if err := s.CmdDisp.Dispatch(r.Context(), UpdateTitleCmd{
			BasicCommand: gomust.Must(command.New(cmdUpdateTitle, taskID)), Title: body.Title,
		}); err != nil {
			writeCQRSError(w, err)

			return
		}
	}

	if body.Priority != "" {
		if err := s.CmdDisp.Dispatch(r.Context(), ChangePriorityCmd{
			BasicCommand: gomust.Must(command.New(cmdChangePrio, taskID)), Priority: body.Priority,
		}); err != nil {
			writeCQRSError(w, err)

			return
		}
	}

	if body.DueDate != "" {
		dd, err := time.Parse(time.RFC3339, body.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueDate (use RFC3339)")

			return
		}

		if err := s.CmdDisp.Dispatch(r.Context(), SetDueDateCmd{
			BasicCommand: gomust.Must(command.New(cmdSetDueDate, taskID)), DueDate: &dd,
		}); err != nil {
			writeCQRSError(w, err)

			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) dispatchSimple(
	w http.ResponseWriter,
	r *http.Request,
	_ id.StreamID,
	cmd command.Command,
) {
	if err := s.CmdDisp.Dispatch(r.Context(), cmd); err != nil {
		writeCQRSError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── Error mapping ──────────────────────────────────────────────────────────

// writeCQRSError maps the 5-family error taxonomy to HTTP status codes via
// errorfamily.HTTPStatus.
func writeCQRSError(w http.ResponseWriter, err error) {
	writeError(w, errorfamily.HTTPStatus(err), err.Error())
}

// ── JSON helpers ───────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.MarshalWrite(w, body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := jsontext.NewDecoder(r.Body)

	defer func() { _ = r.Body.Close() }()

	return json.UnmarshalDecode(dec, dst, json.RejectUnknownMembers(true))
}

// contextKey is an unexported type for context keys in this package.
type contextKey string

const ctxKeyRequestID contextKey = "requestID"

// loggingMiddleware adds structured logging to every request.
func loggingMiddleware(logger interface{ Info(string, ...any) }, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)
		logger.Info(
			"request",
			"method",
			r.Method,
			"path",
			r.URL.Path,
			"duration",
			time.Since(start),
		)
	})
}
