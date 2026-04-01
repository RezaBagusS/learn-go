package handler

import (
	"belajar-go/projectAPI/dto"
	"belajar-go/projectAPI/helper"
	"belajar-go/projectAPI/internal/api/task/repository"
	"belajar-go/projectAPI/internal/api/task/service"
	"belajar-go/projectAPI/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
)

type TaskHandler struct {
	mux *http.ServeMux
	svc service.TaskService
}

func NewTaskHandler(mux *http.ServeMux, db *sqlx.DB) *TaskHandler {
	taskRepo := repository.NewTaskRepository(db)
	taskSvc := service.NewTaskService(taskRepo)

	return &TaskHandler{
		mux: mux,
		svc: taskSvc,
	}
}

func (a *TaskHandler) MapRoutes() {
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodGet, "/tasks"),
		a.GetAll(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPost, "/tasks"),
		a.Create(),
	)
	a.mux.HandleFunc(
		helper.NewAPIPath(http.MethodPatch, "/tasks/{id}"),
		a.Patch(),
	)
}

// GET /tasks
func (h *TaskHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		tasks, err := h.svc.FetchAllTasks()
		if err != nil {
			dto.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusOK, "Berhasil mengambil data task", map[string]any{
			"tasks": tasks,
		})
	}
}

// POST /tasks
func (h *TaskHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var payload models.Task
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format JSON tidak valid!")
			return
		}

		newTask, err := h.svc.CreateNewTask(payload)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		// fmt.Println("New Task created : ", newTask)
		dto.WriteResponse(w, http.StatusCreated, "Berhasil membuat data task", map[string]any{
			"tasks": newTask,
		})
	}
}

// PATCH /tasks/{id}
func (h *TaskHandler) Patch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// ✅ Ambil id dari path
		idStr := r.PathValue("id")
		fmt.Printf("Id received [task][handler]: %v \n", idStr)
		if idStr == "" {
			dto.WriteError(w, http.StatusBadRequest, "ID tidak boleh kosong!")
			return
		}

		// ✅ Konversi string ke int
		id, err := strconv.Atoi(idStr)
		fmt.Printf("Id convertion from str to int [task][handler]: %v \n", id)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format ID tidak valid!")
			return
		}

		var payload models.Task
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format JSON tidak valid!")
			return
		}

		payload.ID = id
		fmt.Printf("Payload [task][handler]: %v \n", payload)

		getId, err := h.svc.PatchTaskById(payload)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusOK, "Berhasil update data task", map[string]any{
			"id": getId,
		})
	}
}

// Delete /tasks/{id}
func (h *TaskHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// ✅ Ambil id dari path
		idStr := r.PathValue("id")
		fmt.Printf("Id received [task][handler]: %v \n", idStr)
		if idStr == "" {
			dto.WriteError(w, http.StatusBadRequest, "ID tidak boleh kosong!")
			return
		}

		// ✅ Konversi string ke int
		id, err := strconv.Atoi(idStr)
		fmt.Printf("Id convertion from str to int [task][handler]: %v \n", id)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, "Format ID tidak valid!")
			return
		}

		err = h.svc.DeleteTaskById(id)
		if err != nil {
			dto.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		dto.WriteResponse(w, http.StatusOK, "Berhasil update data task", map[string]any{
			"id": 0,
		})
	}
}
