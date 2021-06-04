package google

import (
	"fmt"

	"github.com/infracost/infracost/internal/schema"
	"github.com/shopspring/decimal"
)

type EgressResourceType int

const (
	StorageBucketEgress EgressResourceType = iota
	ContainerRegistryEgress
	ComputeExternalVPNGateway
)

type egressRegionData struct {
	gRegion        string
	apiDescription string
	usageKey       string
}

type egressRegionUsageFilterData struct {
	usageNumber int64
	usageName   string
}

func networkEgress(region string, u *schema.UsageData, resourceName, prefixName string, egressResourceType EgressResourceType) *schema.Resource {
	resource := &schema.Resource{
		Name:           resourceName,
		CostComponents: []*schema.CostComponent{},
	}

	// Same continent
	if doesEgressIncludeSameContinent(egressResourceType) {
		var quantity *decimal.Decimal
		if u != nil && u.Get("monthly_egress_data_transfer_gb.same_continent").Exists() {
			quantity = decimalPtr(decimal.NewFromInt(u.Get("monthly_egress_data_transfer_gb.same_continent").Int()))
		}
		resource.CostComponents = append(resource.CostComponents, &schema.CostComponent{
			Name:            fmt.Sprintf("%s in same continent", prefixName),
			Unit:            "GB",
			UnitMultiplier:  1,
			MonthlyQuantity: quantity,
			ProductFilter: &schema.ProductFilter{
				VendorName: strPtr("gcp"),
				Region:     strPtr("global"),
				Service:    strPtr("Cloud Storage"),
				AttributeFilters: []*schema.AttributeFilter{
					{Key: "description", Value: strPtr("Inter-region GCP Storage egress within EU")},
				},
			},
		})
	}

	regionsData := getEgressRegionsData(prefixName, egressResourceType)
	usageFiltersData := getEgressUsageFiltersData(egressResourceType)

	for _, regData := range regionsData {
		gRegion := regData.gRegion
		apiDescription := regData.apiDescription
		usageKey := regData.usageKey

		// TODO: Reformat it to use tier helpers.
		var usage int64
		var used int64
		var lastEndUsageAmount int64
		if u != nil && u.Get(usageKey).Exists() {
			usage = u.Get(usageKey).Int()
		}

		for idx, usageFilter := range usageFiltersData {
			usageName := usageFilter.usageName
			endUsageAmount := usageFilter.usageNumber
			var quantity *decimal.Decimal
			if endUsageAmount != 0 && usage >= endUsageAmount {
				used = endUsageAmount - used
				lastEndUsageAmount = endUsageAmount
				quantity = decimalPtr(decimal.NewFromInt(used))
			} else if usage > lastEndUsageAmount {
				used = usage - lastEndUsageAmount
				lastEndUsageAmount = endUsageAmount
				quantity = decimalPtr(decimal.NewFromInt(used))
			}
			var usageFilter string
			if endUsageAmount != 0 {
				usageFilter = fmt.Sprint(endUsageAmount)
			} else {
				usageFilter = ""
			}
			if quantity == nil && idx > 0 {
				continue
			}
			resource.CostComponents = append(resource.CostComponents, &schema.CostComponent{
				Name:            fmt.Sprintf("%v (%v)", gRegion, usageName),
				Unit:            "GB",
				UnitMultiplier:  1,
				MonthlyQuantity: quantity,
				ProductFilter: &schema.ProductFilter{
					Region:     getEgressAPIRegionName(region, egressResourceType),
					VendorName: strPtr("gcp"),
					Service:    getEgressAPIServiceName(egressResourceType),
					AttributeFilters: []*schema.AttributeFilter{
						{Key: "description", Value: strPtr(apiDescription)},
					},
				},
				PriceFilter: &schema.PriceFilter{
					EndUsageAmount: strPtr(usageFilter),
				},
			})
		}
	}

	return resource
}

func doesEgressIncludeSameContinent(egressResourceType EgressResourceType) bool {
	switch egressResourceType {
	case ComputeExternalVPNGateway:
		return false
	default:
		return true
	}
}

func getEgressRegionsData(prefixName string, egressResourceType EgressResourceType) []*egressRegionData {
	switch egressResourceType {
	case ComputeExternalVPNGateway:
		return []*egressRegionData{
			{
				gRegion: fmt.Sprintf("%s to worldwide excluding China, Australia but including Hong Kong", prefixName),
				// There is no worldwide option in APIs, so we take a random region.
				apiDescription: "Network Vpn Internet Egress from Americas to Western Europe",
				usageKey:       "monthly_egress_data_transfer_gb.worldwide",
			},
			{
				gRegion:        fmt.Sprintf("%s to China excluding Hong Kong", prefixName),
				apiDescription: "Network Vpn Internet Egress from Americas to China",
				usageKey:       "monthly_egress_data_transfer_gb.china",
			},
			{
				gRegion:        fmt.Sprintf("%s to Australia", prefixName),
				apiDescription: "Network Vpn Internet Egress from Americas to Australia",
				usageKey:       "monthly_egress_data_transfer_gb.australia",
			},
		}
	default:
		return []*egressRegionData{
			{
				gRegion:        fmt.Sprintf("%s to worldwide excluding Asia, Australia", prefixName),
				apiDescription: "Download Worldwide Destinations (excluding Asia & Australia)",
				usageKey:       "monthly_egress_data_transfer_gb.worldwide",
			},
			{
				gRegion:        fmt.Sprintf("%s to Asia excluding China, but including Hong Kong", prefixName),
				apiDescription: "Download APAC",
				usageKey:       "monthly_egress_data_transfer_gb.asia",
			},
			{
				gRegion:        fmt.Sprintf("%s to China excluding Hong Kong", prefixName),
				apiDescription: "Download China",
				usageKey:       "monthly_egress_data_transfer_gb.china",
			},
			{
				gRegion:        fmt.Sprintf("%s to Australia", prefixName),
				apiDescription: "Download Australia",
				usageKey:       "monthly_egress_data_transfer_gb.australia",
			},
		}
	}
}

func getEgressUsageFiltersData(egressResourceType EgressResourceType) []*egressRegionUsageFilterData {
	usageFiltersData := []*egressRegionUsageFilterData{
		{
			usageName:   "first 1TB",
			usageNumber: 1024,
		},
		{
			usageName:   "next 9TB",
			usageNumber: 10240,
		},
		{
			usageName:   "over 10TB",
			usageNumber: 0,
		},
	}
	return usageFiltersData
}

func getEgressAPIRegionName(region string, egressResourceType EgressResourceType) *string {
	switch egressResourceType {
	case ComputeExternalVPNGateway:
		return strPtr(region)
	default:
		return nil
	}
}

func getEgressAPIServiceName(egressResourceType EgressResourceType) *string {
	switch egressResourceType {
	case ComputeExternalVPNGateway:
		return strPtr("Compute Engine")
	default:
		return strPtr("Cloud Storage")
	}
}