package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// create a static page
func CreateStaticPage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "HomePage")
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.StaticPageRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "HomePage") {
		return
	}
	err := models.CreateStaticPage(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to create static page" + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusCreated,
			Description: "Static page created successfully",
		},
		Payload:   nil,
		Message:   "Static page created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

func GetStaticPages(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	query := r.URL.Query().Get("q")
	staticPages, err := models.GetStaticPages(query)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to fetch static pages" + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusOK,
			Description: "Static pages fetched successfully",
		},
		Payload:   staticPages,
		Message:   "Static pages fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})

}

// get a static page by id
func GetStaticPageByID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	staticPageID := mux.Vars(r)["static_page_id"]
	staticPage, err := models.GetStaticPageByID(staticPageID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to fetch static page with ID " + staticPageID + ": " + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusOK,
			Description: "Static page with ID" + staticPageID + " fetched successfully",
		},
		Payload:   staticPage,
		Message:   "Static page fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// delete a static page
func DeleteStaticPage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "HomePage")
	if !ok {
		return
	}
	staticPageID := mux.Vars(r)["static_page_id"]
	err := models.DeleteStaticPage(staticPageID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to delete static page with ID " + staticPageID + ": " + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusOK,
			Description: "Static page with ID " + staticPageID + " deleted successfully",
		},
		Payload:   nil,
		Message:   "Static page deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}

// update a static page
func UpdateStaticPage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary, "HomePage")
	if !ok {
		return
	}
	staticPageID := mux.Vars(r)["static_page_id"]
	req, ok := DecodeRequestBody[dtos.StaticPageRequest](r, w, requestSummary, start)
	if !ok {
		return
	}
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start, "HomePage") {
		return
	}
	staticPage, err := models.UpdateStaticPage(staticPageID, *req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			CollectiveInfo: utils.CollectiveInfo{
				Module:      "HomePage",
				Code:        http.StatusNotFound,
				Description: "Failed to update static page with ID " + staticPageID + ": " + err.Error(),
			},
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary,
		})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		CollectiveInfo: utils.CollectiveInfo{
			Module:      "HomePage",
			Code:        http.StatusOK,
			Description: "Static page with ID " + staticPageID + " updated successfully",
		},
		Payload:   staticPage,
		Message:   "Static page updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary,
	})
}
