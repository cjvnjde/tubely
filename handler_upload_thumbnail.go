package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	fileType := header.Header.Get("Content-Type")

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "No access", err)
		return
	}

	mediaType, _, err := mime.ParseMediaType(fileType)

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Incorrect file type", err)
		return
	}

	d := strings.Split(fileType, "/")
	fileExtension := d[1]
	if fileExtension == "" {

		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	randomBytes := make([]byte, 32)
	_, err = rand.Read(randomBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	urlEncoding := base64.RawURLEncoding.EncodeToString(randomBytes)

	fileName := fmt.Sprintf("%s.%s", urlEncoding, fileExtension)
	assetPath := filepath.Join(cfg.assetsRoot, fileName)
	assetFile, err := os.Create(assetPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}
	_, err = io.Copy(assetFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileName)
	video.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
