package utils

import (
	"fmt"

	"github.com/brian1917/illumioapi"
	"github.com/spf13/viper"
)

// GetPCE reads the PCE information from the JSON generated by the login command
func GetPCE(GetLabelMaps bool) (illumioapi.PCE, error) {
	var pce illumioapi.PCE
	if viper.IsSet("fqdn") {
		pce = illumioapi.PCE{FQDN: viper.Get("fqdn").(string), Port: viper.Get("port").(int), Org: viper.Get("org").(int), User: viper.Get("user").(string), Key: viper.Get("key").(string), DisableTLSChecking: viper.Get("disableTLSChecking").(bool)}
		if GetLabelMaps {
			pce.GetLabelMaps()
		}
		return pce, nil
	}

	return illumioapi.PCE{}, fmt.Errorf("Could not retrieve PCE information - run workloader login command")
}
