package rest_api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"job-schedular-service/internal/core/domain"
	"job-schedular-service/internal/core/ports"
	"job-schedular-service/internal/core/services"
	"net/http"
	"strconv"
)

type Handler struct {
	scheduler *services.SchedulerService
	processor *services.ProcessorService
}

func NewHandler(scheduler *services.SchedulerService, processor *services.ProcessorService) *Handler {
	return &Handler{
		scheduler: scheduler,
		processor: processor,
	}
}

func (h *Handler) ScheduleJob(c *gin.Context) {
	var req services.ScheduleJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.scheduler.ScheduleJob(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, job)
}

func (h *Handler) GetJob(c *gin.Context) {
	jobID := c.Param("id")
	if _, err := uuid.Parse(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	job, err := h.scheduler.GetJob(c.Request.Context(), uuid.UUID(uuid.MustParse(jobID)))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *Handler) ListJobs(c *gin.Context) {
	filter := ports.JobFilter{
		Limit:  50,
		Offset: 0,
	}

	if status := c.Query("status"); status != "" {
		s := domain.Status(status)
		filter.Status = &s
	}

	if jobType := c.Query("type"); jobType != "" {
		t := domain.JobType(jobType)
		filter.JobType = &t
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	jobs, err := h.scheduler.ListJobs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (h *Handler) GetQueueStats(c *gin.Context) {
	stats, err := h.scheduler.GetQueueStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
