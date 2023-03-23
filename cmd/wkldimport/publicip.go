package wkldimport

import (
	"fmt"

	"github.com/brian1917/illumioapi/v2"
	"github.com/brian1917/workloader/cmd/wkldexport"
	"github.com/brian1917/workloader/utils"
)

func (w *importWkld) publcIP(input Input) {
	if index, ok := input.Headers[wkldexport.HeaderPublicIP]; ok {
		if w.csvLine[index] != illumioapi.PtrToVal(w.wkld.PublicIP) {
			// Validate it first
			if !publicIPIsValid(w.csvLine[index]) {
				utils.LogError(fmt.Sprintf("csv line %d - invalid Public IP address format.", w.csvLineNum))
			}
			if w.wkld.Href != "" && input.UpdateWorkloads {
				w.change = true
				utils.LogInfo(fmt.Sprintf("csv line %d - %s- public ip to be changed from %s to %s", w.csvLineNum, w.compareString, utils.LogBlankValue(illumioapi.PtrToVal(w.wkld.PublicIP)), w.csvLine[index]), false)
			}
			w.wkld.PublicIP = &w.csvLine[index]
		}
	}
}
