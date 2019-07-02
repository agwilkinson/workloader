package cmd

import (
	"log"

	"github.com/brian1917/workloader/utils"
	"github.com/spf13/cobra"
)

// TrafficCmd runs the workload identifier
var flowCmd = &cobra.Command{
	Use:   "flowupload",
	Short: "Upload flows to PCE from a CSV file.",
	Run: func(cmd *cobra.Command, args []string) {

		_, err := utils.GetPCE("pce.json")
		if err != nil {
			log.Fatalf("Error getting PCE for traffic command - %s", err)
		}
	},
}
