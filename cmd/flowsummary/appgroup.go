package flowsummary

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/brian1917/illumioapi"
	"github.com/brian1917/workloader/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var app, start, end string
var exclAllowed, exclPotentiallyBlocked, exclBlocked, appGroupLoc, ignoreIPGroup, consolidate, debug bool
var pce illumioapi.PCE
var err error

func init() {

	AppGroupFlowSummaryCmd.Flags().StringVarP(&app, "app", "a", "", "app name to limit Explorer results to flows with that app as a provider or a consumer. default is all apps.")
	AppGroupFlowSummaryCmd.Flags().StringVarP(&start, "start", "s", time.Date(time.Now().Year()-5, time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).Format("2006-01-02"), "start date in the format of yyyy-mm-dd. Date is set as midnight UTC.")
	AppGroupFlowSummaryCmd.Flags().StringVarP(&end, "end", "e", time.Now().Add(time.Hour*24).Format("2006-01-02"), "end date in the format of yyyy-mm-dd. Date is set as midnight UTC.")
	AppGroupFlowSummaryCmd.Flags().BoolVar(&exclAllowed, "excl-allowed", false, "excludes allowed traffic flows.")
	AppGroupFlowSummaryCmd.Flags().BoolVar(&exclPotentiallyBlocked, "excl-potentially-blocked", false, "excludes potentially blocked traffic flows.")
	AppGroupFlowSummaryCmd.Flags().BoolVar(&exclBlocked, "excl-blocked", false, "excludes blocked traffic flows.")
	AppGroupFlowSummaryCmd.Flags().BoolVarP(&appGroupLoc, "appgrp-loc", "l", false, "use location in app group")
	AppGroupFlowSummaryCmd.Flags().BoolVarP(&ignoreIPGroup, "ignore-ip", "i", false, "exlude IP address app groups from output")
	AppGroupFlowSummaryCmd.Flags().BoolVarP(&consolidate, "consolidate", "c", false, "consolidate all communication between 2 app groups into one CSV entry. See description below for example of output formats.")
	AppGroupFlowSummaryCmd.Flags().SortFlags = false

}

// AppGroupFlowSummaryCmd summarizes flows
var AppGroupFlowSummaryCmd = &cobra.Command{
	Use:   "appgroup",
	Short: "Summarize flows by port and protocol between app groups.",
	Long: `
Summarize flows by port and protocol between app groups.

Default output as each unique port/proto on a separet entry:
+------------------------------+------------------------------+-----------+---------------+---------------------------+---------------+
|        SRC APP GROUP         |        DST APP GROUP         |  SERVICE  | ALLOWED FLOWS | POTENTIALLY BLOCKED FLOWS | BLOCKED FLOWS |
+------------------------------+------------------------------+-----------+---------------+---------------------------+---------------+
| AssetManagement | Production | HREnrollment | Production    | 8070 TCP  |               | 30                        |               |
+------------------------------+------------------------------+-----------+---------------+---------------------------+---------------+
| AssetManagement | Production | HREnrollment | Production    | 3306 TCP  |               | 9                         |               |
+------------------------------+------------------------------+-----------+---------------+---------------------------+---------------+
| 45.54.45.54                  | eCommerce | Production       | 443 TCP   |               | 108                       |               |
+------------------------------+------------------------------+-----------+---------------+---------------------------+---------------+



Including the consolidate flag (--consolidate, -c) will combine all entries between two groups onto one line:
+------------------------------+------------------------------+----------------------+----------------------------------+----------------------+
|        SRC APP GROUP         |        DST APP GROUP         | ALLOWED FLOW SUMMARY | POTENTIALLY BLOCKED FLOW SUMMARY | BLOCKED FLOW SUMMARY |
+------------------------------+------------------------------+----------------------+----------------------------------+----------------------+
| AssetManagement | Production | HREnrollment | Production    |                      | 8070 TCP (30 flows);3306 TCP     |                      |
|                              |                              |                      | (9 flows)                        |                      |
+------------------------------+------------------------------+----------------------+----------------------------------+----------------------+
| 45.54.45.54                  | Point-of-Sale | Staging      |                      | 443 TCP (126 flows)              |                      |
+------------------------------+------------------------------+----------------------+----------------------------------+----------------------+
`,
	Run: func(cmd *cobra.Command, args []string) {

		pce, err = utils.GetPCE(true)
		if err != nil {
			utils.Log(1, err.Error())
		}

		// Get the debug value from viper
		debug = viper.Get("debug").(bool)

		flowSummary()
	},
}

// A summary consists of a common policy status, source app group, and destination app group.
type summary struct {
	policyStatus string
	srcAppGroup  string
	dstAppGroup  string
}

// A svcSummary consists of a port and protocol and flow count
type svcSummary struct {
	port  int
	proto string
	count int
}

func flowSummary() {

	// Build policy status slice
	var pStatus []string
	if !exclAllowed {
		pStatus = append(pStatus, "allowed")
	}
	if !exclPotentiallyBlocked {
		pStatus = append(pStatus, "potentially_blocked")
	}
	if !exclBlocked {
		pStatus = append(pStatus, "blocked")
	}

	// Get the state and end date
	startDate, err := time.Parse(fmt.Sprintf("2006-01-02 MST"), fmt.Sprintf("%s %s", start, "UTC"))
	if err != nil {
		utils.Log(1, err.Error())
	}
	startDate = startDate.In(time.UTC)

	endDate, err := time.Parse(fmt.Sprintf("2006-01-02 MST"), fmt.Sprintf("%s %s", end, "UTC"))
	if err != nil {
		utils.Log(1, err.Error())
	}
	endDate = endDate.In(time.UTC)

	// Create the default query struct
	tq := illumioapi.TrafficQuery{
		StartTime:      startDate,
		EndTime:        endDate,
		PolicyStatuses: pStatus,
		MaxFLows:       100000}

	// If an app is provided, adjust query to include it
	if app != "" {
		label, a, err := pce.GetLabelbyKeyValue("app", app)
		if debug {
			utils.LogAPIResp("GetLabelbyKeyValue", a)
		}
		if err != nil {
			utils.Log(1, fmt.Sprintf("getting label HREF - %s", err))
		}
		if label.Href == "" {
			utils.Log(1, fmt.Sprintf("%s does not exist as an app label.", app))
		}
		tq.SourcesInclude = []string{label.Href}
	}

	// Run traffic query
	traffic, a, err := pce.GetTrafficAnalysis(tq)
	if debug {
		utils.LogAPIResp("GetTrafficAnalysis", a)
	}
	if err != nil {
		utils.Log(1, fmt.Sprintf("making explorer API call - %s", err))
	}

	// If app is provided, switch to the destination include, clear the sources include, run query again, append to previous result
	if app != "" {
		tq.DestinationsInclude = tq.SourcesInclude
		tq.SourcesInclude = []string{}
		traffic2, a, err := pce.GetTrafficAnalysis(tq)
		if debug {
			utils.LogAPIResp("GetTrafficAnalysis", a)
		}
		if err != nil {
			utils.Log(1, fmt.Sprintf("making second explorer API call - %s", err))
		}
		traffic = append(traffic, traffic2...)
	}

	// Get the protocol list
	protoMap := illumioapi.ProtocolList()

	// Build the map of results
	entryMap := make(map[summary]map[svcSummary]int)

	// Cycle through the traffic results and build what we need
	for _, t := range traffic {
		var srcAppGroup, dstAppGroup string

		// Get src appgroup
		if t.Src.Workload == nil {
			if ignoreIPGroup {
				continue
			}
			srcAppGroup = t.Src.IP
		} else {
			srcAppGroup = t.Src.Workload.GetAppGroup(pce.LabelMapH)
			if appGroupLoc {
				srcAppGroup = t.Src.Workload.GetAppGroupL(pce.LabelMapH)
			}
		}

		// Get Dst appgroup
		if t.Dst.Workload == nil {
			if ignoreIPGroup {
				continue
			}
			dstAppGroup = t.Dst.IP
		} else {
			dstAppGroup = t.Dst.Workload.GetAppGroup(pce.LabelMapH)
			if appGroupLoc {
				dstAppGroup = t.Dst.Workload.GetAppGroupL(pce.LabelMapH)
			}
		}

		// Check if we already have this result captured. If we do, increment the flow counter
		entry := summary{srcAppGroup: srcAppGroup, dstAppGroup: dstAppGroup, policyStatus: t.PolicyDecision}
		if _, ok := entryMap[entry]; !ok {
			entryMap[entry] = make(map[svcSummary]int)
		}
		svc := svcSummary{port: t.ExpSrv.Port, proto: protoMap[t.ExpSrv.Proto]}
		//svc := fmt.Sprintf("%d %s", t.ExpSrv.Port, protoMap[t.ExpSrv.Proto])
		entryMap[entry][svc] = entryMap[entry][svc] + t.NumConnections
	}

	// Build the data slices
	data := [][]string{[]string{"src_app_group", "dst_app_group", "service", "allowed_flows", "potentially_blocked_flows", "blocked_flows"}}
	if consolidate {
		data = [][]string{[]string{"src_app_group", "dst_app_group", "allowed_flow_summary", "potentially_blocked_flow_summary", "blocked_flow_summary"}}
	}

	// Cylce through our entry map, add flows to struct, sort, create dataEntry if consolidate is set, append to data
	for e, v := range entryMap {
		x := []svcSummary{}
		var dataEntry []string
		for a, b := range v {
			a.count = b
			x = append(x, a)

		}
		sort.Slice(x, func(i, j int) bool {
			return x[i].count > x[j].count
		})
		for _, i := range x {
			if consolidate {
				dataEntry = append(dataEntry, fmt.Sprintf("%d %s (%d flows)", i.port, i.proto, i.count))
			}
		}
		if consolidate {
			switch e.policyStatus {
			case "allowed":
				data = append(data, []string{e.srcAppGroup, e.dstAppGroup, strings.Join(dataEntry, ";"), "", ""})
			case "potentially_blocked":
				data = append(data, []string{e.srcAppGroup, e.dstAppGroup, "", strings.Join(dataEntry, ";"), ""})
			case "blocked":
				data = append(data, []string{e.srcAppGroup, e.dstAppGroup, "", "", strings.Join(dataEntry, ";")})
			}
		} else {
			switch e.policyStatus {
			case "allowed":
				for _, a := range x {
					data = append(data, []string{e.srcAppGroup, e.dstAppGroup, fmt.Sprintf("%d %s", a.port, a.proto), fmt.Sprintf("%d", a.count), "", ""})
				}
			case "potentially_blocked":
				for _, a := range x {
					data = append(data, []string{e.srcAppGroup, e.dstAppGroup, fmt.Sprintf("%d %s", a.port, a.proto), "", fmt.Sprintf("%d", a.count), ""})
				}
			case "blocked":
				for _, a := range x {
					data = append(data, []string{e.srcAppGroup, e.dstAppGroup, fmt.Sprintf("%d %s", a.port, a.proto), "", "", fmt.Sprintf("%d", a.count)})
				}
			}
		}
	}

	// Write the data
	if len(data) > 1 {
		utils.WriteOutput(data, data, fmt.Sprintf("workloader-flowsummary-%s.csv", time.Now().Format("20060102_150405")))
		fmt.Printf("\r\n%d summaries exported.\r\n\r\n", len(data)-1)
		utils.Log(0, fmt.Sprintf("flowsummary complete - %d summaries exported", len(data)-1))
	} else {
		// Log command execution for 0 results
		fmt.Println("no explorer data to summarize")
		utils.Log(0, "no explorer data to summarize")
	}

}
