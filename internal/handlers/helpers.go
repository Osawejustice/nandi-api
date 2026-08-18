package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func pageParams(c *gin.Context) (int, int) {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 0)
	if perPage == 0 {
		perPage = queryInt(c, "limit", 20)
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}

func queryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

type PageMeta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

func writePage(c *gin.Context, items any, total int64, page, perPage int) {
	c.JSON(200, gin.H{
		"data": items,
		"meta": PageMeta{Page: page, PerPage: perPage, Total: total},
	})
}
