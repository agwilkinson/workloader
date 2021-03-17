package utils

import (
	"fmt"

	"github.com/brian1917/illumioapi"
	"github.com/spf13/viper"
)

// GetTargetPCE gets the target PCE for a command
func GetTargetPCE(GetLabelMaps bool) (illumioapi.PCE, error) {
	if viper.Get("target_pce") == nil || viper.Get("target_pce").(string) == "" {
		return GetDefaultPCE(GetLabelMaps)
	}

	return GetPCEbyName(viper.Get("target_pce").(string), GetLabelMaps)
}

// GetDefaultPCE reads the PCE information from the JSON generated by the login command
func GetDefaultPCE(GetLabelMaps bool) (illumioapi.PCE, error) {

	// Get the default PCE name
	if viper.Get("default_pce_name") == nil {
		LogError("There is not default pce. Either run workloader pce-add to add your first pce or workloader set-default to set an existing PCE as default.")
	}
	defaultPCE := viper.Get("default_pce_name").(string)

	var pce illumioapi.PCE
	if viper.IsSet(defaultPCE + ".fqdn") {
		pce = illumioapi.PCE{FQDN: viper.Get(defaultPCE + ".fqdn").(string), Port: viper.Get(defaultPCE + ".port").(int), Org: viper.Get(defaultPCE + ".org").(int), User: viper.Get(defaultPCE + ".user").(string), Key: viper.Get(defaultPCE + ".key").(string), DisableTLSChecking: viper.Get(defaultPCE + ".disableTLSChecking").(bool)}
		if GetLabelMaps {
			pce.Load(illumioapi.LoadInput{Labels: true})
		}
		return pce, nil
	}

	return illumioapi.PCE{}, fmt.Errorf("Could not retrieve PCE information - run workloader login command")
}

// GetPCEbyName gets a PCE by it's provided name
func GetPCEbyName(name string, GetLabelMaps bool) (illumioapi.PCE, error) {
	var pce illumioapi.PCE
	if viper.IsSet(name + ".fqdn") {
		pce = illumioapi.PCE{FQDN: viper.Get(name + ".fqdn").(string), Port: viper.Get(name + ".port").(int), Org: viper.Get(name + ".org").(int), User: viper.Get(name + ".user").(string), Key: viper.Get(name + ".key").(string), DisableTLSChecking: viper.Get(name + ".disableTLSChecking").(bool)}
		if GetLabelMaps {
			pce.Load(illumioapi.LoadInput{Labels: true})
		}
		return pce, nil
	}

	return illumioapi.PCE{}, fmt.Errorf("Could not retrieve %s PCE information", name)
}
