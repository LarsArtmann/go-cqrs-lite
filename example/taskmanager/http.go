package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/httputil"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

const (
	jsonKeyStatus   = "status"
	jsonKeyUpdated  = "updated"
	maxPathSegments = 2
	listPageSize    = 100
)

// taskViewCollection is the metaengine collection behind TaskView reads
// (declared in buildProjections; shared by the reader and the query handler).
const taskViewCollection = "task_views"

// routes builds the HTTP mux with all task management endpoints.
func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)

	mux.Handle("GET /api/tasks", errorfamily.HTTPHandler(
		bindQuery[taskListResult](s.Sys, decodeListTasks, codec.JSONCodec{}),
	))
	mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	mux.HandleFunc("GET /api/tasks/", s.handleTaskSubresource)
	mux.HandleFunc("PATCH /api/tasks/", s.handleTaskSubresource)
	mux.HandleFunc("DELETE /api/tasks/", s.handleTaskSubresource)
	mux.HandleFunc("GET /api/stats", s.handleGetTaskStats)

	// SSE: live TaskView updates from the metaengine read model
	if s.taskWatcher != nil {
		mux.HandleFunc("GET /events", s.handleEvents)
	}

	return mux
}

// handleEvents streams TaskView updates as Server-Sent Events. Clients
// reconnect with the Last-Event-ID header to replay missed updates. Built on
// go-sse via metaengine.ServeSSE (ADR-0127); replaces the deprecated
// transport/http.SSEBroker.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	_ = metaengine.ServeSSE(w, r, s.taskWatcher,
		metaengine.WithSSEHeartbeat(sseHeartbeatSecs*time.Second),
	)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{jsonKeyStatus: "ok"})
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
		BasicCommand: Must(command.New(cmdCreateTask, taskID)),
		Title:        body.Title,
		Description:  body.Description,
		Priority:     body.Priority,
	}); err != nil {
		writeCQRSError(w, err)

		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": taskID.String()})
}

// apiListTasksQuery is the typed query behind GET /api/tasks — registered
// as "api_list_tasks" in DomainConfig.Queries (setup.go).
type apiListTasksQuery struct {
	Status string
	Limit  int
}

func (apiListTasksQuery) Type() query.Type { return "api_list_tasks" }

// taskListResult preserves the pre-§2.47 response shape: {"tasks": …, "count": …}.
type taskListResult struct {
	Tasks []*TaskView `json:"tasks"`
	Count int         `json:"count"`
}

// decodeListTasks turns query params into the typed query. The limit is
// clamped to the page ceiling — the client never sizes the scan.
func decodeListTasks(r *http.Request) (apiListTasksQuery, error) {
	limit := int(httputil.ParseUintQuery(r, "limit"))
	if limit == 0 {
		limit = listPageSize
	}
	if limit > listPageSize {
		limit = listPageSize
	}

	return apiListTasksQuery{Status: r.URL.Query().Get("status"), Limit: limit}, nil
}

func (s *Server) handleTaskSubresource(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.SplitN(path, "/", maxPathSegments)
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
				DeleteTaskCmd{BasicCommand: Must(command.New(cmdDeleteTask, taskID))},
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
			BasicCommand: Must(
				command.New(cmdAssignTask, taskID),
			),
			AssigneeID: NewAssigneeID(body.AssigneeID),
		})

	case "start":
		s.dispatchSimple(
			w,
			r,
			taskID,
			StartTaskCmd{BasicCommand: Must(command.New(cmdStartTask, taskID))},
		)

	case "complete":
		s.dispatchSimple(
			w,
			r,
			taskID,
			CompleteTaskCmd{BasicCommand: Must(command.New(cmdCompleteTask, taskID))},
		)

	case "archive":
		s.dispatchSimple(
			w,
			r,
			taskID,
			ArchiveTaskCmd{BasicCommand: Must(command.New(cmdArchiveTask, taskID))},
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
				BasicCommand: Must(command.New(cmdAddBlocker, taskID)), DependencyID: depID,
			})
		}

	default:
		writeError(w, http.StatusNotFound, "unknown sub-resource: "+action)
	}
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, taskID id.StreamID) {
	view, found, err := s.TaskReader.Get(r.Context(), taskID.Get())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get task: "+err.Error())

		return
	}

	if !found {
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
			BasicCommand: Must(command.New(cmdUpdateTitle, taskID)), Title: body.Title,
		}); err != nil {
			writeCQRSError(w, err)

			return
		}
	}

	if body.Priority != "" {
		if err := s.CmdDisp.Dispatch(r.Context(), ChangePriorityCmd{
			BasicCommand: Must(command.New(cmdChangePrio, taskID)), Priority: body.Priority,
		}); err != nil {
			writeCQRSError(w, err)

			return
		}
	}

	if body.DueDate != "" {
		dueDate, err := time.Parse(time.RFC3339, body.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueDate (use RFC3339)")

			return
		}

		if err := s.CmdDisp.Dispatch(r.Context(), SetDueDateCmd{
			BasicCommand: Must(command.New(cmdSetDueDate, taskID)), DueDate: &dueDate,
		}); err != nil {
			writeCQRSError(w, err)

			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{jsonKeyStatus: jsonKeyUpdated})
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

	writeJSON(w, http.StatusOK, map[string]string{jsonKeyStatus: "ok"})
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

	defer r.Body.Close()

	return json.UnmarshalDecode(dec, dst, json.RejectUnknownMembers(true))
}
