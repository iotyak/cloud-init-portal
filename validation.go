package main

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var hostnameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,62}$`)

func validateInput(templateName, boxTypeName, hostname, staticIP, cidr, gateway string, dns []string, templates map[string]CloudInitTemplate, boxTypes map[string]BoxType) error {
	templateName = strings.TrimSpace(templateName)
	boxTypeName = strings.TrimSpace(boxTypeName)
	hostname = strings.TrimSpace(hostname)
	staticIP = strings.TrimSpace(staticIP)
	cidr = strings.TrimSpace(cidr)
	gateway = strings.TrimSpace(gateway)

	if _, ok := templates[templateName]; !ok {
		return errors.New("unknown template")
	}
	if _, ok := boxTypes[boxTypeName]; !ok {
		return errors.New("unknown box type")
	}
	if !hostnameRe.MatchString(hostname) {
		return errors.New("invalid hostname (use letters, numbers, dash; max 63 chars)")
	}
	if ip := net.ParseIP(staticIP); ip == nil {
		return errors.New("invalid static IP")
	}
	n, err := strconv.Atoi(cidr)
	if err != nil || n < 1 || n > 32 {
		return errors.New("invalid CIDR (expected 1-32)")
	}
	if gateway != "" {
		if ip := net.ParseIP(gateway); ip == nil {
			return errors.New("invalid gateway IP")
		}
	}
	for _, dnsIP := range dns {
		trimmed := strings.TrimSpace(dnsIP)
		if ip := net.ParseIP(trimmed); ip == nil {
			return fmt.Errorf("invalid DNS server IP: %s", dnsIP)
		}
	}
	return nil
}
