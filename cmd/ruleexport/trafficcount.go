package ruleexport

import (
	"fmt"
	"time"

	"github.com/brian1917/illumioapi"
	"github.com/brian1917/workloader/utils"
)

func trafficCounter(input Input, rs illumioapi.RuleSet, r illumioapi.Rule, counterStr string) ([]string, bool) {
	// Build the new explorer query object
	// Using the raw data structure for more flexibility versus illumioapi.TrafficQuery
	trafficReq := illumioapi.TrafficAnalysisRequest{MaxResults: input.ExplorerMax}

	// Build the holder consumer label slice
	var consumerLabels, providerLabels []illumioapi.Label

	// Scope exists - used to for checking AMS
	scopeExists := false

	// Scopes - iterate and fill provider slices. If unscoped consumers is false, also fill consumer slices. If it's a label group, expand it first.
	if rs.Scopes != nil {
		for _, scope := range *rs.Scopes {
			for _, scopeEntity := range scope {
				if scopeEntity.Label != nil {
					scopeExists = true
					providerLabels = append(providerLabels, input.PCE.Labels[scopeEntity.Label.Href])
					if !*r.UnscopedConsumers {
						consumerLabels = append(consumerLabels, input.PCE.Labels[scopeEntity.Label.Href])
					}
				}
				if scopeEntity.LabelGroup != nil {
					scopeExists = true
					labelHrefs := input.PCE.ExpandLabelGroup(scopeEntity.LabelGroup.Href)
					for _, labelHref := range labelHrefs {
						providerLabels = append(providerLabels, input.PCE.Labels[labelHref])
						if !*r.UnscopedConsumers {
							consumerLabels = append(consumerLabels, input.PCE.Labels[labelHref])
						}
					}
				}
			}
		}
	}

	// Consumers
	for _, consumer := range r.Consumers {
		// Add labels to slice
		if consumer.Label != nil {
			consumerLabels = append(consumerLabels, input.PCE.Labels[consumer.Label.Href])
		}
		// If it's a label group, expand it and add to the slice
		if consumer.LabelGroup != nil {
			labelHrefs := input.PCE.ExpandLabelGroup(consumer.LabelGroup.Href)
			for _, labelHref := range labelHrefs {
				consumerLabels = append(consumerLabels, input.PCE.Labels[labelHref])
			}
		}
		// Add IP lists directly to the traffic query
		if consumer.IPList != nil {
			trafficReq.Sources.Include = append(trafficReq.Sources.Include, []illumioapi.Include{{IPList: &illumioapi.IPList{Href: consumer.IPList.Href}}})
		}
		// Workload
		if consumer.Workload != nil {
			trafficReq.Sources.Include = append(trafficReq.Sources.Include, []illumioapi.Include{{Workload: &illumioapi.Workload{Href: consumer.Workload.Href}}})
		}
		// All workloads
		if consumer.Actors == "ams" && (!scopeExists || *r.UnscopedConsumers) {
			trafficReq.Sources.Include = append(trafficReq.Sources.Include, []illumioapi.Include{{Actors: "ams"}})
		}

	}

	// Providers
	for _, provider := range r.Providers {
		// Add labels to slice
		if provider.Label != nil {
			providerLabels = append(providerLabels, input.PCE.Labels[provider.Label.Href])
		}
		// If it's a label group, expand it and add to the slice
		if provider.LabelGroup != nil {
			labelHrefs := input.PCE.ExpandLabelGroup(provider.LabelGroup.Href)
			for _, labelHref := range labelHrefs {
				providerLabels = append(providerLabels, input.PCE.Labels[labelHref])
			}
		}
		// Add IP lists directly to the traffic query
		if provider.IPList != nil {
			trafficReq.Destinations.Include = append(trafficReq.Destinations.Include, []illumioapi.Include{{IPList: &illumioapi.IPList{Href: provider.IPList.Href}}})
		}
		// Workload
		if provider.Workload != nil {
			trafficReq.Destinations.Include = append(trafficReq.Destinations.Include, []illumioapi.Include{{Workload: &illumioapi.Workload{Href: provider.Workload.Href}}})
		}
		// All workloads
		if provider.Actors == "ams" && !scopeExists {
			trafficReq.Destinations.Include = append(trafficReq.Destinations.Include, []illumioapi.Include{{Actors: "ams"}})
		}
	}

	// Processes the consumer labels
	consumerLabelSets, err := illumioapi.LabelsToRuleStructure(consumerLabels)
	if err != nil {
		utils.LogError(err.Error())
	}
	for _, consumerLabelSet := range consumerLabelSets {
		inc := []illumioapi.Include{}
		for _, consumerLabel := range consumerLabelSet {
			inc = append(inc, illumioapi.Include{Label: &illumioapi.Label{Href: consumerLabel.Href}})
		}
		trafficReq.Sources.Include = append(trafficReq.Sources.Include, inc)
	}

	// Process the provider labels
	providerLabelSets, err := illumioapi.LabelsToRuleStructure(providerLabels)
	if err != nil {
		utils.LogError(err.Error())
	}
	for _, providerLabels := range providerLabelSets {
		inc := []illumioapi.Include{}
		for _, providerLabel := range providerLabels {
			inc = append(inc, illumioapi.Include{Label: &illumioapi.Label{Href: providerLabel.Href}})
		}
		trafficReq.Destinations.Include = append(trafficReq.Destinations.Include, inc)
	}

	// Check we have a valid rule
	if len(trafficReq.Sources.Include) == 0 {
		utils.LogWarning(fmt.Sprintf("rule %s - %s - ruleset %s - does not have valid consumers for explorer query: labels, label groups, workloads, ip lists, or all workloads. skipping.", counterStr, r.Href, rs.Name), true)
		return []string{"invalid rule for querying traffic", "invalid rule for querying traffic", "invalid rule for querying traffic", "invalid rule for querying traffic", "invalid rule for querying traffic"}, true
	}
	if len(trafficReq.Destinations.Include) == 0 {
		utils.LogWarning(fmt.Sprintf("rule %s - %s - ruleset %s - does not have valid providers for explorer query: labels, label groups, workloads, ip lists, or all workloads. skipping.", counterStr, r.Href, rs.Name), true)
		return []string{"invalid rule for querying traffic", "", "", "", ""}, true
	}
	if r.ConsumingSecurityPrincipals != nil && len(r.ConsumingSecurityPrincipals) > 0 {
		utils.LogWarning(fmt.Sprintf("rule %s - ruleset %s - ad user groups not considered in traffic queries. %s", counterStr, rs.Name, r.Href), true)

	}

	// Parse services
	// Create the array
	for _, ingressService := range *r.IngressServices {
		// Process the policy services
		if ingressService.Href != nil && *ingressService.Href != "" {
			svc := input.PCE.Services[*ingressService.Href]
			includes, _ := svc.ToExplorer()
			if len(includes) == 0 {
				trafficReq.ExplorerServices.Include = make([]illumioapi.Include, 0)
			} else {
				trafficReq.ExplorerServices.Include = append(trafficReq.ExplorerServices.Include, includes...)
			}
			// Process port ranges
		} else if ingressService.Port != nil && ingressService.ToPort != nil {
			trafficReq.ExplorerServices.Include = append(trafficReq.ExplorerServices.Include, illumioapi.Include{Port: *ingressService.Port, ToPort: *ingressService.ToPort})
			// Process ports
		} else if ingressService.Port != nil && ingressService.ToPort == nil {
			trafficReq.ExplorerServices.Include = append(trafficReq.ExplorerServices.Include, illumioapi.Include{Port: *ingressService.Port})
		} else {
			trafficReq.ExplorerServices.Include = make([]illumioapi.Include, 0)
		}
	}

	if len(*r.IngressServices) == 0 {
		trafficReq.ExplorerServices.Include = make([]illumioapi.Include, 0)
	}

	// Create empty arrays for JSON marshalling for parameters we don't need.
	trafficReq.Sources.Exclude = make([]illumioapi.Exclude, 0)
	trafficReq.Destinations.Exclude = make([]illumioapi.Exclude, 0)
	trafficReq.ExplorerServices.Exclude = make([]illumioapi.Exclude, 0)
	trafficReq.PolicyDecisions = []string{}
	_, api, err := input.PCE.GetVersion()
	utils.LogAPIResp("GetVersion", api)
	if err != nil {
		utils.LogError(err.Error())
	}
	input.PCE.GetVersion()
	if input.PCE.Version.Major > 19 {
		x := false
		trafficReq.ExcludeWorkloadsFromIPListQuery = &x
	}

	// Get the start date
	t, err := time.Parse("2006-01-02 MST", input.ExplorerStart+" UTC")
	if err != nil {
		utils.LogError(err.Error())
	}
	trafficReq.StartDate = t.In(time.UTC)
	// Get the end date
	t, err = time.Parse("2006-01-02 MST", input.ExplorerEnd+" UTC")
	if err != nil {
		utils.LogError(err.Error())
	}
	trafficReq.EndDate = t.In(time.UTC)

	// Give it a name
	name := "workloader-rule-usage-" + r.Href
	trafficReq.QueryName = &name

	// Make the traffic request
	utils.LogInfo(fmt.Sprintf("rule %s - ruleset %s - creating async explorer query for %s", counterStr, rs.Name, r.Href), true)
	asyncTrafficQuery, a, err := input.PCE.CreateAsyncTrafficRequest(trafficReq)
	utils.LogAPIResp("GetTrafficAnalysisAPI", a)
	if err != nil {
		utils.LogError(err.Error())
	}

	return []string{asyncTrafficQuery.Href, "", "", "", a.ReqBody}, false

}
