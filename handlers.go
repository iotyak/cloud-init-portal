package main

import "sort"

type Server struct {
	Store             *Store
	Templates         map[string]CloudInitTemplate
	BoxTypes          map[string]BoxType
	Logger            *ProvisionLogger
	PublicBaseURL     string
	TrustProxyHeaders bool
	StatusLimiter     *fixedWindowLimiter
	WriteLimiter      *fixedWindowLimiter
}

type indexData struct {
	TemplateNames []string
	BoxTypes      []BoxType
	Current       *ActiveConfig
	Status        ProvisionStatus
	Error         string
	Message       string
	Success       *successData
}

type successData struct {
	Hostname     string
	TemplateName string
	BoxTypeName  string
	UserDataURL  string
	MetaDataURL  string
	IPXEExample  string
	CurlExample  string
}

type statusPayload struct {
	Hostname     string `json:"hostname"`
	StaticIP     string `json:"static_ip"`
	TemplateName string `json:"template_name"`
	BoxTypeName  string `json:"box_type"`
	Status       string `json:"status"`
	GeneratedAt  string `json:"generated_at"`
	Active       bool   `json:"active"`
}

func sortedBoxTypes(boxTypes map[string]BoxType) []BoxType {
	names := BoxTypeNames(boxTypes)
	out := make([]BoxType, 0, len(names))
	for _, n := range names {
		out = append(out, boxTypes[n])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
