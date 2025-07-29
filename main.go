package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	Host             string
	MaxFileSize      int64
	DefaultQuality   int64
	AllowedOrigin    string
	AllowedMimeTypes []string
}

func loadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	config := &Config{
		Port:           getEnv("PORT", "7070"),
		Host:           getEnv("HOST", "0.0.0.0"),
		MaxFileSize:    getEnvAsInt("MAX_FILE_SIZE", 10*1024*1024),
		DefaultQuality: getEnvAsInt("DEFAULT_QUALITY", 80),
		AllowedOrigin:  getEnv("ALLOW_ORIGIN", "http://localhost:8080"),
		AllowedMimeTypes: []string{
			"image/jpeg",
			"image/png",
			"image/jpg",
			"image/webp",
			"image/gif",
		},
	}

	return config
}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func detectMimeType(buffer []byte) string {
	return http.DetectContentType(buffer)
}

func isAllowedMimeType(mimeType string, allowedTypes []string) bool {
	normalizedType := strings.Split(mimeType, ";")[0]
	normalizedType = strings.TrimSpace(normalizedType)

	for _, allowed := range allowedTypes {
		if normalizedType == allowed {
			return true
		}
	}
	return false
}

func uploadImage(config *Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{config.AllowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.MaxMultipartMemory = config.MaxFileSize
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.POST("/compress", func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			log.Println("MultipartForm parse error:", err)
		} else {
			for key, files := range form.File {
				log.Printf("Got field: %s with %d file(s)\n", key, len(files))
			}
		}
		fileheader, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}
		file, err := fileheader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
			return
		}
		if fileheader.Size > config.MaxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large"})
			return
		}
		defer file.Close()

		buffer, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}

		detectedMimeType := detectMimeType(buffer)
		log.Printf("Detected MIME type: %s", detectedMimeType)

		if !isAllowedMimeType(detectedMimeType, config.AllowedMimeTypes) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         "Unsupported file type",
				"detected_type": detectedMimeType,
				"allowed_types": config.AllowedMimeTypes,
			})
			return
		}

		quality := int(config.DefaultQuality)
		if c.Request.FormValue("quality") != "" {
			quality, err = strconv.Atoi(c.Request.FormValue("quality"))
			if err != nil {
				panic(err)
			}
		}

		imgBuffer, err := imageProcessingToBuffer(buffer, quality)
		if err != nil {
			panic(err)
		}

		base64Image := base64.StdEncoding.EncodeToString(imgBuffer)

		c.JSON(200, gin.H{
			"image":     base64Image,
			"extension": "jpeg",
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

func imageProcessingToBuffer(buffer []byte, quality int) ([]byte, error) {
	reader := bytes.NewReader(buffer)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	processed := imaging.Resize(img, 800, 0, imaging.Lanczos)

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, processed, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

func main() {
	config := loadConfig()
	log.Printf("Starting server on %s:%s", config.Host, config.Port)
	log.Printf("Max file size: %d bytes", config.MaxFileSize)
	log.Printf("Default quality: %d", config.DefaultQuality)

	r := uploadImage(config)
	addr := config.Host + ":" + config.Port
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
