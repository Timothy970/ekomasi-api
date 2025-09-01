package handlers

import (
	"adenzo_backend/dtos"
	"adenzo_backend/models"
	"adenzo_backend/notification"
	"adenzo_backend/utils"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Delete Blog
// @Summary Create notification
// @Description Create notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications [POST]
func CreateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)

	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	req, ok := DecodeRequestBody[dtos.Notification](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	//check if user exists
	exists, err := models.RecordExists("users", "user_id = ?", req.RecipientID)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	if !exists {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "user not found",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	//check channle
	if req.Channel != "email" && req.Channel != "sms" && req.Channel != "whatsapp" && req.Channel != "push" {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   "Invalid notification type",
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	err = sendNotification(*req)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	req.SentAt = time.Now().Format("2006-01-02 15:04:05")
	if err := models.CreateNotification(*req); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Notification created successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
func sendNotification(req dtos.Notification) error {
	user, err := models.GetUserByUserID(req.RecipientID)
	if err != nil {
		return err
	}
	if (req.Channel == "whatsapp" || req.Channel == "sms") && user.Phone == "" {
		return errors.New("user doesn't have a phone number")
	}
	switch req.Channel {
	case "sms":
		notification.SendSmsMessages(user.Phone, req.Content)
	case "whatsapp":
		//to include template name
		phoneInt, _ := strconv.Atoi(user.Phone)
		notification.SendWhatsappMessages(phoneInt, req.Content, "Notification")
	case "email":
		notification.SendEmail(user.Email, "Notification", req.Content)
	case "push":
		//handle websocketing sending
		utils.SendToUser(req.RecipientID, "", "", map[string]interface{}{
			"event":   "Notification",
			"message": "Your payment was successful!",
		})
	}

	return nil
}

// @Summary List all Notifications
// @Description List all notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications [GET]
func ListNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	notifications, meta, err := models.ListNotifications(page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.NotificationListResponse{
		Notifications: notifications,
		Meta:          *meta,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   resp,
		Message:   "Notifications fetched successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Update Notification
// @Description Update notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications{notification_id} [PATCH]
func UpdateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	//check if user is admin
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	id := mux.Vars(r)["notification_id"]
	req, ok := DecodeRequestBody[dtos.UpdateNotification](r, w, requestSummary, start)
	if !ok {
		return
	}

	//Validate the request
	if !utils.ValidateStructAndRespond(req, w, r, requestSummary, start) {
		return
	}
	if err := models.UpdateNotification(req.Status, id); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Notification updated successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary Delete Notification
// @Description Delete notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications{notification_id} [DELETE]
func DeleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	id := mux.Vars(r)["notification_id"]
	if err := models.DeleteNotification(id); err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}
	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   nil,
		Message:   "Notification deleted successfully",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}

// @Summary List all Notifications
// @Description List all notification
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/admin/notifications [GET]
func ListLogsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Read and restore body FIRST
	requestSummary := utils.GetRequestSummary(r)
	_, ok := utils.RequireAdmin(r, w, start, requestSummary)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	logs, meta, err := models.ListLogs(page, limit)
	if err != nil {
		utils.RespondWithError(w, utils.ErrorJSONResponseOptions{
			Code:      http.StatusInternalServerError,
			Message:   err.Error(),
			TimeTaken: time.Since(start),
			Function:  utils.GetCurrentFuncName(),
			Request:   r,
			RawBody:   requestSummary})
		return
	}

	resp := dtos.LogListResponse{
		Logs: logs,
		Meta: *meta,
	}

	utils.RespondWithJSON(w, utils.SuccessJSONResponseOptions{
		Code:      http.StatusCreated,
		Payload:   resp,
		Message:   "Logs",
		TimeTaken: time.Since(start),
		Function:  utils.GetCurrentFuncName(),
		Request:   r,
		RawBody:   requestSummary})
}
