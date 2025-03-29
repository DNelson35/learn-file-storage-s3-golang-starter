package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	maxMemory := 1 << 30
	http.MaxBytesReader(w, r.Body, int64(maxMemory))

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

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNoContent, "video data not found", err)
		return
	}
	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not Authorized", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	defer file.Close()
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	ext := strings.Split(mediaType, "/")[1]
	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create asset", err)
		return
	}
	defer os.Remove("tubely-upload.mp4")
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to move data", err)
		return
	}
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to move data", err)
		return
	}

	randBytes := make([]byte, 32)
	_, err = rand.Read(randBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create random bytes", err)
	}

	fileName := base64.RawURLEncoding.EncodeToString(randBytes)
	nameWithExt := fmt.Sprintf("%s.%s", fileName, ext)

	_ , err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &nameWithExt,
		Body: tmpFile,
		ContentType: &mediaType,
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to upload", err)
	}

	s3VidoeUrl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, nameWithExt)
	videoData.VideoURL = &s3VidoeUrl
	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "failed to update video data", err)
	}
	
	respondWithJSON(w, http.StatusOK, videoData)
}
