package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/brian1917/illumioapi"
)

func main() {
	fqdn := flag.String("fqdn", "", "The fully qualified domain name of the PCE.")
	port := flag.Int("port", 8443, "The port for the PCE.")
	user := flag.String("user", "", "API user or email address.")
	pwd := flag.String("pwd", "", "API key if using API user or password if using email address.")
	iplistName := flag.String("ipl", "LocationMap", "IP List name to use as a reference")
	disableTLS := flag.Bool("x", false, "Disable TLS checking.")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println("-fqdn  string")
		fmt.Println("       The fully qualified domain name of the PCE. Required.")
		fmt.Println("-port  int")
		fmt.Println("       The port of the PCE. (default 8443)")
		fmt.Println("-user  string")
		fmt.Println("       API user or email address. Required.")
		fmt.Println("-pwd   string")
		fmt.Println("       API key if using API user or password if using email address. Required.")
		fmt.Println("-ipl   string")
		fmt.Println("       IP List name to use as a reference. (default LocationMap)")
		fmt.Println("-x     Disable TLS checking.")
	}

	// Parse flags
	flag.Parse()

	// Run some checks on the required fields
	if len(*fqdn) == 0 || len(*user) == 0 || len(*pwd) == 0 {
		log.Fatalf("ERROR - Required arguments not included. Run -h for usgae.")
	}

	// Build the PCE Object
	pce, err := illumioapi.PCEbuilder(*fqdn, *user, *pwd, *port, *disableTLS)
	if err != nil {
		log.Fatalf("ERROR - Building PCE - %s", err)
	}

	// Get our IP List
	ipl, _, err := illumioapi.GetIPList(pce, *iplistName)
	subnets := make(map[string]string)
	for _, i := range ipl.IPRanges {
		subnets[i.FromIP] = i.Description
	}

	// Get all labels
	labels, _, err := illumioapi.GetAllLabels(pce)
	if err != nil {
		log.Fatalf("Error - Getting labels - %s", err)
	}
	labelKeys := make(map[string]string)
	labelValues := make(map[string]string)
	for _, l := range labels {
		labelKeys[l.Href] = l.Key
		labelValues[l.Href] = l.Value
	}

	// Get all workloads
	wls, _, err := illumioapi.GetAllWorkloads(pce)
	if err != nil {
		log.Fatalf("Error getting all workloads - %s", err)
	}

	// Cycle through each workload
	for _, wl := range wls {
		// Cycle through each subnet from IP List
		for snet, loc := range subnets {
			// Check if the workload interface is in the subnet
			_, ipv4Net, _ := net.ParseCIDR(snet)
			if ipv4Net.Contains(net.ParseIP(wl.Interfaces[0].Address)) {
				// Cycle through the workloads labels
				for _, l := range wl.Labels {
					// If it's the location and doesn't match what it should be, update it
					if labelKeys[l.Href] == "loc" {
						if labelValues[l.Href] != loc {
							if err := wl.UpdateLabel(pce, "loc", loc); err != nil {
								log.Fatalf("Error updating workload struct - %s", err)
							}
							_, err = illumioapi.UpdateWorkload(pce, wl)
							if err != nil {
								log.Fatalf("Error updating workload - %s", err)
							}
							log.Printf("Updated the location of %s with IP Address %s to %s", wl.Hostname, wl.Interfaces[0].Address, loc)
						}

					}
				}
			}
		}
	}
}
