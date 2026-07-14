package service

import (
	"log"
	"mime/multipart"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

var cld *cloudinary.Cloudinary

// InitCloudinary initializes Cloudinary once from the env URL.
func InitCloudinary(cloudURL string) {
	var err error
	cld, err = cloudinary.NewFromURL(cloudURL)
	if err != nil {
		log.Fatalf("Cloudinary init error: %v", err)
	}
}

// UploadImageToCloudinary uploads a file and returns the secure URL.
func UploadImageToCloudinary(file multipart.File, filename string) (string, error) {
	uploadResp, err := cld.Upload.Upload(db.Ctx, file, uploader.UploadParams{
		Folder:   "books",     // optional folder in Cloudinary
		PublicID: filename,    // use filename as Cloudinary public_id
	})
	if err != nil {
		return "", err
	}

	return uploadResp.SecureURL, nil
}
func UploadProfileImgToCloudinary(file multipart.File, filename string) (string, string, error) {
	uploadResp, err := cld.Upload.Upload(db.Ctx, file, uploader.UploadParams{
		Folder:   "profile_pictures",
		PublicID: filename,
	})
	if err != nil {
		return "", "", err
	}

	return uploadResp.SecureURL, uploadResp.PublicID, nil
}
func DeleteImageFromCloudinary(publicID string) error {
	// Must call Destroy (Upload API) to delete a single asset
	_, err := cld.Upload.Destroy(db.Ctx, uploader.DestroyParams{
		PublicID: publicID,
	})

	return err
}