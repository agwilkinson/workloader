package subnet

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/brian1917/illumioapi"
)

// A subnet is extracted from the CSV and has an assoicated location and environment
type subnet struct {
	network net.IPNet
	loc     string
	env     string
}

// used to parse subnet to environment and location labels
func locParser(csvFile string, netCol, envCol, locCol int) []subnet {
	var results []subnet

	// Open CSV File
	file, err := os.Open(csvFile)
	if err != nil {
		log.Fatalf("Error opening CSV - %s", err)
	}
	defer file.Close()
	reader := csv.NewReader(bufio.NewReader(file))

	// Start the counter
	i := 0

	// Iterate through CSV entries
	for {

		// Increment the counter
		i++

		// Read the line
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("Error - reading CSV file - %s", err)
		}

		// Skipe the header row
		if i == 1 {
			continue
		}

		//make sure location label not empty
		if line[locCol] == "" {
			log.Fatal("Error - Label field cannot be empty")
		}

		//Place subnet into net.IPNet data structure as part of subnetLabel struct
		_, network, err := net.ParseCIDR(line[netCol])
		if err != nil {
			log.Fatal("Error - The Subnet field cannot be parsed.  The format is 10.10.10.0/24")
		}

		//Set struct values
		results = append(results, subnet{network: *network, env: line[envCol], loc: line[locCol]})
	}
	return results
}

func csvWriter(pce illumioapi.PCE, matches []match) {
	// Get all the labels again so we have a map
	labels, _, err := illumioapi.GetAllLabels(pce)
	if err != nil {
		log.Fatalf("ERROR - Getting all labes in - %s", err)
	}
	labelMap := make(map[string]illumioapi.Label)
	for _, l := range labels {
		labelMap[l.Href] = l
	}

	// Get time stamp for output files
	timestamp := time.Now().Format("20060102_150405")

	// Always create the default file
	outputFile, err := os.Create("subnet-output-" + timestamp + ".csv")
	if err != nil {
		log.Fatalf("ERROR - Creating file - %s\n", err)
	}
	defer outputFile.Close()

	fmt.Fprintf(outputFile, "hostname,ip_address,original_loc,original_env,new_loc,new_env\r\n")

	for _, m := range matches {
		// Update the workload label
		m.workload.RefreshLabels(labelMap)
		fmt.Fprintf(outputFile, "%s,%s,%s,%s,%s,%s\r\n", m.workload.Hostname, m.workload.Interfaces[0].Address, m.oldLoc, m.oldEnv, m.workload.Loc.Value, m.workload.Env.Value)
	}

}
