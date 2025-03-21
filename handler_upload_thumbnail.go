package main

import (
	"fmt"
	"io"
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

	const maxMemory = 10 << 20 
	r.ParseMultipartForm(maxMemory)

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	defer file.Close()
	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNoContent, "video data not found", err)
		return
	}
	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not Authorized", err)
		return
	}

	ext := strings.Split(mediaType, "/")[1]
	imageFileName :=	fmt.Sprintf("%s.%s", videoData.ID, ext)
	imageFilePath := filepath.Join(cfg.assetsRoot, imageFileName)
	imageFile, err := os.Create(imageFilePath)
	if err != nil {
		fmt.Println(imageFilePath)
		respondWithError(w, http.StatusInternalServerError, "failed to create asset", err)
		return
	}
	defer imageFile.Close()

	_, err = io.Copy(imageFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to move data", err)
		return
	}

	imgPath := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, imageFileName)
	videoData.ThumbnailURL = &imgPath
	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
