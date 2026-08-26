package handler

import (
	"golang-s3-batch-uploader/errs"
	"golang-s3-batch-uploader/service"

	"github.com/gofiber/fiber/v2"
)

type batchHandler struct {
	batchSvc service.BatchService
}

func NewBatchHandler(batchSvc service.BatchService) batchHandler {
	return batchHandler{batchSvc: batchSvc}
}

func (h batchHandler) Run(c *fiber.Ctx) error {
	var req service.BatchRequest
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, errs.NewValidationError("invalid request body"))
	}
	if req.SourceDir == "" {
		return handleError(c, errs.NewValidationError("source_dir is required"))
	}

	result, err := h.batchSvc.Run(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return sendSuccess(c, fiber.StatusOK, "batch processed", result)
}
