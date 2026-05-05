package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db          *gorm.DB
	redisClient *redis.Client
}

func NewHealthHandler(db *gorm.DB, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
	}
}

func (h *HealthHandler) Check(c *fiber.Ctx) error {
	status := "UP"
	components := make(map[string]string)

	// Check Database
	sqlDB, err := h.db.DB()
	if err == nil && sqlDB.Ping() == nil {
		components["database"] = "UP"
	} else {
		components["database"] = "DOWN"
		status = "DOWN"
	}

	// Check Redis
	if err := h.redisClient.Ping(context.Background()).Err(); err == nil {
		components["redis"] = "UP"
	} else {
		components["redis"] = "DOWN"
		status = "DOWN"
	}

	response := fiber.Map{
		"status":     status,
		"components": components,
	}

	if status == "DOWN" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(response)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
