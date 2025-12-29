package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func UploadMiddleware(maxFileSizeMB int64, allowedTypes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			c.Abort()
			return
		}

		if file.Size == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is empty"})
			c.Abort()
			return
		}

		if file.Size > maxFileSizeMB*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file too large"})
			c.Abort()
			return
		}

		contentType := file.Header.Get("Content-Type")
		validType := false
		for _, t := range allowedTypes {
			if strings.HasPrefix(contentType, t) {
				validType = true
				break
			}
		}

		if !validType {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":        "invalid file type",
				"fileType":     contentType,
				"allowedTypes": allowedTypes,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
