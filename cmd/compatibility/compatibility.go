package compatibility

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/brian1917/illumioapi"
	"github.com/brian1917/workloader/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var debug, modeChangeInput, issuesOnly bool
var pce illumioapi.PCE
var outputFileName string
var err error

func init() {
	CompatibilityCmd.Flags().BoolVarP(&modeChangeInput, "mode-input", "m", false, "generate the input file to change all idle workloads to build using workloader mode command")
	CompatibilityCmd.Flags().BoolVarP(&issuesOnly, "issues-only", "i", false, "only export compatibility checks with an issue")
	CompatibilityCmd.Flags().StringVar(&outputFileName, "output-file", "", "optionally specify the name of the output file location. default is current location with a timestamped filename.")
}

// CompatibilityCmd runs the workload identifier
var CompatibilityCmd = &cobra.Command{
	Use:   "compatibility",
	Short: "Generate a compatibility report for all Idle workloads.",
	Long: `
Generate a compatibility report for all Idle workloads.

The update-pce and --no-prompt flags are ignored for this command.`,
	Run: func(cmd *cobra.Command, args []string) {

		pce, err = utils.GetTargetPCE(false)
		if err != nil {
			utils.LogError(err.Error())
		}

		// Get the debug value from viper
		debug = viper.Get("debug").(bool)

		compatibilityReport()
	},
}

func compatibilityReport() {

	// Log command
	utils.LogStartCommand("compatibility")

	// Start the data slice with the headers. We will append data to this.
	var csvData, stdOutData, modeChangeInputData [][]string
	csvData = append(csvData, []string{"hostname", "href", "status", "raw_data"})
	stdOutData = append(stdOutData, []string{"hostname", "href", "status"})
	modeChangeInputData = append(modeChangeInputData, []string{"href", "mode"})

	// Get all idle  workloads
	qp := map[string]string{"mode": "idle"}
	wklds, a, err := pce.GetAllWorkloadsQP(qp)
	if debug {
		utils.LogAPIResp("GetAllWorkloadsH", a)
	}
	if err != nil {
		utils.LogError(err.Error())
	}

	// Get Idle workload count
	idleWklds := []illumioapi.Workload{}
	for _, w := range wklds {
		if w.Agent.Config.Mode == "idle" {
			idleWklds = append(idleWklds, w)
		}
	}

	// Iterate through each workload
	for i, w := range idleWklds {

		// Get the compatibility report and append
		cr, a, err := pce.GetCompatibilityReport(w)
		if debug {
			utils.LogAPIResp("GetCompatibilityReport", a)
		}
		if err != nil {
			utils.LogError(fmt.Sprintf("getting compatibility report for %s (%s) - %s", w.Hostname, w.Href, err))
		}

		// Put into slice if it's NOT green and issuesOnly is true
		if (cr.QualifyStatus != "green" && issuesOnly) || !issuesOnly {
			csvData = append(csvData, []string{w.Hostname, w.Href, cr.QualifyStatus, a.RespBody})
			stdOutData = append(stdOutData, []string{w.Hostname, w.Href, cr.QualifyStatus})
		}

		if cr.QualifyStatus == "green" {
			modeChangeInputData = append(modeChangeInputData, []string{w.Href, "build"})
		}

		// Update stdout
		end := ""
		if i+1 == len(idleWklds) {
			end = "\r\n"
		}
		fmt.Printf("\r[INFO] - Exported %d of %d idle workloads (%d%%).%s", i+1, len(wklds), (i+1)*100/len(wklds), end)
	}

	// Print a line at the end of our counter

	// If the CSV data has more than just the headers, create output file and write it.
	if len(csvData) > 1 {

		utils.WriteOutput(csvData, stdOutData, fmt.Sprintf("workloader-compatibility-%s.csv", time.Now().Format("20060102_150405")))
		utils.LogInfo(fmt.Sprintf("%d compatibility reports exported.", len(csvData)-1), true)
	} else {
		// Log command execution for 0 results
		utils.LogInfo("no workloads in idle mode.", true)
	}

	// Write the mode change CSV
	if modeChangeInput && len(modeChangeInputData) > 1 {
		// Create CSV
		if outputFileName == "" {
			outputFileName = fmt.Sprintf("workloader-mode-input-%s.csv", time.Now().Format("20060102_150405"))
		}
		outFile, err := os.Create(outputFileName)
		if err != nil {
			utils.LogError(fmt.Sprintf("creating CSV - %s\n", err))
		}

		// Write CSV data
		writer := csv.NewWriter(outFile)
		writer.WriteAll(modeChangeInputData)
		if err := writer.Error(); err != nil {
			utils.LogError(fmt.Sprintf("writing CSV - %s\n", err))
		}
		// Log
		utils.LogInfo(fmt.Sprintf("Created a file to be used with workloader mode command to change all green status IDLE workloads to build: %s", outFile.Name()), true)
	}
	utils.LogEndCommand("compatibility")

}
