package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Upload images to Basecamp",
}

var (
	imageFileFlag    string
	imageCaptionFlag string
)

var imagesUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload an image file, returns sgid and HTML snippet",
	RunE: func(cmd *cobra.Command, args []string) error {
		if imageFileFlag == "" {
			return fmt.Errorf("--file is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
		result, err := client.UploadImage(imageFileFlag, imageCaptionFlag, getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("SGID: %s\n\nHTML snippet:\n%s\n", result.SGID, result.HTML)
		}
		return nil
	},
}

func init() {
	imagesUploadCmd.Flags().StringVar(&imageFileFlag, "file", "", "Path to image file (required)")
	imagesUploadCmd.Flags().StringVar(&imageCaptionFlag, "caption", "Image", "Caption for the image")

	imagesCmd.AddCommand(imagesUploadCmd)
	rootCmd.AddCommand(imagesCmd)
}
